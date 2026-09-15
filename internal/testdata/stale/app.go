// Package stale exists to prove that a generated file which no longer compiles
// is overwritten rather than treated as broken input.
package stale

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{ Name string }

// App is the assembled application.
type App struct{ Config *Config }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{Name: "stale"} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewApp),
)
