// Package load wraps go/packages so that the rest of Loom works with a single,
// consistent go/types universe.
//
// A single packages.Load call is essential: types from two different loads are
// never types.Identical, so graph resolution would silently fail across package
// boundaries. Build constraints are handled by the go toolchain; Loom never
// parses //go:build lines itself.
package load

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Mode requests everything Loom needs: syntax, imports, types, and type info
// for the roots and every transitive dependency.
const Mode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo

// Loader holds the loaded package graph.
type Loader struct {
	Fset   *token.FileSet
	Roots  []*packages.Package
	byPath map[string]*packages.Package
}

// Load loads the packages matched by patterns, relative to dir.
func Load(dir string, patterns ...string) (*Loader, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	cfg := &packages.Config{
		Mode: Mode,
		Dir:  dir,
		Fset: token.NewFileSet(),
	}
	roots, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loom: loading packages: %w", err)
	}

	ld := &Loader{Fset: cfg.Fset, Roots: roots, byPath: map[string]*packages.Package{}}
	packages.Visit(roots, nil, func(p *packages.Package) {
		if p.PkgPath == "" {
			return
		}
		if _, ok := ld.byPath[p.PkgPath]; !ok {
			ld.byPath[p.PkgPath] = p
		}
	})

	if err := rootErrors(roots); err != nil {
		return nil, err
	}
	return ld, nil
}

// rootErrors returns an error if any root package failed to type-check.
func rootErrors(roots []*packages.Package) error {
	var msgs []string
	for _, p := range roots {
		for _, e := range p.Errors {
			if e.Kind == packages.ListError {
				msgs = append(msgs, e.Error())
			}
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("loom: cannot load packages:\n\t%s", strings.Join(msgs, "\n\t"))
	}
	var typeErrs []string
	for _, p := range roots {
		for _, e := range p.Errors {
			typeErrs = append(typeErrs, e.Error())
		}
	}
	if len(typeErrs) > 0 {
		sort.Strings(typeErrs)
		return fmt.Errorf("loom: packages do not type-check:\n\t%s", strings.Join(typeErrs, "\n\t"))
	}
	return nil
}

// Package returns a loaded package by import path. Every transitive
// dependency of the roots is available.
func (l *Loader) Package(path string) (*packages.Package, bool) {
	p, ok := l.byPath[path]
	return p, ok
}

// Type returns the named type decl in the package at path, or nil.
func (l *Loader) Type(path, name string) types.Type {
	p, ok := l.byPath[path]
	if !ok || p.Types == nil {
		return nil
	}
	obj := p.Types.Scope().Lookup(name)
	if obj == nil {
		return nil
	}
	return obj.Type()
}
