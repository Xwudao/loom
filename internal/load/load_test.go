package load

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestGeneratedQualifiedErrorUsesImportPath(t *testing.T) {
	const src = "package client\nimport alias \"example.org/other\"\nvar _ = alias.InitApp\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "client.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var qualifier *ast.Ident
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "InitApp" {
			qualifier = sel.X.(*ast.Ident)
		}
		return true
	})
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	client := types.NewPackage("example.org/client", "client")
	info.Uses[qualifier] = types.NewPkgName(qualifier.Pos(), client, "alias", types.NewPackage("example.org/other", "other"))
	pkg := &packages.Package{PkgPath: client.Path(), Fset: fset, Syntax: []*ast.File{file}, TypesInfo: info}
	e := packages.Error{Pos: fset.Position(qualifier.Pos()).String(), Msg: "undefined: alias.InitApp"}
	g := NewGenerated()
	g.Add(&packages.Package{PkgPath: "example.org/generated", Name: "other"}, "InitApp")
	if g.ignores(pkg, e) {
		t.Fatal("ignored error from a different package with the same name")
	}
	info.Uses[qualifier] = types.NewPkgName(qualifier.Pos(), client, "alias", types.NewPackage("example.org/generated", "other"))
	if !g.ignores(pkg, e) {
		t.Fatal("did not ignore missing generated function through an aliased import")
	}
	e.Pos = "client.go:1:1"
	if g.ignores(pkg, e) {
		t.Fatal("ignored an error at another location")
	}
}
