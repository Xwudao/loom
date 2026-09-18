// Package graphbindsamefile verifies that a graph may bind a constructor a
// module provides when both are declared in the same file, and that the order
// of the As entry and the module does not matter.
package graphbindsamefile

import "github.com/Xwudao/loom"

// Store is the interface NewStore's result is bound to.
type Store interface{ Get() string }

type store struct{}

func (s *store) Get() string { return "store" }

// NewStore builds the implementation.
func NewStore() *store { return &store{} }

// App consumes the binding.
type App struct{ Store Store }

// NewApp builds the application.
func NewApp(s Store) *App { return &App{Store: s} }

// StoreModule provides NewStore without exposing the interface: the binding
// must live in the graph, because only the graph knows what it needs.
var StoreModule = loom.Module(
	loom.Provide(NewStore),
)

// AppGraph lists the binding before the module that provides the constructor.
var AppGraph = loom.Graph[*App](
	loom.As[Store](NewStore),
	StoreModule,
	loom.Provide(NewApp),
)
