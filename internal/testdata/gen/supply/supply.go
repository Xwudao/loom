// Package supply exercises loom.Supply with a package variable and a composite
// literal.
package supply

import "github.com/Xwudao/loom"

// Labels are attached to the application.
type Labels struct {
	App string
}

// Config is supplied as a package-level variable.
type Config struct {
	Name string
}

// Version is supplied as a composite literal.
type Version struct {
	Major int
	Minor int
}

// App is the assembled application.
type App struct {
	Config  *Config
	Version Version
}

var config = &Config{Name: "supply"}

// NewApp builds the application.
func NewApp(cfg *Config, v Version) *App { return &App{Config: cfg, Version: v} }

// AppGraph supplies existing values instead of constructing them.
var AppGraph = loom.Graph[*App](
	loom.Supply(config),
	loom.Supply(Version{Major: 1, Minor: 2}),
	loom.Provide(NewApp),
)
