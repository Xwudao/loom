package asnotinterface

import "github.com/Xwudao/loom"

// Thing is a concrete type.
type Thing struct{}

// App is the assembled application.
type App struct{ Thing *Thing }

// NewThing builds a Thing.
func NewThing() *Thing { return &Thing{} }

// NewApp builds the application.
func NewApp(t *Thing) *App { return &App{Thing: t} }

// AppGraph passes a non-interface type argument to loom.As.
var AppGraph = loom.Graph[*App](
	loom.As[Thing](NewThing),
	loom.Provide(NewApp),
)
