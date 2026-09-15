// Package parse discovers loom.Graph declarations and turns them into the
// model IR. It reads AST nodes only to find source locations and reproduce
// expressions; every type decision goes through go/types.
package parse

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Xwudao/loom/internal/diag"
	"github.com/Xwudao/loom/internal/load"
	"github.com/Xwudao/loom/internal/model"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// DefaultLoomPath is the import path of the loom runtime package.
const DefaultLoomPath = "github.com/Xwudao/loom"

// Parser turns loom.Graph declarations into graphs.
type Parser struct {
	ld           *load.Loader
	loomPath     string
	base         string
	contextType  types.Type
	moduleType   types.Type
	lifecyclePtr types.Type
}

// New returns a Parser over a loaded package graph. base is the directory used
// to render source paths in diagnostics.
func New(ld *load.Loader, loomPath, base string) *Parser {
	if loomPath == "" {
		loomPath = DefaultLoomPath
	}
	p := &Parser{
		ld:          ld,
		loomPath:    loomPath,
		base:        base,
		contextType: ld.Type("context", "Context"),
	}
	if mod := ld.Type(loomPath, "ModuleDef"); mod != nil {
		p.moduleType = mod
	}
	if lc := ld.Type(loomPath, "Lifecycle"); lc != nil {
		p.lifecyclePtr = types.NewPointer(lc)
	}
	return p
}

// LifecycleType returns *loom.Lifecycle, or nil if the loom package is not part
// of the loaded graph.
func (p *Parser) LifecycleType() types.Type { return p.lifecyclePtr }

// ContextType returns context.Context, or nil.
func (p *Parser) ContextType() types.Type { return p.contextType }

// Package returns the graphs declared in pkg, ordered by variable name.
func (p *Parser) Package(pkg *packages.Package) ([]*model.Graph, error) {
	var graphs []*model.Graph
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, val := range vs.Values {
					call, ok := unparen(val).(*ast.CallExpr)
					if !ok || p.loomFunc(pkg.TypesInfo, call, "Graph") == nil {
						continue
					}
					varName := ""
					if i < len(vs.Names) {
						varName = vs.Names[i].Name
					}
					g, err := p.graph(pkg, varName, call)
					if err != nil {
						return nil, err
					}
					graphs = append(graphs, g)
				}
			}
		}
	}
	sort.Slice(graphs, func(i, j int) bool { return graphs[i].VarName < graphs[j].VarName })
	return graphs, nil
}

type parseState struct {
	seenModules map[*types.Var]bool
	// explicitName is set by loom.Name; empty means derive from the variable.
	explicitName string
	withCtx      bool
}

