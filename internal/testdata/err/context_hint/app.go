package contexthint

import (
	"context"

	"github.com/Xwudao/loom"
)

// DB needs a context.
type DB struct{}

// App is the assembled application.
type App struct{ DB *DB }

// NewDB opens a database with a context.
func NewDB(ctx context.Context) *DB { return &DB{} }

// NewApp builds the application.
func NewApp(db *DB) *App { return &App{DB: db} }

// AppGraph forgets loom.WithContext, so context.Context cannot be injected.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewDB),
	loom.Provide(NewApp),
)
