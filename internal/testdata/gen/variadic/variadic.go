// Package variadic verifies that a variadic constructor receives a []T
// provider and is called with a spread argument.
package variadic

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{}

// App is the assembled application.
type App struct{ Opts []string }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{} }

// NewOptions builds the variadic argument.
func NewOptions() []string { return []string{"a", "b"} }

// NewApp takes a variadic parameter; loom spreads the []string provider.
func NewApp(_ *Config, opts ...string) *App { return &App{Opts: opts} }

// AppGraph wires the application.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewOptions),
	loom.Provide(NewApp),
)
