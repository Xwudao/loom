// Package lifecycle shows the two resource-management styles loom supports:
// constructor cleanups (used for rollback and teardown) and lifecycle hooks
// (used to start and stop long-lived services).

//go:generate go run github.com/Xwudao/loom/cmd/loom generate
package lifecycle

import (
	"context"

	"github.com/Xwudao/loom"
)

// Config holds runtime configuration.
type Config struct{ Addr string }

// DB is a resource released by a loom.Cleanup.
type DB struct {
	Addr   string
	Closed bool
}

// Cache is a resource released by a plain func() cleanup.
type Cache struct {
	Closed bool
}

// Server registers lifecycle hooks.
type Server struct {
	DB      *DB
	Cache   *Cache
	Running bool
}

// App is the assembled application.
type App struct {
	Server *Server
}

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{Addr: ":8080"} }

// NewDB opens a database and returns a context-aware cleanup.
func NewDB(cfg *Config) (*DB, loom.Cleanup, error) {
	db := &DB{Addr: cfg.Addr}
	return db, func(context.Context) error {
		db.Closed = true
		return nil
	}, nil
}

// NewCache returns a plain func() cleanup, which loom adapts.
func NewCache() (*Cache, func()) {
	cache := &Cache{}
	return cache, func() { cache.Closed = true }
}

// NewServer wires a server and registers start/stop hooks.
func NewServer(lc *loom.Lifecycle, db *DB, cache *Cache) *Server {
	server := &Server{DB: db, Cache: cache}
	lc.Append(loom.Hook{
		OnStart: func(context.Context) error {
			server.Running = true
			return nil
		},
		OnStop: func(context.Context) error {
			server.Running = false
			return nil
		},
	})
	return server
}

// NewApp builds the application.
func NewApp(server *Server) *App { return &App{Server: server} }

// AppGraph wires the application.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewDB),
	loom.Provide(NewCache),
	loom.Provide(NewServer),
	loom.Provide(NewApp),
)
