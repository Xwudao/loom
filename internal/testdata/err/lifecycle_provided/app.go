package lifecycleprovided

import "github.com/Xwudao/loom"

// App is the assembled application.
type App struct{ Lifecycle *loom.Lifecycle }

// NewLifecycle tries to provide a type that loom owns.
func NewLifecycle() *loom.Lifecycle { return loom.NewLifecycle() }

// NewApp builds the application.
func NewApp(lc *loom.Lifecycle) *App { return &App{Lifecycle: lc} }

// AppGraph provides *loom.Lifecycle, which loom reserves.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewLifecycle),
	loom.Provide(NewApp),
)
