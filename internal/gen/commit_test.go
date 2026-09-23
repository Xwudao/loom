package gen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loom_gen.go")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := &Result{File: path, Source: []byte("new")}
	if err := commit(res, true); err != nil || !res.Changed {
		t.Fatalf("dry run: changed=%v err=%v", res.Changed, err)
	}
	if got, _ := os.ReadFile(path); string(got) != "old" {
		t.Fatalf("dry run wrote %q", got)
	}
	res.Changed = false
	if err := commit(res, false); err != nil || !res.Changed {
		t.Fatalf("commit: changed=%v err=%v", res.Changed, err)
	}
	if got, _ := os.ReadFile(path); string(got) != "new" {
		t.Fatalf("commit wrote %q", got)
	}
	res.Changed = false
	if err := commit(res, false); err != nil || res.Changed {
		t.Fatalf("unchanged: changed=%v err=%v", res.Changed, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files remain: %v, %v", entries, err)
	}
}
