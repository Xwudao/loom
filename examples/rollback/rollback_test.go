package rollback

import (
	"strings"
	"testing"
)

func TestFailedConstructionRollsBackInReverse(t *testing.T) {
	events = nil
	app, lc, err := InitApp()
	if err == nil {
		t.Fatal("expected construction to fail")
	}
	if app != nil || lc != nil {
		t.Fatal("InitApp must return nil value and lifecycle on error")
	}
	if !strings.Contains(err.Error(), "server: boom") {
		t.Fatalf("error = %v, want the server error", err)
	}
	if got := strings.Join(events, ","); got != "cache,db" {
		t.Fatalf("cleanup order = %q, want cache,db", got)
	}
}
