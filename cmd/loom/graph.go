package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
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
func (p *treePrinter) typeLabel(t types.Type) string { return typeLabel(t, p.current) }

func typeLabel(t types.Type, current *types.Package) string {
	if t == nil {
		return "<nil>"
	}
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg == nil || pkg == current {
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

type jsonGraph struct {
	Name     string    `json:"name"`
	Position string    `json:"position"`
	Target   string    `json:"target"`
	Root     *jsonNode `json:"root"`
}

type jsonNode struct {
	Type         string      `json:"type"`
	Provider     string      `json:"provider,omitempty"`
	Shared       bool        `json:"shared,omitempty"`
	Dependencies []*jsonNode `json:"dependencies,omitempty"`
}

func jsonGraphs(plans []gen.GraphPlan) ([]byte, error) {
	out := make([]jsonGraph, 0, len(plans))
	for _, gp := range plans {
		current := gp.Graph.Pkg.Types
		seen := map[*resolve.Step]bool{}
		var node func(*resolve.Step, types.Type) *jsonNode
		node = func(step *resolve.Step, typ types.Type) *jsonNode {
			n := &jsonNode{Type: typeLabel(typ, current)}
			if step == nil {
				return n
			}
			if seen[step] {
				n.Shared = true
				return n
			}
			seen[step] = true
			if step.Provider != nil {
				n.Provider = step.Provider.Name
				for i, dep := range step.Deps {
					depType := dep.Type
					if i < len(step.Provider.Inputs) {
						depType = step.Provider.Inputs[i]
					}
					n.Dependencies = append(n.Dependencies, node(dep, depType))
				}
			}
			return n
		}
		out = append(out, jsonGraph{
			Name: gp.Graph.Name, Position: relPos(gp.Graph.Pos), Target: typeLabel(gp.Graph.Target, current), Root: node(gp.Plan.Root, gp.Graph.Target),
		})
	}
	return json.MarshalIndent(out, "", "  ")
}

func dotGraphs(plans []gen.GraphPlan) {
	fmt.Println("digraph loom {")
	fmt.Println("  rankdir=LR;")
	id := 0
	for _, gp := range plans {
		current := gp.Graph.Pkg.Types
		ids := map[*resolve.Step]string{}
		var visit func(*resolve.Step, types.Type) string
		visit = func(step *resolve.Step, typ types.Type) string {
			if name, ok := ids[step]; ok {
				return name
			}
			name := fmt.Sprintf("n%d", id)
			id++
			ids[step] = name
			label := typeLabel(typ, current)
			if step != nil && step.Provider != nil {
				label += "\\n" + step.Provider.Name
			}
			fmt.Printf("  %s [label=%s];\n", name, strconv.Quote(label))
			if step != nil && step.Provider != nil {
				for i, dep := range step.Deps {
					depType := dep.Type
					if i < len(step.Provider.Inputs) {
						depType = step.Provider.Inputs[i]
					}
					child := visit(dep, depType)
					fmt.Printf("  %s -> %s;\n", name, child)
				}
			}
			return name
		}
		visit(gp.Plan.Root, gp.Graph.Target)
	}
	fmt.Println("}")
}

// graphCmd implements `loom graph`.
func graphCmd(args []string) error {
	fs := flag.NewFlagSet("loom graph", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text, dot, or json")
	loomPath := fs.String("loom-path", parse.DefaultLoomPath, "import path of the loom runtime package")
	if err := fs.Parse(args); err != nil {
		return err
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	plans, err := gen.Inspect(gen.Options{Dir: ".", Patterns: patterns, LoomPath: *loomPath})
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Println("loom: no graph declarations found")
		return nil
	}
	switch *format {
	case "text":
		printGraphs(plans)
	case "dot":
		dotGraphs(plans)
	case "json":
		out, err := jsonGraphs(plans)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	default:
		return fmt.Errorf("loom: unknown graph format %q (want text, dot, or json)", *format)
	}
	return nil
}
