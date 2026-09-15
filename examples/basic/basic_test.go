package basic

import (
	"context"
	"testing"
)

func TestInitAppWiresTheChain(t *testing.T) {
	app, lc, err := InitApp()
	if err != nil {
		t.Fatalf("InitApp: %v", err)
	}
	if app == nil || app.Service == nil || app.Service.Repo == nil || app.Service.Repo.DB == nil {
		t.Fatal("InitApp returned an incomplete graph")
	}
	if got := app.Service.Repo.DB.DSN; got != "memory://" {
		t.Fatalf("DSN = %q, want memory://", got)
	}
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
