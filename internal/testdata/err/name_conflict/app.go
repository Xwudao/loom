package nameconflict

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{}

// App is the assembled application.
type App struct{ Config *Config }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

var (
	// AppGraph and App both derive the generated name InitApp.
	AppGraph = loom.Graph[*App](
		loom.Provide(NewConfig),
		loom.Provide(NewApp),
	)

	appGraph = loom.Graph[*App](
		loom.Provide(NewConfig),
		loom.Provide(NewApp),
	)
)