func (p *Parser) graph(pkg *packages.Package, varName string, call *ast.CallExpr) (*model.Graph, error) {
	if varName == "" || varName == "_" {
		return nil, p.errf(pkg, call.Pos(), "loom.Graph must be assigned to a named package-level variable")
	}
	target, err := p.targetType(pkg, call)
	if err != nil {
		return nil, err
	}

	g := &model.Graph{
		VarName: varName,
		Target:  target,
		Pkg:     pkg,
		Pos:     pkg.Fset.Position(call.Pos()),
	}

	st := &parseState{seenModules: map[*types.Var]bool{}}
	if err := p.options(g, st, pkg, call.Args); err != nil {
		return nil, err
	}
	g.WithCtx = st.withCtx

	name := st.explicitName
	if name == "" {
		name = deriveName(varName)
	}
	if name == "" {
		return nil, p.errf(pkg, call.Pos(),
			"cannot derive a function name from variable %q; add loom.Name(\"...\")", varName)
	}
	g.Name = name

	if err := p.checkProviders(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (p *Parser) targetType(pkg *packages.Package, call *ast.CallExpr) (types.Type, error) {
	tv, ok := pkg.TypesInfo.Types[call]
	if !ok {
		return nil, p.errf(pkg, call.Pos(), "internal error: no type information for loom.Graph call")
	}
	named, ok := types.Unalias(tv.Type).(*types.Named)
	if !ok || named.TypeArgs().Len() != 1 {
		return nil, p.errf(pkg, call.Pos(), "loom.Graph requires exactly one type argument")
	}
	return named.TypeArgs().At(0), nil
}

// options walks the arguments of a Graph or Module call.
func (p *Parser) options(g *model.Graph, st *parseState, pkg *packages.Package, args []ast.Expr) error {
	info := pkg.TypesInfo
	for _, arg := range args {
		arg = unparen(arg)

		// A reference to a module declared elsewhere.
		if obj := objectOf(info, arg); obj != nil && p.isModule(obj) {
			v, ok := obj.(*types.Var)
			if !ok {
				return p.errf(pkg, arg.Pos(), "loom: invalid module reference")
			}
			if st.seenModules[v] {
				continue // modules are sets: including one twice is a no-op
			}
			st.seenModules[v] = true
			mpkg, call, err := p.moduleCall(v)
			if err != nil {
				return err
			}
			if err := p.options(g, st, mpkg, call.Args); err != nil {
				return err
			}
			continue
		}

		call, ok := arg.(*ast.CallExpr)
		if !ok {
			return p.errf(pkg, arg.Pos(), "loom: unsupported graph element %s", describeExpr(arg))
		}
		switch fn := p.loomFunc(info, call, ""); {
		case fn == nil:
			return p.errf(pkg, arg.Pos(), "loom: unsupported graph element %s", describeExpr(arg))
		case fn.Name() == "Provide":
			pr, err := p.provide(pkg, call, nil)
			if err != nil {
				return err
			}
			g.Providers = append(g.Providers, pr)
		case fn.Name() == "As":
			iface, err := p.bindingType(pkg, call)
			if err != nil {
				return err
			}
			pr, err := p.provide(pkg, call, iface)
			if err != nil {
				return err
			}
			g.Providers = append(g.Providers, pr)
		case fn.Name() == "Supply":
			pr, err := p.supply(pkg, call)
			if err != nil {
				return err
			}
			g.Providers = append(g.Providers, pr)
		case fn.Name() == "Module":
			if err := p.options(g, st, pkg, call.Args); err != nil {
				return err
			}
		case fn.Name() == "Name":
			name, err := p.nameOption(pkg, call)
			if err != nil {
				return err
			}
			st.explicitName = name
		case fn.Name() == "WithContext":
			st.withCtx = true
		default:
			return p.errf(pkg, arg.Pos(), "loom: %s is not a valid graph element", fn.Name())
		}
	}
	return nil
}

func (p *Parser) nameOption(pkg *packages.Package, call *ast.CallExpr) (string, error) {
	if len(call.Args) != 1 {
		return "", p.errf(pkg, call.Pos(), "loom.Name requires exactly one string argument")
	}
	lit, ok := unparen(call.Args[0]).(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", p.errf(pkg, call.Pos(), "loom.Name requires a string literal")
	}
	name, err := strconv.Unquote(lit.Value)
	if err != nil || !isIdentifier(name) {
		return "", p.errf(pkg, call.Pos(), "loom.Name(%s) is not a valid Go identifier", lit.Value)
	}
	return name, nil
}

func (p *Parser) bindingType(pkg *packages.Package, call *ast.CallExpr) (types.Type, error) {
	args := explicitTypeArgs(pkg.TypesInfo, call.Fun)
	if len(args) == 0 || args[0] == nil {
		return nil, p.errf(pkg, call.Pos(), "loom.As requires an interface type argument")
	}
	iface := args[0]
	if _, ok := types.Unalias(iface).Underlying().(*types.Interface); !ok {
		return nil, p.errf(pkg, call.Pos(), "loom.As[%s]: type argument must be an interface", p.tf(pkg, iface))
	}
	return iface, nil
}

func (p *Parser) provide(pkg *packages.Package, call *ast.CallExpr, iface types.Type) (*model.Provider, error) {
	if len(call.Args) != 1 {
		return nil, p.errf(pkg, call.Pos(), "loom: provider requires exactly one constructor argument")
	}
	pr, err := p.constructor(pkg, call.Args[0])
	if err != nil {
		return nil, err
	}
	if iface != nil {
		if !types.AssignableTo(pr.Output, iface) {
			return nil, p.errf(pkg, call.Pos(),
				"provider %s returns %s, which does not implement %s",
				pr.Name, p.tf(pkg, pr.Output), p.tf(pkg, iface))
		}
		pr.Binding = iface
	}
	return pr, nil
}

func (p *Parser) constructor(pkg *packages.Package, ctor ast.Expr) (*model.Provider, error) {
	info := pkg.TypesInfo
	sig, _ := info.TypeOf(ctor).(*types.Signature)
	if sig == nil {
		return nil, p.errf(pkg, ctor.Pos(), "provider %s is not a function", describeExpr(ctor))
	}
	if sig.TypeParams().Len() > 0 {
		return nil, p.errf(pkg, ctor.Pos(),
			"provider %s is generic; instantiate it with type arguments, e.g. %s[T]",
			describeExpr(ctor), describeExpr(baseExpr(ctor)))
	}

	refObj, err := p.constructorRef(pkg, ctor)
	if err != nil {
		return nil, err
	}
	refPkg := refObj.Pkg()
	refName := refObj.Name()

	pr := &model.Provider{
		Kind:     model.KindConstructor,
		Name:     refName,
		Pos:      pkg.Fset.Position(ctor.Pos()),
		DeclPos:  pkg.Fset.Position(refObj.Pos()),
		RefObj:   refObj,
		RefPkg:   refPkg,
		RefName:  refName,
		TypeArgs: typeArgsOf(info, ctor),
	}

	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		if sig.Variadic() && i == params.Len()-1 {
			if slice, ok := t.(*types.Slice); ok {
				t = slice.Elem()
				pr.Variadic = true
			}
		}
		pr.Inputs = append(pr.Inputs, t)
	}

	res := sig.Results()
	if res.Len() == 0 {
		return nil, p.errf(pkg, ctor.Pos(), "provider %s must return a value", refName)
	}
	pr.Output = res.At(0).Type()

	rest := make([]types.Type, 0, res.Len()-1)
	for i := 1; i < res.Len(); i++ {
		rest = append(rest, res.At(i).Type())
	}
	switch len(rest) {
	case 0:
	case 1:
		switch {
		case p.isError(rest[0]):
			pr.HasError = true
		case p.isCleanup(rest[0]):
			pr.Cleanup = rest[0]
		default:
			return nil, p.errf(pkg, ctor.Pos(),
				"provider %s returns an unsupported second result %s (want error or a cleanup function)",
				refName, p.tf(pkg, rest[0]))
		}
	case 2:
		if !p.isCleanup(rest[0]) || !p.isError(rest[1]) {
			return nil, p.errf(pkg, ctor.Pos(),
				"provider %s must return (T, Cleanup, error); got (%s, %s, %s)",
				refName, p.tf(pkg, pr.Output), p.tf(pkg, rest[0]), p.tf(pkg, rest[1]))
		}
		pr.Cleanup = rest[0]
		pr.HasError = true
	default:
		return nil, p.errf(pkg, ctor.Pos(),
			"provider %s has %d results; loom supports T, (T, error), (T, Cleanup), (T, Cleanup, error)",
			refName, res.Len())
	}
	return pr, nil
}

// constructorRef validates that ctor names a package-level function or a
// package-level variable of function type.
func (p *Parser) constructorRef(pkg *packages.Package, ctor ast.Expr) (types.Object, error) {
	base := baseExpr(ctor)
	var obj types.Object
	switch e := base.(type) {
	case *ast.Ident:
		obj = pkg.TypesInfo.ObjectOf(e)
	case *ast.SelectorExpr:
		obj = pkg.TypesInfo.ObjectOf(e.Sel)
	}
	if obj == nil {
		return nil, p.errf(pkg, ctor.Pos(),
			"provider %s is not a reference to a package-level function", describeExpr(ctor))
	}
	switch obj.(type) {
	case *types.Func:
	case *types.Var:
		if obj.Parent() != obj.Pkg().Scope() {
			return nil, p.errf(pkg, ctor.Pos(),
				"provider %s refers to a local variable; use a package-level function", describeExpr(ctor))
		}
	default:
		return nil, p.errf(pkg, ctor.Pos(),
			"provider %s is a %s; loom requires a package-level function", describeExpr(ctor), objectKind(obj))
	}
	if obj.Pkg() == nil {
		return nil, p.errf(pkg, ctor.Pos(), "provider %s has no package", obj.Name())
	}
	return obj, nil
}

func (p *Parser) supply(pkg *packages.Package, call *ast.CallExpr) (*model.Provider, error) {
	if len(call.Args) != 1 {
		return nil, p.errf(pkg, call.Pos(), "loom.Supply requires exactly one value")
	}
	expr := call.Args[0]
	t := pkg.TypesInfo.TypeOf(expr)
	if t == nil {
		return nil, p.errf(pkg, expr.Pos(), "loom.Supply: cannot determine the value's type")
	}
	name := "value"
	switch e := unparen(expr).(type) {
	case *ast.Ident:
		name = e.Name
	case *ast.SelectorExpr:
		name = e.Sel.Name
	}
	return &model.Provider{
		Kind:    model.KindValue,
		Name:    name,
		Pos:     pkg.Fset.Position(expr.Pos()),
		Output:  t,
		Expr:    expr,
		ExprPkg: pkg,
	}, nil
}

// checkProviders validates uniqueness and framework-reserved types.
func (p *Parser) checkProviders(g *model.Graph) error {
	var index model.Index
	for _, pr := range g.Providers {
		targets := []types.Type{pr.Output}
		if pr.Binding != nil {
			targets = append(targets, pr.Binding)
		}
		for _, t := range targets {
			if p.lifecyclePtr != nil && types.Identical(t, p.lifecyclePtr) {
				return p.errPos(g.Pkg, pr.Pos, "provider %s provides %s, which loom owns and cannot be provided", pr.Name, p.tf(g.Pkg, t))
			}
			existing := index.Lookup(t)
			if len(existing) > 0 {
				return p.duplicateError(g, t, existing, pr)
			}
			index.Add(t, pr)
		}
	}
	return nil
}

func (p *Parser) duplicateError(g *model.Graph, t types.Type, existing []*model.Provider, pr *model.Provider) error {
	f := diag.Formatter{Current: g.Pkg.Types, Base: p.base}
	frames := make([]diag.Frame, 0, len(existing)+1)
	for _, e := range existing {
		frames = append(frames, diag.Frame{
			Type:  t,
			Label: diag.ProviderLabel(e.Name, e.Inputs, f),
			Pos:   e.Pos,
		})
	}
	frames = append(frames, diag.Frame{
		Type:  t,
		Label: diag.ProviderLabel(pr.Name, pr.Inputs, f),
		Pos:   pr.Pos,
	})
	return &diag.Error{
		Msg:       fmt.Sprintf("multiple providers found for %s", f.Type(t)),
		Providers: frames,
		Fmt:       f,
	}
}

func (p *Parser) moduleCall(v *types.Var) (*packages.Package, *ast.CallExpr, error) {
	pkg, ok := p.ld.Package(v.Pkg().Path())
	if !ok {
		return nil, nil, fmt.Errorf("loom: cannot load package %s for module %s", v.Pkg().Path(), v.Name())
	}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range vs.Names {
					if n.Name != v.Name() || i >= len(vs.Values) {
						continue
					}
					call, ok := unparen(vs.Values[i]).(*ast.CallExpr)
					if !ok {
						return nil, nil, fmt.Errorf("loom: module %s must be initialized with loom.Module(...)", v.Name())
					}
					return pkg, call, nil
				}
			}
		}
	}
	return nil, nil, fmt.Errorf("loom: cannot find the declaration of module %s in %s", v.Name(), v.Pkg().Path())
}

