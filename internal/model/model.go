// Package model holds Loom's intermediate representation: providers and
// graphs expressed in terms of go/types, never in terms of strings or AST
// nodes alone.
package model

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// Kind distinguishes how a provider produces its value.
type Kind uint8

const (
	// KindConstructor calls a package-level function (or function-valued
	// package-level variable).
	KindConstructor Kind = iota
	// KindValue copies an existing expression, registered with loom.Supply.
	KindValue
)

// Provider is a single value-producing node of a dependency graph.
type Provider struct {
	Kind Kind

	// Name is the display name used in diagnostics and variable naming, such
	// as "NewDB" or "config".
	Name string
	// Pos is the provider's source position: the reference in the graph
	// declaration.
	Pos token.Position
	// DeclPos is the position where the constructor is defined.
	DeclPos token.Position
	// Modules records the named module expansion path that included this
	// provider. It is empty for providers written directly in the graph.
	Modules []string

	// Output is the type of the value this provider produces.
	Output types.Type
	// Bindings are the interface types this provider also serves, set by
	// loom.As[I](ctor). A graph may add one to a constructor a module already
	// provides, which is how Wire's graph-level wire.Bind is expressed.
	Bindings []types.Type

	// Inputs are the constructor's parameter types, in order. For a variadic
	// constructor the element type is used and Variadic is set.
	Inputs   []types.Type
	Variadic bool

	// HasError reports whether the constructor returns an error as its last
	// result.
	HasError bool
	// Cleanup is the exact function type of the constructor's cleanup result,
	// or nil. Its signature is normalized during code generation.
	Cleanup types.Type

	// Reference to the constructor, used to emit the call.
	RefObj   types.Object
	RefPkg   *types.Package
	RefName  string
	TypeArgs []types.Type

	// Value expression for KindValue.
	Expr ast.Expr
	// ExprPkg is the package in which Expr appears; its TypesInfo resolves the
	// identifiers in Expr.
	ExprPkg *packages.Package
}

// Graph is a parsed dependency graph declaration.
type Graph struct {
	// VarName is the package-level variable holding the declaration.
	VarName string
	// Name is the name of the generated initializer function.
	Name string
	// Target is the type the graph builds.
	Target types.Type
	// WithCtx reports whether loom.WithContext() was declared.
	WithCtx bool
	// Pos is the declaration's source position.
	Pos token.Position
	// Pkg is the package containing the declaration.
	Pkg *packages.Package
	// Providers are the flattened providers, in declaration order.
	Providers []*Provider
}

// Index maps types to providers using go/types identity: equal types are
// determined by types.Identical, not by string comparison.
type Index struct {
	m typeutil.Map // types.Type -> []*Provider
}

// Add registers p as a provider of t.
func (ix *Index) Add(t types.Type, p *Provider) {
	if t == nil {
		return
	}
	if v := ix.m.At(t); v != nil {
		ix.m.Set(t, append(v.([]*Provider), p))
		return
	}
	ix.m.Set(t, []*Provider{p})
}

// Lookup returns the providers registered for t.
func (ix *Index) Lookup(t types.Type) []*Provider {
	if v := ix.m.At(t); v != nil {
		return v.([]*Provider)
	}
	return nil
}

// Targets lists every type this provider serves: its result plus each interface
// it is bound to.
// ModuleSuffix returns a concise provenance annotation for diagnostics.
func (p *Provider) ModuleSuffix() string {
	if len(p.Modules) == 0 {
		return ""
	}
	return " (via " + strings.Join(p.Modules, " -> ") + ")"
}

func (p *Provider) Targets() []types.Type {
	targets := make([]types.Type, 0, 1+len(p.Bindings))
	targets = append(targets, p.Output)
	return append(targets, p.Bindings...)
}

// HasBinding reports whether the provider already serves the interface t.
func HasBinding(p *Provider, t types.Type) bool {
	for _, b := range p.Bindings {
		if types.Identical(b, t) {
			return true
		}
	}
	return false
}

// SameConstructorHint explains the most common duplicate provider: one
// constructor listed twice in the same provider declaration.
//
// loom.As already provides the concrete type as well as the interface, so the
// plain Provide beside it is redundant. Without this hint the provider list
// shows the same constructor name twice at two nearby positions, which reads
// like a bug in loom rather than a redundant line.
//
// This covers only entries in one declaration. A graph may legitimately expose
// a constructor that a module already provides; see Parser.checkProviders.
//
// typeString renders a type for display, so this package does not need to know
// how diagnostics format types.
func SameConstructorHint(cands []*Provider, typeString func(types.Type) string) string {
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			a, b := cands[i], cands[j]
			if a.RefObj == nil || a.RefObj != b.RefObj || !SameTypeArgs(a, b) {
				continue
			}
			bound, plain := a, b
			if len(bound.Bindings) == 0 {
				bound, plain = b, a
			}
			if len(bound.Bindings) == 0 {
				continue
			}
			iface := typeString(bound.Bindings[0])
			return "loom.As[" + iface + "](" + bound.RefName +
				") already provides both " + typeString(plain.Output) + " and " +
				iface + "; remove the separate loom.Provide(" + plain.RefName + ")"
		}
	}
	return ""
}

// SameTypeArgs reports whether two providers instantiate the same generic
// declaration the same way, so Repository[User] and Repository[Article] are
// never treated as the same constructor.
func SameTypeArgs(a, b *Provider) bool {
	if len(a.TypeArgs) != len(b.TypeArgs) {
		return false
	}
	for i := range a.TypeArgs {
		if !types.Identical(a.TypeArgs[i], b.TypeArgs[i]) {
			return false
		}
	}
	return true
}
