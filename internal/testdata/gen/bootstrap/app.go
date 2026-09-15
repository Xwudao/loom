package bootstrap

import (
	"context"

	"github.com/Xwudao/loom"
)

// Config holds configuration.
type Config struct{ DSN string }

// App is the assembled application.
type App struct{ Config *Config }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{DSN: "memory://"} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewApp),
)

// Run calls InitApp, which loom has not generated yet. This is the normal state
// of a fresh checkout, of a brand new graph, and of a codebase migrating to
// loom: the call site is written before the file that implements it exists.
// Loom must treat "undefined: InitApp" as expected and still generate.
func Run(ctx context.Context) (*App, error) {
	app, lifecycle, err := InitApp()
	if err != nil {
		return nil, err
	}
	defer func() { _ = lifecycle.Stop(ctx) }()
	return app, nil
}
