package parse

import "testing"

func TestDeriveName(t *testing.T) {
	tests := map[string]string{
		"AppGraph":    "InitApp",
		"appGraph":    "InitApp",
		"WorkerGraph": "InitWorker",
		"workerGraph": "InitWorker",
		"Worker":      "InitWorker",
		"Server":      "InitServer",
		"graph":       "InitGraph",
		"Graph":       "InitGraph",
	}
	for in, want := range tests {
		if got := deriveName(in); got != want {
			t.Errorf("deriveName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsIdentifier(t *testing.T) {
	for _, s := range []string{"InitApp", "initApp", "_x", "a1"} {
		if !isIdentifier(s) {
			t.Errorf("isIdentifier(%q) = false", s)
		}
	}
	for _, s := range []string{"", "1abc", "has space", "has-dash", "func"} {
		if isIdentifier(s) {
			t.Errorf("isIdentifier(%q) = true", s)
		}
	}
}
