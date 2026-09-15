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
	"path/filepath"
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
	base   string
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

	ld := &Loader{Fset: cfg.Fset, Roots: roots, base: absDir(dir), byPath: map[string]*packages.Package{}}
	packages.Visit(roots, nil, func(p *packages.Package) {
		if p.PkgPath == "" {
			return
		}
		if _, ok := ld.byPath[p.PkgPath]; !ok {
			ld.byPath[p.PkgPath] = p
		}
	})

	if err := listErrors(roots); err != nil {
		return nil, err
	}
	return ld, nil
}

// listErrors reports failures that make the loaded universe unusable: an
// unknown pattern, an unreadable directory, a missing module. They cannot be
// recovered from, so they are fatal immediately.
func listErrors(roots []*packages.Package) error {
	var msgs []string
	for _, p := range roots {
		for _, e := range p.Errors {
			if e.Kind == packages.ListError {
				msgs = append(msgs, e.Error())
			}
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	sort.Strings(msgs)
	return fmt.Errorf("loom: cannot load packages:\n\t%s", strings.Join(msgs, "\n\t"))
}

// GeneratedFile is the base name of the file Loom writes. Loom owns this file
// outright: it is always rewritten in full, so diagnostics inside it never
// describe a real problem in the input.
const GeneratedFile = "loom_gen.go"

// Generated records the functions Loom is about to emit.
//
// Loom has no stub file: the initializer only comes into existence when the
// generated file is written. On a first run every call site of that initializer
// is therefore reported by the type checker as "undefined", which must not stop
// generation. Generated is used by [Loader.CheckErrors] to tell those expected
// errors apart from real ones.
type Generated struct {
	// local maps a package path to the initializer names generated in it, for
	// references written without a qualifier.
	local map[string]map[string]bool
	// qualified maps a package name to the initializer names generated in it,
	// for references written as pkg.Func from a dependent package.
	qualified map[string]map[string]bool
	// packages records the package paths whose generated file Loom is about to
	// rewrite.
	packages map[string]bool
}

// NewGenerated returns an empty set.
func NewGenerated() *Generated {
	return &Generated{
		local:     map[string]map[string]bool{},
		qualified: map[string]map[string]bool{},
		packages:  map[string]bool{},
	}
}

// Add records that name will be generated in pkg.
func (g *Generated) Add(pkg *packages.Package, name string) {
	if name == "" {
		return
	}
	g.packages[pkg.PkgPath] = true
	if g.local[pkg.PkgPath] == nil {
		g.local[pkg.PkgPath] = map[string]bool{}
	}
	g.local[pkg.PkgPath][name] = true
	if pkg.Name != "" {
		if g.qualified[pkg.Name] == nil {
			g.qualified[pkg.Name] = map[string]bool{}
		}
		g.qualified[pkg.Name][name] = true
	}
}

// ignores reports whether msg is an "undefined" error in pkg that refers to a
// function Loom is about to generate.
func (g *Generated) ignores(pkg *packages.Package, msg string) bool {
	ref, ok := undefinedRef(msg)
	if !ok {
		return false
	}
	if qual, ident, found := strings.Cut(ref, "."); found {
		return g.qualified[qual][ident]
	}
	return g.local[pkg.PkgPath][ref]
}

// undefinedRef extracts the identifier from an "undefined: X" error message.
// go/types batches diagnostics, so the message may carry a suffix such as
// " (and 2 more errors)".
func undefinedRef(msg string) (string, bool) {
	const prefix = "undefined: "
	i := strings.Index(msg, prefix)
	if i < 0 {
		return "", false
	}
	ref := msg[i+len(prefix):]
	if j := strings.IndexAny(ref, " ("); j >= 0 {
		ref = ref[:j]
	}
	if ref == "" {
		return "", false
	}
	return ref, true
}

// absDir returns an absolute form of dir, used to keep reported positions
// relative so diagnostics do not depend on where the checkout lives.
func absDir(dir string) string {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	return abs
}

// relPos rewrites a "file:line:col" position reported by go/packages so that it
// is relative to the directory the patterns were resolved against, matching the
// paths loom's own diagnostics use.
func (l *Loader) relPos(pos string) string {
	file, tail := splitPos(pos)
	if file == "" || l.base == "" {
		return pos
	}
	rel, err := filepath.Rel(l.base, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return pos
	}
	return rel + tail
}

// splitPos separates a "file:line:col" string into its file part and the
// remaining ":line:col" suffix. Requiring the suffix to be numeric keeps
// Windows drive letters and colons inside file names intact.
func splitPos(pos string) (file, tail string) {
	i := strings.LastIndex(pos, ":")
	if i < 0 || !allDigits(pos[i+1:]) {
		return pos, ""
	}
	j := strings.LastIndex(pos[:i], ":")
	if j < 0 || !allDigits(pos[j+1:i]) {
		return pos, ""
	}
	return pos[:j], pos[j:]
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ignoresStaleFile reports whether an error position points into a generated
// file that Loom is about to overwrite.
//
// A generated file can stop compiling on its own — a newer Loom may emit a
// different import set, or a constructor signature may have changed since the
// last run. Loom rewrites the whole file, so those errors must not block the run
// that would fix them.
func (g *Generated) ignoresStaleFile(pkg *packages.Package, pos string) bool {
	if !g.packages[pkg.PkgPath] || pos == "" {
		return false
	}
	file, _ := splitPos(pos)
	return filepath.Base(file) == GeneratedFile
}

// CheckErrors returns an error if any root package reported a type error other
// than a reference to a function Loom is about to generate. Generated may be
// nil, in which case every type error is reported.
func (l *Loader) CheckErrors(gen *Generated) error {
	var msgs []string
	for _, p := range l.Roots {
		for _, e := range p.Errors {
			if e.Kind == packages.ListError {
				continue // reported by Load
			}
			if gen != nil {
				if gen.ignores(p, e.Msg) || gen.ignoresStaleFile(p, e.Pos) {
					continue
				}
			}
			if pos := l.relPos(e.Pos); pos != "" {
				msgs = append(msgs, pos+": "+e.Msg)
				continue
			}
			msgs = append(msgs, e.Msg)
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	sort.Strings(msgs)
	return fmt.Errorf("loom: packages do not type-check:\n\t%s", strings.Join(msgs, "\n\t"))
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
