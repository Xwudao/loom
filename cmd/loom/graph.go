package main

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xwudao/loom/internal/gen"
	"github.com/Xwudao/loom/internal/parse"
	"github.com/Xwudao/loom/internal/resolve"
)

// treePrinter renders resolved graphs as ASCII trees.
type treePrinter struct {
	current *types.Package
}

// printGraphs renders every resolved graph as an ASCII dependency tree.
func printGraphs(plans []gen.GraphPlan) {
	p := &treePrinter{}
	for i, gp := range plans {
		if i > 0 {
			fmt.Println()
		}
		p.current = gp.Graph.Pkg.Types
		fmt.Printf("%s  (%s)\n", gp.Graph.Name, relPos(gp.Graph.Pos))
		p.step(gp.Plan.Root, gp.Graph.Target, "", true, true, map[*resolve.Step]bool{})
	}
}

// step draws one node and its dependencies. typ is the type requested by the
// parent, which may be an interface while the node's provider returns a
// concrete type.
func (p *treePrinter) step(step *resolve.Step, typ types.Type, prefix string, last, root bool, seen map[*resolve.Step]bool) {
	label := p.typeLabel(typ)
	if !root {
		connector := "├── "
		if last {
			connector = "└── "
		}
		switch {
		case step == nil:
			label += "  (missing)"
		case seen[step]:
			label += "  (shared)"
		}
		fmt.Printf("%s%s%s\n", prefix, connector, label)
	} else {
		fmt.Printf("%s\n", label)
	}
	if step == nil || seen[step] {
		return
	}
	seen[step] = true

	childPrefix := prefix
	if !root {
		if last {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	var children []types.Type
	if step.Provider != nil {
		children = step.Provider.Inputs
	}
	for i, dep := range step.Deps {
		childType := dep.Type
		if i < len(children) {
			childType = children[i]
		}
		p.step(dep, childType, childPrefix, i == len(step.Deps)-1, false, seen)
	}
}

// typeLabel renders a type using package names rather than full import paths
// and omits the qualifier for the graph's own package.
func (p *treePrinter) typeLabel(t types.Type) string {
	if t == nil {
		return "<nil>"
	}
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg == nil || pkg == p.current {
			return ""
		}
		return pkg.Name()
	})
}

// relPos renders a position relative to the working directory.
func relPos(pos token.Position) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, pos.Filename); err == nil && !strings.HasPrefix(rel, "..") {
			return fmt.Sprintf("%s:%d", rel, pos.Line)
		}
	}
	return pos.String()
}

// graphCmd implements `loom graph`.
func graphCmd(args []string) error {
	loomPath := parse.DefaultLoomPath
	var patterns []string
	for _, a := range args {
		if strings.HasPrefix(a, "-loom-path=") {
			loomPath = strings.TrimPrefix(a, "-loom-path=")
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		patterns = append(patterns, a)
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	plans, err := gen.Inspect(gen.Options{Dir: ".", Patterns: patterns, LoomPath: loomPath})
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Println("loom: no graph declarations found")
		return nil
	}
	printGraphs(plans)
	return nil
}
