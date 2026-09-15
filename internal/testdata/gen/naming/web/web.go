package web

import (
	"github.com/Xwudao/loom/internal/testdata/gen/naming/db"
	"github.com/Xwudao/loom/internal/testdata/gen/naming/settings"
)

// MainApp is the assembled web application.
type MainApp struct {
	DB       *db.DB
	Settings *settings.Settings
}

// NewMainApp builds the application.
func NewMainApp(database *db.DB, settings *settings.Settings) *MainApp {
	return &MainApp{DB: database, Settings: settings}
}
