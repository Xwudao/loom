// Package model holds Loom's intermediate representation: providers and
// graphs expressed in terms of go/types, never in terms of strings or AST
// nodes alone.
package model

import (
	"go/ast"
	"go/token"
	"go/types"

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

	// Output is the type of the value this provider produces.
	Output types.Type
	// Binding, when non-nil, is an interface type this provider also serves.
	// It is set by loom.As[I](ctor).
	Binding types.Type

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

// SameConstructorHint explains the most common duplicate provider: one
// constructor registered both directly and through loom.As.
//
// As already provides the concrete type as well as the interface, so the plain
// Provide is redundant. Without this hint the provider list shows the same
// constructor name twice at two nearby positions, which reads like a bug in
// loom rather than a redundant line.
//
// typeString renders a type for display, so this package does not need to know
// how diagnostics format types.
func SameConstructorHint(cands []*Provider, typeString func(types.Type) string) string {
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			a, b := cands[i], cands[j]
			if a.RefObj == nil || a.RefObj != b.RefObj {
				continue
			}
			bound, plain := a, b
			if bound.Binding == nil {
				bound, plain = b, a
			}
			if bound.Binding == nil {
				continue
			}
			return "loom.As[" + typeString(bound.Binding) + "](" + bound.RefName +
				") already provides both " + typeString(plain.Output) + " and " +
				typeString(bound.Binding) + "; remove the separate loom.Provide(" +
				plain.RefName + ")"
		}
	}
	return ""
}
