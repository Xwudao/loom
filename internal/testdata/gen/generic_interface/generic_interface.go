// Package genericinterface binds an instantiated generic interface.
package genericinterface

import (
	"context"

	"github.com/Xwudao/loom"
)

// User is a domain entity.
type User struct{ Name string }

// Store is a generic persistence interface.
type Store[T any] interface {
	Get(ctx context.Context, id int64) (T, error)
}

type userStore struct{}

func (s *userStore) Get(context.Context, int64) (User, error) { return User{Name: "ada"}, nil }

// NewUserStore returns the concrete implementation.
func NewUserStore() *userStore { return &userStore{} }

// Service depends on Store[User] specifically.
type Service struct{ Store Store[User] }

// NewService builds the service.
func NewService(s Store[User]) *Service { return &Service{Store: s} }

// App is the assembled application.
type App struct{ Service *Service }

// NewApp builds the application.
func NewApp(s *Service) *App { return &App{Service: s} }

// AppGraph binds *userStore to the instantiated interface Store[User].
var AppGraph = loom.Graph[*App](
	loom.As[Store[User]](NewUserStore),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
