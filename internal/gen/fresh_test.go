package gen_test

import (
	"path/filepath"
	"testing"

	"github.com/Xwudao/loom/internal/gen"
)

// TestGeneratedFilesAreFresh ensures the committed generated files match what
// the generator produces today. This is the CI guard that keeps examples from
// drifting: run `loom generate ./...` after changing providers.
func TestGeneratedFilesAreFresh(t *testing.T) {
	root := repoRoot(t)
	results, err := gen.Run(gen.Options{Dir: root, Patterns: []string{"./examples/..."}})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no graphs found under ./examples/...")
	}
	for _, r := range results {
		if r.Changed {
			rel, _ := filepath.Rel(root, r.File)
			t.Errorf("%s is out of date; run `loom generate ./...`", rel)
		}
	}
}
