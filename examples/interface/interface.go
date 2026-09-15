// Package iface demonstrates explicit interface binding with loom.As.

//go:generate go run github.com/Xwudao/loom/cmd/loom generate
package iface

import (
	"context"
	"errors"

	"github.com/Xwudao/loom"
)

// User is a domain entity.
type User struct {
	ID   int64
	Name string
}

// UserStore is the persistence boundary the service depends on.
type UserStore interface {
	Find(ctx context.Context, id int64) (*User, error)
}

// UserService depends on the interface, not the implementation.
type UserService struct {
	Store UserStore
}

// App is the assembled application.
type App struct {
	Service *UserService
}

// DB is a placeholder database handle.
type DB struct {
	Users map[int64]*User
}

type userStore struct {
	db *DB
}

// Find loads a user.
func (s *userStore) Find(_ context.Context, id int64) (*User, error) {
	u, ok := s.db.Users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

// NewDB opens the database and seeds one user.
func NewDB() *DB {
	return &DB{Users: map[int64]*User{1: {ID: 1, Name: "ada"}}}
}

// NewUserStore returns the concrete store; note it is unexported, so only the
// interface escapes the package.
func NewUserStore(db *DB) *userStore { return &userStore{db: db} }

// NewUserService builds the service.
func NewUserService(store UserStore) *UserService { return &UserService{Store: store} }

// NewApp builds the application.
func NewApp(svc *UserService) *App { return &App{Service: svc} }

// AppGraph wires *userStore into UserStore via loom.As.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewDB),
	loom.As[UserStore](NewUserStore),
	loom.Provide(NewUserService),
	loom.Provide(NewApp),
)
