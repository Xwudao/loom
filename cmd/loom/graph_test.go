package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphMachineFormats(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		format string
		want   string
	}{
		{format: "json", want: `"name": "InitApp"`},
		{format: "dot", want: "digraph loom {"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			cmd := exec.Command("go", "run", "./cmd/loom", "graph", "-format="+tc.format, "./examples/basic")
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("loom graph: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Errorf("output = %s, want %q", out, tc.want)
			}
		})
	}
}
