package graphbind

import (
	"github.com/Xwudao/loom"

	"github.com/Xwudao/loom/internal/testdata/gen/graph_bind/iface"
	"github.com/Xwudao/loom/internal/testdata/gen/graph_bind/store"
)

// App is the assembled application.
type App struct{ Consumer *iface.Consumer }

// NewApp builds the application.
func NewApp(c *iface.Consumer) *App { return &App{Consumer: c} }

// AppGraph exposes store.Module's constructor as iface.Finisher without
// re-declaring it. The binding has to live here rather than in store.Module,
// because iface imports store.
var AppGraph = loom.Graph[*App](
	store.Module,
	loom.As[iface.Finisher](store.NewStore),
	loom.Provide(iface.NewConsumer),
	loom.Provide(NewApp),
)
