# Migrating from Google Wire to Loom

Loom replaces Wire's generated injector stub with an ordinary package-level graph. Migrate one injector at a time, then run `loom generate ./...` and tests.

## API mapping

| Wire | Loom |
| --- | --- |
| `wire.NewSet(a, b)` | `loom.Module(loom.Provide(a), loom.Provide(b))` |
| `wire.Build(...)` in a `wireinject` stub | `loom.Graph[T](...)` in an ordinary file |
| `wire.Bind(new(I), new(*C))` | `loom.As[I](NewC)` |
| injector `(T, func(), error)` | `(T, *loom.Lifecycle, error)` |
| `defer cleanup()` | `defer lifecycle.Stop(ctx)` |

## Procedure

1. Remove the `wireinject` build-tagged injector stub. Do not retain an alternate hand-written stub.
2. In an ordinary file in the injector package, declare a graph. Its variable name controls the generated initializer name: `AppGraph` generates `InitApp`.
3. Turn every constructor in each Wire set into `loom.Provide(Constructor)`, or group them in `loom.Module` values.
4. Replace each `wire.Bind(new(Interface), new(*Concrete))` with `loom.As[Interface](NewConcrete)`. Do **not** also write `loom.Provide(NewConcrete)`; `As` already provides the concrete value and the interface from one instance.
5. Run `loom generate ./...`. A new call to the generated initializer may be undefined before this first generation; that is expected.
6. Change the caller to receive the lifecycle and defer its stop. Start it before serving requests when the graph registers hooks.
7. Delete `wire_gen.go`, Wire imports, `wire` build tags, and the Wire tool dependency after all injectors are migrated.

## Example

Wire:

```go
//go:build wireinject

func InitApp() (*App, func(), error) {
    wire.Build(NewConfig, NewDB, NewStore, wire.Bind(new(Store), new(*store)), NewApp)
    return nil, nil, nil
}
```

Loom:

```go
var AppGraph = loom.Graph[*App](
    loom.Provide(NewConfig),
    loom.Provide(NewDB),
    loom.As[Store](NewStore),
    loom.Provide(NewApp),
)
```

After generation, call it as follows:

```go
app, lifecycle, err := InitApp()
if err != nil {
    return err
}
if err := lifecycle.Start(ctx); err != nil {
    return err
}
defer lifecycle.Stop(shutdownCtx)
```

## Important differences

### No stub file

Wire hides an injector implementation behind a build tag, so callers type-check only after Wire has generated code. Loom graphs are normal package-level values; gopls, `go vet`, and `go test` can see them. The first `loom generate` intentionally permits the missing generated initializer it is about to produce, while still reporting other type errors.

### `As` is a replacement for `Provide`

`loom.As[Store](NewStore)` registers `NewStore`, provides its concrete return type, and exposes the same instance as `Store`. Adding `loom.Provide(NewStore)` duplicates the provider and fails generation.

### Cleanup is lifecycle-managed

Wire returns an unstructured `func()` cleanup. Loom tracks constructor cleanups and hooks in `*loom.Lifecycle`: construction failures roll back completed resources, and `Stop` executes hooks and cleanups in reverse order. Do not manually call individual constructor cleanups after moving them into Loom.

### Context injection

Use `loom.WithContext()` when the generated initializer should receive the caller's `context.Context`. If the Wire injector got context from a provider function, keep that as `loom.Provide(ProvideContext)` instead; an explicit provider keeps the generated initializer parameterless.

### Modules and import cycles

Wire sets become Loom modules. If the interface package imports the package containing a module, put `loom.As[Interface](constructor)` in the consuming graph rather than the module, otherwise the binding introduces an import cycle.
