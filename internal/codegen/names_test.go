package codegen

import (
	"go/types"
	"testing"
)

func fakePkg(path, name string) *types.Package { return types.NewPackage(path, name) }

func TestLowerFirst(t *testing.T) {
	tests := map[string]string{
		"Config":      "config",
		"DB":          "db",
		"HTTPServer":  "httpServer",
		"ID":          "id",
		"URLParser":   "urlParser",
		"UserStore":   "userStore",
		"userStore":   "userStore",
		"API":         "api",
		"A":           "a",
		"":            "",
		"AB":          "ab",
		"Repository":  "repository",
		"OAuthClient": "oAuthClient",
	}
	for in, want := range tests {
		if got := lowerFirst(in); got != want {
			t.Errorf("lowerFirst(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUpperFirst(t *testing.T) {
	if got := upperFirst("db"); got != "Db" {
		t.Errorf("upperFirst(db) = %q, want Db", got)
	}
	if got := upperFirst(""); got != "" {
		t.Errorf("upperFirst(\"\") = %q, want \"\"", got)
	}
}

func TestIdentFromProvider(t *testing.T) {
	tests := map[string]string{
		"NewConfig":      "config",
		"NewUserService": "userService",
		"NewDB":          "db",
		"ProvideFoo":     "foo",
		"MakeThing":      "thing",
		"helper":         "helper",
		"New":            "new",
	}
	for in, want := range tests {
		if got := identFromProvider(in); got != want {
			t.Errorf("identFromProvider(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNamerDeduplicates(t *testing.T) {
	n := newNamer()
	n.reserve("ctx", "err")
	if got := n.alloc("db"); got != "db" {
		t.Fatalf("alloc(db) = %q", got)
	}
	if got := n.alloc("db"); got != "db2" {
		t.Fatalf("second alloc(db) = %q, want db2", got)
	}
	if got := n.alloc("db"); got != "db3" {
		t.Fatalf("third alloc(db) = %q, want db3", got)
	}
	if got := n.alloc("ctx"); got != "ctx2" {
		t.Fatalf("alloc(reserved ctx) = %q, want ctx2", got)
	}
	if got := n.alloc("error"); got != "errorValue" {
		t.Fatalf("alloc(predeclared error) = %q, want errorValue", got)
	}
	if got := n.alloc("not an ident!"); got != "value" {
		t.Fatalf("alloc(invalid) = %q, want value", got)
	}
}

func TestImportSetAliases(t *testing.T) {
	// Two packages named model must not collide.
	s := newImportSet()
	if got := s.add(fakePkg("example.com/a/model", "model")); got != "model" {
		t.Fatalf("first alias = %q, want model", got)
	}
	if got := s.add(fakePkg("example.com/b/model", "model")); got != "model2" {
		t.Fatalf("second alias = %q, want model2", got)
	}
	// The same path must return a stable alias.
	if got := s.add(fakePkg("example.com/a/model", "model")); got != "model" {
		t.Fatalf("repeat alias = %q, want model", got)
	}
	// Predeclared names are avoided.
	if got := s.add(fakePkg("example.com/error", "error")); got != "error2" {
		t.Fatalf("predeclared alias = %q, want error2", got)
	}
}

func TestIsStdlib(t *testing.T) {
	for _, p := range []string{"context", "go/types", "net/http"} {
		if !isStdlib(p) {
			t.Errorf("isStdlib(%q) = false", p)
		}
	}
	for _, p := range []string{"github.com/Xwudao/loom", "example.com/x"} {
		if isStdlib(p) {
			t.Errorf("isStdlib(%q) = true", p)
		}
	}
}

func TestSimpleNamedType(t *testing.T) {
	dataPkg := types.NewPackage("example.com/internal/data", "data")
	data := types.NewNamed(types.NewTypeName(0, dataPkg, "Data", nil), types.NewStruct(nil, nil), nil)

	if got := simpleNamedType(data); got != data {
		t.Error("simpleNamedType(Data) did not return the named type")
	}
	if got := simpleNamedType(types.NewPointer(data)); got != data {
		t.Error("simpleNamedType(*Data) did not unwrap the pointer")
	}
	// Generic instantiations keep their type-argument prefix and are excluded.
	generic := types.NewNamed(types.NewTypeName(0, dataPkg, "Repo", nil), types.NewStruct(nil, nil), nil)
	generic.SetTypeParams([]*types.TypeParam{
		types.NewTypeParam(types.NewTypeName(0, dataPkg, "T", nil), types.NewInterfaceType(nil, nil)),
	})
	inst, err := types.Instantiate(nil, generic, []types.Type{types.Typ[types.String]}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := simpleNamedType(inst); got != nil {
		t.Errorf("simpleNamedType(Repo[string]) = %v, want nil", got)
	}
	// Non-named types have no package-qualified form.
	if got := simpleNamedType(types.Typ[types.String]); got != nil {
		t.Errorf("simpleNamedType(string) = %v, want nil", got)
	}
}

func TestNamerAllocPreferred(t *testing.T) {
	n := newNamer()
	// The plain name is free: use it.
	if got := n.allocPreferred("data", "dataData"); got != "data" {
		t.Fatalf("first allocPreferred = %q, want data", got)
	}
	// The plain name is taken by an alias: fall back to the qualified name
	// rather than a bare numeric suffix.
	if got := n.allocPreferred("data", "dataData"); got != "dataData" {
		t.Fatalf("second allocPreferred = %q, want dataData", got)
	}
	// Both taken: numeric suffix.
	if got := n.allocPreferred("data", "dataData"); got != "data2" {
		t.Fatalf("third allocPreferred = %q, want data2", got)
	}
	// A duplicate fallback must not shadow the base.
	n2 := newNamer()
	if got := n2.allocPreferred("db", "db"); got != "db" {
		t.Fatalf("allocPreferred(db, db) = %q, want db", got)
	}
	if got := n2.allocPreferred("db", "db"); got != "db2" {
		t.Fatalf("allocPreferred(db, db) second = %q, want db2", got)
	}
}
