package full

import (
	"context"
	"testing"
)

func TestInitApp(t *testing.T) {
	ctx := context.Background()
	app, lc, err := InitApp(ctx)
	if err != nil {
		t.Fatalf("InitApp: %v", err)
	}

	if app.Name != "full" {
		t.Fatalf("Name = %q, want full (supplied value)", app.Name)
	}
	if app.Config.LogLevel != "info" || app.Logger.Level != "info" {
		t.Fatal("configuration was not threaded through")
	}
	if app.Users == nil || app.Users.Store == nil {
		t.Fatal("UserStore binding missing")
	}
	if app.Articles == nil || app.Articles.Repo == nil {
		t.Fatal("Repository[Article] missing")
	}
	user, err := app.Users.Store.Find(ctx, 1)
	if err != nil || user.Name != "ada" {
		t.Fatalf("Find = %+v, %v", user, err)
	}

	if err := lc.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !app.Server.Running {
		t.Fatal("server hook did not run")
	}
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !app.Database.Closed {
		t.Fatal("database cleanup did not run")
	}
}

func TestInitWorker(t *testing.T) {
	worker, lc, err := InitWorker()
	if err != nil {
		t.Fatalf("InitWorker: %v", err)
	}
	if worker.DB == nil || worker.Log == nil {
		t.Fatal("worker was not wired")
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !worker.DB.Closed {
		t.Fatal("database cleanup did not run")
	}
}
