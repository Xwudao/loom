// Package rollback verifies that a constructor failure releases everything
// that was already built, in reverse order.

//go:generate go run github.com/Xwudao/loom/cmd/loom generate
package rollback

import (
	"context"
	"errors"

	"github.com/Xwudao/loom"
)

// Config holds configuration.
type Config struct{}

// DB is released through a cleanup.
type DB struct{}

// Cache is released through a plain func() cleanup.
type Cache struct{}

// Server fails to construct, forcing a rollback.
type Server struct{}

// App is the assembled application.
type App struct{ Server *Server }

var events []string

func record(name string) { events = append(events, name) }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{} }

// NewDB opens a database.
func NewDB(*Config) (*DB, loom.Cleanup, error) {
	return &DB{}, func(context.Context) error {
		record("db")
		return nil
	}, nil
}

// NewCache builds a cache.
func NewCache() (*Cache, func(), error) {
	return &Cache{}, func() { record("cache") }, nil
}

// NewServer always fails; the two resources above must be rolled back.
func NewServer(*DB, *Cache) (*Server, error) {
	return nil, errors.New("server: boom")
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
