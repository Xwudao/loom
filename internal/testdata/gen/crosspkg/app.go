package crosspkg

import (
	"github.com/Xwudao/loom"
	"github.com/Xwudao/loom/internal/testdata/gen/crosspkg/store"
)

// Repository stores records.
type Repository struct{ DB *store.DB }

// App is the assembled application.
type App struct{ Repo *Repository }

// NewRepository builds a repository.
func NewRepository(db *store.DB) *Repository { return &Repository{DB: db} }

// NewApp builds the application.
func NewApp(repo *Repository) *App { return &App{Repo: repo} }

// AppGraph uses providers from another package.
var AppGraph = loom.Graph[*App](
	loom.Provide(store.NewConfig),
	loom.Provide(store.NewDB),
	loom.Provide(NewRepository),
	loom.Provide(NewApp),
)
