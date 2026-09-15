# Compatibility policy

Loom follows semantic versioning. This policy distinguishes the API consumed by
applications from generator implementation details.

## Stable within a major version

- The exported marker API: `Graph`, `Module`, `Provide`, `As`, `Supply`,
  `Name`, `WithContext`, and their documented marker types.
- The `Lifecycle`, `Hook`, and `Cleanup` APIs and their documented lifecycle
  ordering semantics.
- Generated initializer names and signatures: `InitX() (T, *loom.Lifecycle,
  error)`, plus a leading `context.Context` only for `WithContext` graphs.
- The generated file name (`loom_gen.go`) and its ownership by Loom.
- `loom generate -dry-run` exiting non-zero when generated output is stale.

## May change in a minor release

Generated local variable names, import aliases, whitespace, graph text output,
and diagnostic wording may improve without notice. Generated output remains
ordinary, formatted Go and is intended to be committed, not edited.

The JSON and DOT forms of `loom graph` are intended for tooling. Additive fields
and attributes may be introduced; consumers should ignore fields they do not
recognize.

## Upgrading

Pin the generator with Go's tool dependency mechanism and upgrade it together
with the runtime package. Run `go tool loom generate ./...`, review the generated
diff, then run `go test ./...`. CI should run `go tool loom generate -dry-run
./...` to prevent drift.
