package duplicatebinding

import "github.com/Xwudao/loom"

// Store is the interface both providers are bound to.
type Store interface{ Get() string }

type storeA struct{}

func (s *storeA) Get() string { return "a" }

type storeB struct{}

func (s *storeB) Get() string { return "b" }

// NewStoreA is one implementation.
func NewStoreA() *storeA { return &storeA{} }

// NewStoreB is another implementation.
func NewStoreB() *storeB { return &storeB{} }

// Service depends on Store.
type Service struct{ Store Store }

// NewService builds the service.
func NewService(s Store) *Service { return &Service{Store: s} }

// App is the assembled application.
type App struct{ Service *Service }

// NewApp builds the application.
func NewApp(s *Service) *App { return &App{Service: s} }

// AppGraph binds two implementations to the same interface.
var AppGraph = loom.Graph[*App](
	loom.As[Store](NewStoreA),
	loom.As[Store](NewStoreB),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
