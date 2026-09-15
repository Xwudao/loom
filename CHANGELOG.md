# Changelog

All notable changes to Loom are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and Loom adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-09-15

### Added

- MIT license, a compatibility policy, and CI checks for generated-file drift,
  `go vet`, and race-enabled tests.
- `loom graph -format=json` and `loom graph -format=dot` for tooling and graph
  visualization. The existing text format remains the default.
- Module expansion paths in provider diagnostics, making it clear which nested
  module included a provider.

### Fixed

- `Lifecycle.Append` and `Lifecycle.AddCleanup` now reject calls as soon as
  `Start` begins. Previously an entry appended during startup could miss its
  `OnStart` callback and have inconsistent shutdown semantics.

## [0.2.0] - 2026-09-15

### Added

- A graph may bind a constructor to an interface without re-declaring it:
  `loom.As[I](pkg.NewC)` alongside the module that already provides `NewC` now
  adds the binding to that provider instead of reporting a duplicate. This is
  Wire's graph-level `wire.Bind` shape, and it is the only way to express the
  binding when the interface lives in a package that imports the module —
  putting `As` in the module would close an import cycle.
  Listing one constructor twice in the same declaration is still an error, since
  that is a redundant line rather than a statement about what the graph exposes.
- A provider returning `context.Context` is now used instead of being reported
  as a missing dependency. `loom.WithContext()` still threads the caller's
  context and is still the recommended way, but a graph may supply its own, as
  Wire projects with a `ProvideContext` function do. An explicit provider wins,
  which keeps the generated function signature a straight conversion.

Found while migrating a fourth project, which used both patterns.

## [0.1.0] - 2026-09-15

First release intended for real projects. The API below is the stable surface;
anything not listed under "What Loom is not" in the README is likely to change.

### Added

- Declarative graphs: `loom.Graph[T]`, `loom.Provide`, `loom.As`, `loom.Supply`,
  `loom.Module`, `loom.Name`, `loom.WithContext`.
- Compile-time resolution over `go/types`, with `Repository[User]` and
  `Repository[Article]` treated as distinct keys.
- Deterministic code generation that emits ordinary Go calling constructors in
  dependency order, with stable imports and variable names.
- `*loom.Lifecycle`: ordered start/stop hooks, constructor cleanups registered
  in construction order, and `Rollback` so a failed build releases what it
  already created.
- Diagnostics that print the full dependency path, every provider's source
  position, and an actionable hint where one exists.
- `loom generate`, `loom graph`, `loom version`, `loom help`.
- `loom generate -dry-run`, which exits non-zero when a committed generated file
  is out of date, so CI can gate on drift.
- Generation is all-or-nothing: every package is rendered before anything is
  written, and unchanged files are left alone.

### Notes

- Generation tolerates call sites of initializers that do not exist yet, which
  is unavoidable without Wire-style stub files. Only the exact `undefined:
  <name>` errors Loom is about to satisfy are ignored.
- `loom.As[I](ctor)` registers the constructor and exposes its result as `I`, so
  it replaces a `loom.Provide(ctor)` entry rather than accompanying it.
- Requires Go 1.25 or newer; built and tested on Go 1.27.

## [0.0.4] - 2026-09-15

### Added

- A hint for the most common duplicate provider, registering one constructor
  both directly and through `loom.As`. The provider list otherwise showed the
  same constructor name twice at two nearby positions, which reads like a bug
  in Loom rather than a redundant line. Found while migrating a project from
  Wire, where the equivalent `wire.NewSet(NewStore, wire.Bind(...))` spells the
  constructor only once.

## [0.0.3] - 2026-09-15

### Changed

- `loom generate -dry-run` now exits non-zero when a generated file is out of
  date, so it can be used as a CI freshness check.

## [0.0.2] - 2026-09-15

### Fixed

- A graph whose initializer is already referenced by a call site could not be
  generated at all: package loading failed with `undefined: mainApp` before the
  initializer existed. Loom now records the names it is about to generate and
  ignores exactly those errors, while still reporting genuine typos.
- A stale `loom_gen.go` carrying a type error no longer blocks regeneration.
  Loom owns that file and rewrites it whole.
- Generated variable names now prefer a package-qualified name over a numeric
  suffix when two providers would otherwise collide, so `data.Data` becomes
  `dataData` rather than `data2`, and `cmd.MainApp` inside `func mainApp`
  becomes `cmdMainApp`.
- Import registration during naming no longer depends on provider order, which
  could emit an unused import and leave the generated file uncompilable.

## [0.0.1] - 2026-09-15

### Added

- Initial release.

[0.2.1]: https://github.com/Xwudao/loom/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Xwudao/loom/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Xwudao/loom/compare/v0.0.4...v0.1.0
[0.0.4]: https://github.com/Xwudao/loom/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/Xwudao/loom/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/Xwudao/loom/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/Xwudao/loom/releases/tag/v0.0.1
