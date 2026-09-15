package asalsoprovided

import "github.com/Xwudao/loom"

// Store is the interface NewStore's result is bound to.
type Store interface{ Get() string }

type store struct{}

func (s *store) Get() string { return "store" }

// NewStore builds the implementation.
func NewStore() *store { return &store{} }

// Service consumes both the interface and the concrete type.
type Service struct {
	Store Store
	Raw   *store
}

// NewService builds the service.
func NewService(s Store, raw *store) *Service { return &Service{Store: s, Raw: raw} }

// App is the assembled application.
type App struct{ Service *Service }

// NewApp builds the application.
func NewApp(s *Service) *App { return &App{Service: s} }

// AppGraph registers NewStore twice: once directly and once bound to Store.
// loom.As already provides both types, so the plain Provide is redundant.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewStore),
	loom.As[Store](NewStore),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
