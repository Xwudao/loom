package methodexpression

import "github.com/Xwudao/loom"

// Config is a dependency.
type Config struct{}

// App is the assembled application.
type App struct{ Config *Config }

// Factory builds Config values.
type Factory struct{}

// NewConfig is a method, not a package-level function.
func (f *Factory) NewConfig() *Config { return &Config{} }

// NewApp builds the application.
func NewApp(cfg *Config) *App { return &App{Config: cfg} }

// AppGraph uses a method expression as a provider, which loom must reject
// before emitting a call to a function that does not exist.
var AppGraph = loom.Graph[*App](
	loom.Provide((*Factory).NewConfig),
	loom.Provide(NewApp),
)
