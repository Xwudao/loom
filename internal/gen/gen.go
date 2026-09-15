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
	ld       *load.Loader
	parser   *parse.Parser
	fileOpts codegen.Options
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
	var out []GraphPlan
	for _, root := range s.ld.Roots {
		graphs, err := s.parser.Package(root)
		if err != nil {
			return nil, err
		}
		for _, g := range graphs {
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

	var results []Result
	for _, root := range s.ld.Roots {
		graphs, err := s.parser.Package(root)
		if err != nil {
			return nil, err
		}
		if len(graphs) == 0 {
			continue
		}
		dir, err := packageDir(root)
		if err != nil {
			return nil, err
		}

		if err := checkNames(root, graphs, baseDir(opts.Dir)); err != nil {
			return nil, err
		}

		file := codegen.NewFile(root, s.fileOpts, "")
		var names []string
		for _, g := range graphs {
			plan, err := resolve.Build(g, s.parser.LifecycleType(), s.parser.ContextType(), baseDir(opts.Dir))
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

		out := filepath.Join(dir, generatedFile)
		res := Result{Package: root.PkgPath, File: out, Graphs: names, Source: src}
		existing, readErr := os.ReadFile(out)
		switch {
		case readErr == nil && bytes.Equal(existing, src):
			// Already up to date: do not touch the file, so mtimes, IDE
			// reloads, and git status stay quiet.
		case opts.DryRun:
			res.Changed = true
		default:
			if err := os.WriteFile(out, src, 0o644); err != nil {
				return nil, fmt.Errorf("loom: writing %s: %w", out, err)
			}
			res.Changed = true
		}
		results = append(results, res)
	}
	return results, nil
}

// generatedFile is the base name of the file loom writes.
const generatedFile = "loom_gen.go"

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
		if strings.HasSuffix(pos.Filename, generatedFile) {
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
