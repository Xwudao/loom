package lifecycle

import (
	"context"
	"testing"
)

func TestHooksAndCleanups(t *testing.T) {
	app, lc, err := InitApp()
	if err != nil {
		t.Fatalf("InitApp: %v", err)
	}
	if app.Server.Running {
		t.Fatal("server must not be running before Start")
	}

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !app.Server.Running {
		t.Fatal("OnStart hook did not run")
	}

	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if app.Server.Running {
		t.Fatal("OnStop hook did not run")
	}
	if !app.Server.DB.Closed {
		t.Fatal("constructor cleanup did not run on Stop")
	}
	if !app.Server.Cache.Closed {
		t.Fatal("adapted func() cleanup did not run on Stop")
	}
}
