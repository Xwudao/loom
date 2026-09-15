package modules

import (
	"github.com/Xwudao/loom"
	"github.com/Xwudao/loom/internal/testdata/gen/modules/infra"
)

// Repository stores records.
type Repository struct{ DB *infra.DB }

// Service depends on storage and cache.
type Service struct {
	Repo  *Repository
	Cache *infra.Cache
}

// App is the assembled application.
type App struct{ Service *Service }

// NewRepository builds a repository.
func NewRepository(db *infra.DB) *Repository { return &Repository{DB: db} }

// NewService builds the service.
func NewService(repo *Repository, cache *infra.Cache) *Service {
	return &Service{Repo: repo, Cache: cache}
}

// NewApp builds the application.
func NewApp(svc *Service) *App { return &App{Service: svc} }

// LocalModule nests the infra module.
var LocalModule = loom.Module(
	infra.Module,
	loom.Provide(NewRepository),
)

// AppGraph includes infra.Module twice: modules are sets, so that is a no-op.
var AppGraph = loom.Graph[*App](
	LocalModule,
	infra.Module,
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
