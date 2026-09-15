package loom

// This file defines the compile-time marker API. Every function here is a
// no-op at runtime: it exists so that graph declarations are ordinary,
// type-checked Go code. The `loom` command finds these calls and generates
// real initialization code; the marker values are never used at runtime.

// Option is a compile-time marker accepted by [Graph] and [Module].
//
// Option is a sealed interface: only this package can implement it, so graph
// declarations cannot be extended with arbitrary values.
type Option interface{ loomOption() }

type marker struct{}

func (marker) loomOption() {}

// Provider marks a constructor as a provider. Construct one with [Provide],
// [As], or [Supply]; the zero value is meaningless.
type Provider struct{ marker }

// ModuleDef is the value produced by a [Module] declaration. Like [GraphDef]
// it is a compile-time marker that generated code does not reference.
type ModuleDef struct{ marker }

// GraphDef is the value produced by a [Graph] declaration. It is a compile-time
// marker: the generated initializer does not reference it.
type GraphDef[T any] struct{ marker }

type nameOption struct{ marker }

type contextOption struct{ marker }

// Provide registers ctor as a provider of the value it returns.
//
// ctor must be a package-level function (or a package-level variable holding a
// function) matching one of:
//
//	func(...) T
//	func(...) (T, error)
//	func(...) (T, Cleanup)
//	func(...) (T, Cleanup, error)
//
// Generic constructors must be instantiated explicitly:
//
//	loom.Provide(NewRepository[User])
func Provide[F any](ctor F) Provider { return Provider{} }

// As registers ctor as a provider of the interface I.
//
//	loom.As[UserStore](NewUserStore)
//
// The generator verifies that the constructor's result implements I and
// reports a diagnostic at the provider's source location if it does not. The
// constructor's concrete result type is provided as well, so a binding never
// duplicates an instance.
func As[I any, F any](ctor F) Provider { return Provider{} }

// Supply registers an existing value as a provider.
//
//	loom.Supply(config)
//
// The expression must be reproducible in generated code: identifiers, package
// qualified names, field selections, literals, composite literals, and calls
// whose operands are themselves reproducible. Note that Go evaluates the
// expression once during package initialization and again inside the generated
// initializer; assign side-effecting expressions to a package-level variable
// first.
func Supply[T any](value T) Provider { return Provider{} }

// Module groups providers so that a related set can be declared once and
// included in several graphs. Modules may contain other modules.
func Module(opts ...Option) ModuleDef { return ModuleDef{} }

// Graph declares the dependency graph used to build T.
//
//	// AppGraph wires the application.
//	var AppGraph = loom.Graph[*App](
//		loom.Provide(NewConfig),
//		loom.Provide(NewDB),
//		loom.Provide(NewApp),
//	)
//
// Graph must be assigned to a package-level variable. The generated initializer
// is named after that variable: AppGraph produces InitApp. Use [Name] to
// override the name.
func Graph[T any](opts ...Option) GraphDef[T] { return GraphDef[T]{} }

// Name sets the name of the function generated for a graph.
//
//	var appGraph = loom.Graph[*App](loom.Name("BuildApp"), ...)
func Name(name string) Option { return nameOption{} }

// WithContext makes the generated initializer accept a context.Context and
// makes context.Context an injectable dependency.
//
//	func InitApp(ctx context.Context) (*App, *loom.Lifecycle, error)
//
// Prefer doing I/O in a lifecycle OnStart hook; use WithContext when a
// constructor genuinely needs a context, such as a database dialer.
func WithContext() Option { return contextOption{} }
