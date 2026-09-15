// Command loom generates dependency-injection code for Go programs.
//
// Usage:
//
//	loom generate [patterns...]   generate initializers (default ./...)
//	loom version                  print the version
//	loom help                     print this help
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xwudao/loom/internal/gen"
	"github.com/Xwudao/loom/internal/parse"
)

// version is overridable at build time with -ldflags "-X main.version=...".
var version = "devel"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage(os.Stderr)
		return fmt.Errorf("loom: missing command")
	}
	switch args[0] {
	case "generate", "gen":
		return generateCmd(args[1:])
	case "graph":
		return graphCmd(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("loom %s\n", version)
		return nil
	case "help", "--help", "-h":
		usage(os.Stdout)
		return nil
	default:
		usage(os.Stderr)
		return fmt.Errorf("loom: unknown command %q", args[0])
	}
}

func generateCmd(args []string) error {
	fs := flag.NewFlagSet("loom generate", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print what would change without writing files")
	loomPath := fs.String("loom-path", parse.DefaultLoomPath, "import path of the loom runtime package")
	if err := fs.Parse(args); err != nil {
		return err
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	results, err := gen.Run(gen.Options{
		Dir:      ".",
		Patterns: patterns,
		LoomPath: *loomPath,
		DryRun:   *dryRun,
	})
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "loom: no graph declarations found")
		return nil
	}
	var stale []string
	for _, r := range results {
		path := relPath(r.File)
		switch {
		case !r.Changed:
			fmt.Printf("loom: %s unchanged\n", path)
		case *dryRun:
			fmt.Printf("loom: %s out of date\n", path)
			stale = append(stale, path)
		default:
			fmt.Printf("loom: wrote %s (%v)\n", path, r.Graphs)
		}
	}
	if len(stale) > 0 {
		// Treat -dry-run as a check so it can gate CI: a non-zero exit means
		// the committed generated files are stale.
		return fmt.Errorf("loom: %d generated file(s) out of date; run 'loom generate'", len(stale))
	}
	return nil
}

// relPath renders path relative to the working directory when possible.
func relPath(path string) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

func usage(w *os.File) {
	fmt.Fprint(w, `loom - compile-time dependency injection for Go

Usage:
    loom generate [patterns...]   generate initializers (default ./...)
    loom graph [flags] [patterns...]  print the dependency graph
    loom version                  print the version
    loom help                     print this help

Flags for generate:
    -dry-run        check that generated files are up to date and exit non-zero
                    if they are not; useful in CI
    -loom-path P    import path of the loom package

Flags for graph:
    -format F       output format: text (default), dot, or json
    -loom-path P    import path of the loom package
`)
}
