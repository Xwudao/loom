package contextprovider

import (
	"context"

	"github.com/Xwudao/loom"
)

// Config carries the application name.
type Config struct{ Name string }

// NewConfig builds the config.
func NewConfig() *Config { return &Config{Name: "app"} }

// Service needs a context, which this graph supplies itself rather than
// threading one in with loom.WithContext, the way a Wire injector does.
type Service struct {
	Config  *Config
	Context context.Context
}

// NewService builds the service.
func NewService(cfg *Config, ctx context.Context) *Service {
	return &Service{Config: cfg, Context: ctx}
}

// App is the assembled application.
type App struct{ Service *Service }

// NewApp builds the application.
func NewApp(s *Service) *App { return &App{Service: s} }

// ProvideContext supplies the application context.
func ProvideContext() context.Context { return context.Background() }

// AppGraph has no loom.WithContext, so ProvideContext is used.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(ProvideContext),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
