package generic

import "testing"

func TestTypeArgumentsAreDistinctKeys(t *testing.T) {
	app, _, err := InitApp()
	if err != nil {
		t.Fatalf("InitApp: %v", err)
	}
	if app.Users == nil || app.Articles == nil {
		t.Fatal("InitApp returned an incomplete graph")
	}
	if app.Users.Repo.DB != app.Articles.Repo.DB {
		t.Fatal("both repositories must share one *DB instance")
	}
	if got := app.Users.Repo.DB.Opens; got != 1 {
		t.Fatalf("database opened %d times, want 1", got)
	}
	// Repository[User] and Repository[Article] must not be interchangeable;
	// the compiler enforces that, and the fields confirm the wiring.
	if app.Users.Repo == nil || app.Articles.Repo == nil {
		t.Fatal("repositories were not wired")
	}
}
