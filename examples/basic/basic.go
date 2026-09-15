// Package basic is the smallest complete loom example: a linear chain of
// constructors with no errors, no cleanups, and no interfaces.

//go:generate go run github.com/Xwudao/loom/cmd/loom generate
package basic

import "github.com/Xwudao/loom"

// Config holds static application configuration.
type Config struct {
	DSN string
}

// DB is a database handle.
type DB struct {
	DSN string
}

// Repository exposes user persistence.
type Repository struct {
	DB *DB
}

// Service contains business logic.
type Service struct {
	Repo *Repository
}

// App is the assembled application.
type App struct {
	Service *Service
}

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{DSN: "memory://"} }

// NewDB opens a database.
func NewDB(cfg *Config) *DB { return &DB{DSN: cfg.DSN} }

// NewRepository builds a repository.
func NewRepository(db *DB) *Repository { return &Repository{DB: db} }

// NewService builds the service layer.
func NewService(repo *Repository) *Service { return &Service{Repo: repo} }

// NewApp builds the application.
func NewApp(svc *Service) *App { return &App{Service: svc} }

// AppGraph wires the application.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewDB),
	loom.Provide(NewRepository),
	loom.Provide(NewService),
	loom.Provide(NewApp),
)
