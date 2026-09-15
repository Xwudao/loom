// Package gen wires loading, parsing, resolving, and code generation into a
// single operation.
package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xwudao/loom/internal/codegen"
	"github.com/Xwudao/loom/internal/diag"
	"github.com/Xwudao/loom/internal/load"
	"github.com/Xwudao/loom/internal/model"
	"github.com/Xwudao/loom/internal/parse"
	"github.com/Xwudao/loom/internal/resolve"

	"golang.org/x/tools/go/packages"
)

// Options configures a generation run.
type Options struct {
	// Dir is the working directory for package patterns.
	Dir string
	// Patterns are package patterns such as ./... or ./cmd/server.
	Patterns []string
	// LoomPath is the loom package import path.
	LoomPath string
	// DryRun generates without writing files.
	DryRun bool
}

// Result reports what happened to one package.
type Result struct {
	Package string
	File    string
	Graphs  []string
	Changed bool
	Source  []byte
}

// GraphPlan pairs a parsed graph with its resolved build plan.
type GraphPlan struct {
	Graph *model.Graph
	Plan  *resolve.Plan
}

// session holds the shared loading, parsing, and codegen configuration.
type session struct {
	ld        *load.Loader
	parser    *parse.Parser
	fileOpts  codegen.Options
	generated *load.Generated
}

// unit pairs a root package with the graphs declared in it.
type unit struct {
	pkg    *packages.Package
	graphs []*model.Graph
}

// collect parses every root package and records the functions Loom is about to
// generate.
//
// It must run before Loader.CheckErrors: the initializer does not exist until
// the generated file is written, so on a first run its call sites are reported
// as undefined and would otherwise look like broken input.
func (s *session) collect() ([]unit, error) {
	s.generated = load.NewGenerated()
	units := make([]unit, 0, len(s.ld.Roots))
	for _, root := range s.ld.Roots {
		graphs, err := s.parser.Package(root)
		if err != nil {
			return nil, err
		}
		for _, g := range graphs {
			s.generated.Add(root, g.Name)
		}
		units = append(units, unit{pkg: root, graphs: graphs})
	}
	return units, nil
}

func open(opts Options) (*session, error) {
	if opts.LoomPath == "" {
		opts.LoomPath = parse.DefaultLoomPath
	}
	ld, err := load.Load(opts.Dir, opts.Patterns...)
	if err != nil {
		return nil, err
	}
	loomPkg, ok := ld.Package(opts.LoomPath)
	if !ok {
		return nil, fmt.Errorf("loom: package %s was not found; add it as a dependency of the module", opts.LoomPath)
	}
	parser := parse.New(ld, opts.LoomPath, baseDir(opts.Dir))
	fileOpts := codegen.Options{
		LoomPkg:      loomPkg.Types,
		LifecyclePtr: parser.LifecycleType(),
		Cleanup:      ld.Type(opts.LoomPath, "Cleanup"),
		Context:      parser.ContextType(),
	}
	if ep, ok := ld.Package("errors"); ok {
		fileOpts.ErrorsPkg = ep.Types
	}
	return &session{ld: ld, parser: parser, fileOpts: fileOpts}, nil
}

// Inspect loads the selected packages and returns every declared graph with its
// resolved plan, without generating code.
func Inspect(opts Options) ([]GraphPlan, error) {
	s, err := open(opts)
	if err != nil {
		return nil, err
	}
	units, err := s.collect()
	if err != nil {
		return nil, err
	}
	if err := s.ld.CheckErrors(s.generated); err != nil {
		return nil, err
	}
	var out []GraphPlan
	for _, u := range units {
		for _, g := range u.graphs {
			plan, err := resolve.Build(g, s.parser.LifecycleType(), s.parser.ContextType(), baseDir(opts.Dir))
			if err != nil {
				return nil, err
			}
			out = append(out, GraphPlan{Graph: g, Plan: plan})
		}
	}
	return out, nil
}

