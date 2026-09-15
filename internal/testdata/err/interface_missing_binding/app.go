package interfacemissingbinding

import "github.com/Xwudao/loom"

// User is a domain entity.
type User struct{ Name string }

// Store is the interface the service needs.
type Store interface{ Get() User }

type userStore struct{}

func (s *userStore) Get() User { return User{} }

// NewUserStore provides the concrete type but is never bound to Store.
func NewUserStore() *userStore { return &userStore{} }

// Service depends on Store.
type Service struct{ Store Store }

// NewService builds the service.
func NewService(s Store) *Service { return &Service{Store: s} }

// App is the assembled application.
type App struct{ Service *Service }

// NewApp builds the application.
func NewApp(s *Service) *App { return &App{Service: s} }

// AppGraph forgets loom.As[Store].
var AppGraph = loom.Graph[*App](
	loom.Provide(NewUserStore),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
