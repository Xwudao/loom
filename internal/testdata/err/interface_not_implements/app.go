package interfacenotimplements

import "github.com/Xwudao/loom"

// Greeter is an interface the provider does not satisfy.
type Greeter interface{ Greet() string }

// Thing does not implement Greeter.
type Thing struct{}

// App is the assembled application.
type App struct{ Thing *Thing }

// NewThing builds a Thing.
func NewThing() *Thing { return &Thing{} }

// NewApp builds the application.
func NewApp(t *Thing) *App { return &App{Thing: t} }

// AppGraph incorrectly binds *Thing to Greeter.
var AppGraph = loom.Graph[*App](
	loom.As[Greeter](NewThing),
	loom.Provide(NewApp),
)
