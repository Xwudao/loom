// Package buildtags verifies that loom honors //go:build constraints: the
// alternative provider in tagged.go must not be visible.
package buildtags

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{ Source string }

// App is the assembled application.
type App struct{ Config *Config }

// NewConfig is used unless the loom_test_tag build tag is set.
func NewConfig() *Config { return &Config{Source: "default"} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

// AppGraph wires the application.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewApp),
)
