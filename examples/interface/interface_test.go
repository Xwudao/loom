package iface

import (
	"context"
	"testing"
)

func TestInterfaceBinding(t *testing.T) {
	app, _, err := InitApp()
	if err != nil {
		t.Fatalf("InitApp: %v", err)
	}
	if app.Service == nil || app.Service.Store == nil {
		t.Fatal("the interface binding was not wired")
	}

	user, err := app.Service.Store.Find(context.Background(), 1)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if user.Name != "ada" {
		t.Fatalf("user = %+v, want ada", user)
	}

	if _, err := app.Service.Store.Find(context.Background(), 99); err == nil {
		t.Fatal("expected a not-found error")
	}
}
