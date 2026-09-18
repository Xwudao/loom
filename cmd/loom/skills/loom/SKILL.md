---
name: loom
description: Use Loom, the compile-time Go dependency-injection generator. Apply when creating, changing, generating, debugging, or migrating Loom dependency graphs.
---

# Loom dependency injection

Loom generates ordinary Go initializer functions from a package-level dependency graph. It has no runtime container or reflection. Run the generator after every graph or provider change:

```bash
loom generate ./...
# CI check: exits non-zero when generated files are stale
loom generate -dry-run ./...
```

Use `loom graph` to inspect a graph before debugging its resolution:

```bash
loom graph ./cmd/server
loom graph -format=dot ./cmd/server | dot -Tsvg > graph.svg
loom graph -format=json ./cmd/server
```

## Define a graph

Declare a package-level `loom.Graph[T]`. A graph variable named `AppGraph` generates `InitApp`; `workerGraph` generates `InitWorker`. Override the name with `loom.Name("BuildApp")`.

```go
var AppGraph = loom.Graph[*App](
    loom.Provide(NewConfig),
    loom.Provide(NewDB),
    loom.Provide(NewApp),
)
```

Generated initializers always return `(T, *loom.Lifecycle, error)`:

```go
app, lifecycle, err := InitApp()
if err != nil { return err }
if err := lifecycle.Start(ctx); err != nil { return err }
defer lifecycle.Stop(shutdownCtx)
```

There is no injector stub or build tag. It is normal for a new call to `InitApp` to be undefined before the first `loom generate`; Loom tolerates only the undefined initializer names it will generate.

## Register dependencies

Use these markers inside `loom.Graph` or `loom.Module`:

```go
loom.Provide(NewDB)                 // constructor result
loom.As[UserStore](NewUserStore)    // concrete result plus interface binding
loom.Supply(config)                 // existing relocatable expression
loom.WithContext()                  // generated initializer receives ctx context.Context
loom.Name("BuildApp")               // generated initializer name
```

A provider must be a package-level function or package-level function variable with one of these forms:

```go
func NewThing(...) T
func NewThing(...) (T, error)
func NewThing(...) (T, loom.Cleanup)
func NewThing(...) (T, loom.Cleanup, error)
```

Cleanup can also be `func()`, `func() error`, or `func(context.Context)`. Variadic constructors take a slice dependency: `func NewThing(opts ...string)` is resolved from a `[]string` provider and called with `opts...`. Split multi-value results such as `(A, B)` into separate constructors.

`loom.As[I](NewC)` replaces—not supplements—`loom.Provide(NewC)`: it makes both `*C` and `I` available from the same instance. Do not list both, or Loom reports a duplicate provider.

Use a module to share provider sets. Modules may nest, and including the same module more than once is a no-op.

```go
var DatabaseModule = loom.Module(
    loom.Provide(NewDatabase),
    loom.Provide(NewRepository[User]),
)

var AppGraph = loom.Graph[*App](
    DatabaseModule,
    loom.As[UserStore](NewUserStore),
    loom.Provide(NewApp),
)
```

Keep an `As` binding in the graph when the interface package imports the module package; putting it in the module would create an import cycle.

## Generics and supplied values

Instantiate generic constructors explicitly. Each instantiation is a separate dependency key:

```go
loom.Provide(NewRepository[User])
loom.Provide(NewRepository[Article])
```

`loom.Supply` copies its expression into the generated initializer. It is evaluated once at package initialization and again in the initializer, so assign side-effecting expressions to a package-level variable before supplying them.

## Context and lifecycle

`*loom.Lifecycle` is injectable. `loom.WithContext()` adds a leading `ctx context.Context` argument to the generated initializer. An explicit `context.Context` provider takes precedence and leaves the generated signature unchanged.

Register long-running services as hooks:

```go
func NewServer(lc *loom.Lifecycle, cfg *Config) *Server {
    server := &Server{Addr: cfg.Addr}
    lc.Append(loom.Hook{
        OnStart: func(ctx context.Context) error { return server.Listen(ctx) },
        OnStop:  func(ctx context.Context) error { return server.Close() },
    })
    return server
}
```

Hooks start in registration order and stop in reverse. Constructor cleanups are also run in reverse order, both after a construction failure and on `lifecycle.Stop`. Do not return a cleanup and register an `OnStop` hook for the same resource.

## Generated files and workflow

- Generated output is `loom_gen.go`; do not edit it.
- Generation is all-or-nothing: fix diagnostics before any files are written.
- Add `//go:generate go run github.com/Xwudao/loom/cmd/loom generate` to integrate with `go generate ./...`.
- Run `go test ./...` after regenerating.
- Read diagnostics' dependency path, provider locations, and suggested `loom.As` binding before adding arbitrary providers.

## Wire migrations

For a step-by-step mapping, cleanup conversion, and stub-file migration guidance, read [the Wire migration reference](references/wire-migration.md). Keep this reference separate rather than copying it into this skill.