// Run generates code for every package matched by the options.
func Run(opts Options) ([]Result, error) {
	s, err := open(opts)
	if err != nil {
		return nil, err
	}
	units, err := s.collect()
	if err != nil {
		return nil, err
	}
	if err := s.ld.CheckErrors(s.generated); err != nil {
		return nil, err
	}

	// Render every package before writing any of it, so that a diagnostic in one
	// package cannot leave the others half-generated.
	base := baseDir(opts.Dir)
	results := make([]Result, 0, len(units))
	for _, u := range units {
		res, err := s.render(u, base)
		if err != nil {
			return nil, err
		}
		if res != nil {
			results = append(results, *res)
		}
	}
	for i := range results {
		if err := commit(&results[i], opts.DryRun); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// render resolves every graph declared in u and produces the file to write. It
// returns nil when the package declares no graphs.
func (s *session) render(u unit, base string) (*Result, error) {
	root, graphs := u.pkg, u.graphs
	if len(graphs) == 0 {
		return nil, nil
	}
	dir, err := packageDir(root)
	if err != nil {
		return nil, err
	}
	if err := checkNames(root, graphs, base); err != nil {
		return nil, err
	}

	file := codegen.NewFile(root, s.fileOpts, "")
	names := make([]string, 0, len(graphs))
	for _, g := range graphs {
		plan, err := resolve.Build(g, s.parser.LifecycleType(), s.parser.ContextType(), base)
		if err != nil {
			return nil, err
		}
		if err := file.AddGraph(g, plan); err != nil {
			return nil, err
		}
		names = append(names, g.Name)
	}
	src, err := file.Bytes()
	if err != nil {
		return nil, err
	}
	return &Result{
		Package: root.PkgPath,
		File:    filepath.Join(dir, load.GeneratedFile),
		Graphs:  names,
		Source:  src,
	}, nil
}

// commit writes res unless the file already has exactly that content, and
// records whether anything changed.
func commit(res *Result, dryRun bool) error {
	existing, readErr := os.ReadFile(res.File)
	switch {
	case readErr == nil && bytes.Equal(existing, res.Source):
		// Already up to date: do not touch the file, so mtimes, IDE reloads, and
		// git status stay quiet.
		return nil
	case dryRun:
		res.Changed = true
		return nil
	default:
		if err := os.WriteFile(res.File, res.Source, 0o644); err != nil {
			return fmt.Errorf("loom: writing %s: %w", res.File, err)
		}
		res.Changed = true
		return nil
	}
}

// checkNames rejects two graphs that would generate the same function, and
// graphs whose generated name collides with a hand-written declaration.
func checkNames(pkg *packages.Package, graphs []*model.Graph, base string) error {
	f := diag.Formatter{Current: pkg.Types, Base: base}
	seen := map[string]*model.Graph{}
	for _, g := range graphs {
		if prev, ok := seen[g.Name]; ok {
			return &diag.Error{
				Msg: fmt.Sprintf("graphs %s and %s both generate %s; rename one or use loom.Name",
					prev.VarName, g.VarName, g.Name),
				Pos: g.Pos,
				Fmt: f,
			}
		}
		seen[g.Name] = g
	}

	scope := pkg.Types.Scope()
	for _, g := range graphs {
		obj := scope.Lookup(g.Name)
		if obj == nil {
			continue
		}
		pos := pkg.Fset.Position(obj.Pos())
		if strings.HasSuffix(pos.Filename, load.GeneratedFile) {
			continue // our own output from a previous run
		}
		return &diag.Error{
			Msg: fmt.Sprintf("generated function %s conflicts with %s declared at %s; use loom.Name",
				g.Name, obj.Name(), f.Pos(pos)),
			Pos: g.Pos,
			Fmt: f,
		}
	}
	return nil
}

// baseDir returns an absolute version of dir for relative diagnostics.
func baseDir(dir string) string {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

// packageDir returns the directory holding the package's source files.
func packageDir(pkg *packages.Package) (string, error) {
	files := pkg.GoFiles
	if len(files) == 0 {
		files = pkg.CompiledGoFiles
	}
	if len(files) == 0 {
		return "", fmt.Errorf("loom: package %s has no Go files", pkg.PkgPath)
	}
	return filepath.Dir(files[0]), nil
}
