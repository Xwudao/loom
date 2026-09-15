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
