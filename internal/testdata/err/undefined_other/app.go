package undefinedother

import "github.com/Xwudao/loom"

// App is the assembled application.
type App struct{}

// NewApp builds the application.
func NewApp() *App { return &App{} }

var AppGraph = loom.Graph[*App](loom.Provide(NewApp))

// Run references the not-yet-generated InitApp (tolerated) and a genuine typo
// (reported).
func Run() {
	_, _, _ = InitApp()
	_ = misspelledName
}
