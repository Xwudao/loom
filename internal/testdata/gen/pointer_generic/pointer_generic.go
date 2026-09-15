// Package pointergeneric distinguishes Repository[*User] from Repository[User].
package pointergeneric

import "github.com/Xwudao/loom"

// User is a domain entity.
type User struct{ Name string }

// Repository is generic over the item type.
type Repository[T any] struct{ Zero T }

// NewRepository builds a repository.
func NewRepository[T any]() *Repository[T] { return &Repository[T]{} }

// App needs the pointer, value, and nested instantiations.
type App struct {
	Ptr    *Repository[*User]
	Value  *Repository[User]
	Nested *Repository[Repository[User]]
}

// NewApp builds the application.
func NewApp(ptr *Repository[*User], value *Repository[User], nested *Repository[Repository[User]]) *App {
	return &App{Ptr: ptr, Value: value, Nested: nested}
}

// AppGraph wires three distinct instantiations.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewRepository[*User]),
	loom.Provide(NewRepository[User]),
	loom.Provide(NewRepository[Repository[User]]),
	loom.Provide(NewApp),
)
