package gen_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Xwudao/loom/internal/gen"
)

// syntheticGraph writes a module containing a balanced binary dependency tree
// of n providers and returns its directory. The graph exercises loading,
// parsing, resolution, and code generation at scale.
func syntheticGraph(b testing.TB, n int) string {
	b.Helper()
	dir := b.TempDir()
	root := repoRoot(b)

	var mod strings.Builder
	fmt.Fprintf(&mod, "module synthetic\n\ngo 1.25\n\nrequire github.com/Xwudao/loom v0.0.0\n\nreplace github.com/Xwudao/loom => %s\n", root)
	write(b, filepath.Join(dir, "go.mod"), mod.String())

	var src strings.Builder
	src.WriteString("package app\n\nimport \"github.com/Xwudao/loom\"\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&src, "type T%d struct{}\n", i)
	}
	src.WriteString("\n")
	for i := 0; i < n; i++ {
		if i == 0 {
			src.WriteString("func NewT0() *T0 { return &T0{} }\n")
			continue
		}
		fmt.Fprintf(&src, "func NewT%d(prev *T%d) *T%d { return &T%d{} }\n", i, (i-1)/2, i, i)
	}
	fmt.Fprintf(&src, "\nvar AppGraph = loom.Graph[*T%d](\n", n-1)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&src, "\tloom.Provide(NewT%d),\n", i)
	}
	src.WriteString(")\n")
	write(b, filepath.Join(dir, "app", "app.go"), src.String())

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		b.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	return dir
}

func write(b testing.TB, path, content string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkGenerate measures end-to-end generation. Loading and type-checking
// the package dominates; resolution and code generation are linear in the
// number of providers.
func BenchmarkGenerate(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("providers=%d", n), func(b *testing.B) {
			dir := syntheticGraph(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := gen.Run(gen.Options{Dir: dir, Patterns: []string{"./app"}, DryRun: true}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkInspect measures loading, parsing, and resolution without code
// generation.
func BenchmarkInspect(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("providers=%d", n), func(b *testing.B) {
			dir := syntheticGraph(b, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := gen.Inspect(gen.Options{Dir: dir, Patterns: []string{"./app"}}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
