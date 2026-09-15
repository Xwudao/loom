package importconflict

import (
	"github.com/Xwudao/loom"
	amodel "github.com/Xwudao/loom/internal/testdata/gen/importconflict/a/model"
	bmodel "github.com/Xwudao/loom/internal/testdata/gen/importconflict/b/model"
)

// Repository is generic over the item it stores.
type Repository[T any] struct{ Item T }

// App holds both repositories.
type App struct {
	A *Repository[*amodel.Item]
	B *Repository[*bmodel.Item]
}

// NewItemA builds an a model.
func NewItemA() *amodel.Item { return &amodel.Item{A: 1} }

// NewItemB builds a b model.
func NewItemB() *bmodel.Item { return &bmodel.Item{B: 2} }

// NewRepository builds a repository.
func NewRepository[T any](item T) *Repository[T] { return &Repository[T]{Item: item} }

// NewApp builds the application.
func NewApp(a *Repository[*amodel.Item], b *Repository[*bmodel.Item]) *App {
	return &App{A: a, B: b}
}

// AppGraph forces import aliasing because both packages are named model.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewItemA),
	loom.Provide(NewItemB),
	loom.Provide(NewRepository[*amodel.Item]),
	loom.Provide(NewRepository[*bmodel.Item]),
	loom.Provide(NewApp),
)
