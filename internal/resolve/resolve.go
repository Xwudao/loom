// Package resolve turns a parsed graph into a topologically ordered build
// plan. Resolution is memoized, detects cycles, and reports missing or
// ambiguous providers with a full dependency path.
package resolve

import (
	"go/token"
	"go/types"

	"github.com/Xwudao/loom/internal/diag"
	"github.com/Xwudao/loom/internal/model"

	"golang.org/x/tools/go/types/typeutil"
)

// StepKind identifies what a step produces.
type StepKind uint8

const (
	// StepProvider invokes a registered provider.
	StepProvider StepKind = iota
	// StepLifecycle is the *loom.Lifecycle created by generated code.
	StepLifecycle
	// StepContext is the context.Context passed to generated code.
	StepContext
)

// Step is one node of an ordered build plan.
type Step struct {
	Kind     StepKind
	Provider *model.Provider
	// Type is the type this step produces.
	Type types.Type
	// Deps are the steps that must run before this one, in input order.
	Deps []*Step
	// Pos is the source position of the provider, if any.
	Pos token.Position
}

// Plan is the resolved build order for a graph.
type Plan struct {
	Graph *model.Graph
	Root  *Step
	// Steps are ordered so that every dependency precedes its dependents.
	Steps []*Step

	byType typeutil.Map // types.Type -> *Step
}

// StepFor returns the step that satisfies t.
func (p *Plan) StepFor(t types.Type) (*Step, bool) {
	if s := p.byType.At(t); s != nil {
		return s.(*Step), true
	}
	return nil, false
}

// HasCleanup reports whether any provider in the plan returns a cleanup
// function. Generated code only emits rollback logic when it might be needed.
func (p *Plan) HasCleanup() bool {
	for _, s := range p.Steps {
		if s.Kind == StepProvider && s.Provider.Cleanup != nil {
			return true
		}
	}
	return false
}

// HasError reports whether any provider in the plan can fail.
func (p *Plan) HasError() bool {
	for _, s := range p.Steps {
		if s.Kind == StepProvider && s.Provider.HasError {
			return true
		}
	}
	return false
}

type frame struct {
	typ      types.Type
	provider *model.Provider
}

type builder struct {
	g         *model.Graph
	base      string
	index     model.Index
	lifecycle types.Type
	context   types.Type

	steps      []*Step
	byType     typeutil.Map
	done       map[*model.Provider]*Step
	inProgress map[*model.Provider]bool
}

// Build resolves g into a plan. lifecycle and contextType are the framework
// types that are always injectable; either may be nil. base is used to render
// source paths in diagnostics.
func Build(g *model.Graph, lifecycle, contextType types.Type, base string) (*Plan, error) {
	b := &builder{
		g:          g,
		base:       base,
		lifecycle:  lifecycle,
		context:    contextType,
		done:       map[*model.Provider]*Step{},
		inProgress: map[*model.Provider]bool{},
	}
	for _, p := range g.Providers {
		b.index.Add(p.Output, p)
		if p.Binding != nil {
			b.index.Add(p.Binding, p)
		}
	}

	root, err := b.resolve(g.Target, nil)
	if err != nil {
		return nil, err
	}
	return &Plan{Graph: g, Root: root, Steps: b.steps, byType: b.byType}, nil
}

