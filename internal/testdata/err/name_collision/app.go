package namecollision

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{}

// App is the assembled application.
type App struct{ Config *Config }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

// InitApp is hand-written, so loom cannot generate it.
func InitApp() {}

// AppGraph would generate InitApp.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewApp),
)