// --- helpers ---------------------------------------------------------------

func (p *Parser) isModule(obj types.Object) bool {
	if p.moduleType == nil {
		return false
	}
	return types.Identical(obj.Type(), p.moduleType)
}

func (p *Parser) isError(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func (p *Parser) isCleanup(t types.Type) bool {
	sig, ok := types.Unalias(t).Underlying().(*types.Signature)
	if !ok || sig.Variadic() {
		return false
	}
	switch sig.Results().Len() {
	case 0:
	case 1:
		if !p.isError(sig.Results().At(0).Type()) {
			return false
		}
	default:
		return false
	}
	switch sig.Params().Len() {
	case 0:
	case 1:
		if p.contextType == nil || !types.Identical(sig.Params().At(0).Type(), p.contextType) {
			return false
		}
	default:
		return false
	}
	return true
}

// loomFunc returns the loom function being called, or nil. When name is empty
// any loom function matches.
func (p *Parser) loomFunc(info *types.Info, call *ast.CallExpr, name string) *types.Func {
	fn, _ := typeutil.Callee(info, call).(*types.Func)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != p.loomPath {
		return nil
	}
	if name != "" && fn.Name() != name {
		return nil
	}
	return fn
}

func (p *Parser) errf(pkg *packages.Package, pos token.Pos, format string, args ...any) error {
	return p.errPos(pkg, pkg.Fset.Position(pos), format, args...)
}

// tf formats a type for diagnostics, relative to pkg.
func (p *Parser) tf(pkg *packages.Package, t types.Type) string {
	return diag.Formatter{Current: pkg.Types, Base: p.base}.Type(t)
}

func (p *Parser) errPos(pkg *packages.Package, pos token.Position, format string, args ...any) error {
	return &diag.Error{
		Msg: fmt.Sprintf(format, args...),
		Pos: pos,
		Fmt: diag.Formatter{Current: pkg.Types, Base: p.base},
	}
}

func unparen(e ast.Expr) ast.Expr {
	for {
		pe, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = pe.X
	}
}

func baseExpr(e ast.Expr) ast.Expr {
	switch v := unparen(e).(type) {
	case *ast.IndexExpr:
		return v.X
	case *ast.IndexListExpr:
		return v.X
	default:
		return unparen(e)
	}
}

func explicitTypeArgs(info *types.Info, fun ast.Expr) []types.Type {
	switch e := unparen(fun).(type) {
	case *ast.IndexExpr:
		return []types.Type{info.TypeOf(e.Index)}
	case *ast.IndexListExpr:
		out := make([]types.Type, 0, len(e.Indices))
		for _, idx := range e.Indices {
			out = append(out, info.TypeOf(idx))
		}
		return out
	}
	return nil
}

// typeArgsOf returns explicit type arguments of an instantiated function
// expression, or nil.
func typeArgsOf(info *types.Info, expr ast.Expr) []types.Type {
	base := baseExpr(expr)
	if base == expr {
		return nil
	}
	var id *ast.Ident
	switch e := base.(type) {
	case *ast.Ident:
		id = e
	case *ast.SelectorExpr:
		id = e.Sel
	}
	if id == nil {
		return explicitTypeArgs(info, expr)
	}
	inst, ok := info.Instances[id]
	if !ok {
		return explicitTypeArgs(info, expr)
	}
	out := make([]types.Type, 0, inst.TypeArgs.Len())
	for i := 0; i < inst.TypeArgs.Len(); i++ {
		out = append(out, inst.TypeArgs.At(i))
	}
	return out
}

func objectOf(info *types.Info, e ast.Expr) types.Object {
	switch v := unparen(e).(type) {
	case *ast.Ident:
		return info.ObjectOf(v)
	case *ast.SelectorExpr:
		return info.ObjectOf(v.Sel)
	}
	return nil
}

func describeExpr(e ast.Expr) string {
	switch v := unparen(e).(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return describeExpr(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return describeExpr(v.Fun) + "(...)"
	case *ast.IndexExpr:
		return describeExpr(v.X) + "[...]"
	case *ast.IndexListExpr:
		return describeExpr(v.X) + "[...]"
	case *ast.CompositeLit:
		return "composite literal"
	default:
		return fmt.Sprintf("expression (%T)", e)
	}
}

func objectKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "function"
	case *types.Var:
		return "variable"
	case *types.Const:
		return "constant"
	case *types.TypeName:
		return "type"
	default:
		return "value"
	}
}

// deriveName maps a graph variable name to a generated function name:
// AppGraph -> InitApp, workerGraph -> InitWorker.
func deriveName(varName string) string {
	base := varName
	for _, suffix := range []string{"Graph", "graph"} {
		if strings.HasSuffix(base, suffix) && len(base) > len(suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	if base == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(base)
	return "Init" + string(unicode.ToUpper(r)) + base[size:]
}

func isIdentifier(s string) bool {
	if s == "" || token.IsKeyword(s) {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}