func (b *builder) resolve(t types.Type, stack []frame) (*Step, error) {
	if s := b.byType.At(t); s != nil {
		return s.(*Step), nil
	}

	if b.lifecycle != nil && types.Identical(t, b.lifecycle) {
		s := &Step{Kind: StepLifecycle, Type: t}
		b.memoize(t, s)
		return s, nil
	}
	if b.context != nil && types.Identical(t, b.context) {
		if !b.g.WithCtx {
			return nil, b.missingError(t, stack, "add loom.WithContext() to the graph to inject context.Context")
		}
		s := &Step{Kind: StepContext, Type: t}
		b.memoize(t, s)
		return s, nil
	}

	cands := b.index.Lookup(t)
	switch len(cands) {
	case 0:
		return nil, b.missingError(t, stack, "")
	case 1:
	default:
		return nil, b.duplicateError(t, cands)
	}
	pr := cands[0]

	if s, ok := b.done[pr]; ok {
		b.memoize(t, s)
		return s, nil
	}
	if b.inProgress[pr] {
		return nil, b.cycleError(t, stack)
	}

	b.inProgress[pr] = true
	s := &Step{Kind: StepProvider, Provider: pr, Type: t, Pos: pr.Pos}
	stack = append(stack, frame{typ: t, provider: pr})
	for _, in := range pr.Inputs {
		dep, err := b.resolve(in, stack)
		if err != nil {
			return nil, err
		}
		s.Deps = append(s.Deps, dep)
	}

	delete(b.inProgress, pr)
	b.done[pr] = s
	b.steps = append(b.steps, s)
	b.memoize(t, s)
	return s, nil
}

func (b *builder) memoize(t types.Type, s *Step) {
	b.byType.Set(t, s)
}

// --- diagnostics -----------------------------------------------------------

func (b *builder) fmt() diag.Formatter {
	return diag.Formatter{Current: b.g.Pkg.Types, Base: b.base}
}

func (b *builder) path(stack []frame, missing types.Type) []diag.Frame {
	f := b.fmt()
	out := make([]diag.Frame, 0, len(stack)+1)
	for _, fr := range stack {
		out = append(out, diag.Frame{
			Type:  fr.typ,
			Label: diag.ProviderLabel(fr.provider.Name, fr.provider.Inputs, f),
			Pos:   fr.provider.DeclPos,
		})
	}
	out = append(out, diag.Frame{Type: missing})
	return out
}

func (b *builder) missingError(t types.Type, stack []frame, hint string) error {
	f := b.fmt()
	e := &diag.Error{
		Msg:  "cannot build " + f.Type(b.g.Target),
		Note: "missing",
		Path: b.path(stack, t),
		Fmt:  f,
		Hint: hint,
	}
	if hint == "" {
		e.Hint = b.interfaceSuggestion(t)
	}
	return e
}

// interfaceSuggestion looks for a provider whose result implements t. Loom
// requires explicit bindings, so this turns a bare "no provider" error into an
// actionable one.
func (b *builder) interfaceSuggestion(t types.Type) string {
	if _, ok := types.Unalias(t).Underlying().(*types.Interface); !ok {
		return ""
	}
	f := b.fmt()
	for _, p := range b.g.Providers {
		if types.Identical(p.Output, t) || p.Binding != nil {
			continue
		}
		if types.AssignableTo(p.Output, t) {
			return "provider " + p.Name + " returns " + f.Type(p.Output) +
				", which implements " + f.Type(t) + "; bind it with loom.As[" + f.Type(t) + "](" + p.Name + ")"
		}
	}
	return ""
}

func (b *builder) cycleError(t types.Type, stack []frame) error {
	f := b.fmt()
	start := 0
	for i, fr := range stack {
		if types.Identical(fr.typ, t) {
			start = i
			break
		}
	}
	cycle := append([]frame(nil), stack[start:]...)
	cycle = append(cycle, frame{typ: t, provider: stack[start].provider})

	path := make([]diag.Frame, 0, len(cycle))
	for _, fr := range cycle {
		path = append(path, diag.Frame{
			Type:  fr.typ,
			Label: diag.ProviderLabel(fr.provider.Name, fr.provider.Inputs, f),
			Pos:   fr.provider.DeclPos,
		})
	}

	return &diag.Error{
		Msg:  "dependency cycle detected",
		Note: "cycle",
		Path: path,
		Fmt:  f,
	}
}

func (b *builder) duplicateError(t types.Type, cands []*model.Provider) error {
	f := b.fmt()
	e := &diag.Error{
		Msg:  "multiple providers found for " + f.Type(t),
		Pos:  cands[1].Pos,
		Fmt:  f,
		Hint: cands[0].Name + " at " + cands[0].Pos.String() + " and " + cands[1].Name + " at " + cands[1].Pos.String(),
	}
	return e
}
