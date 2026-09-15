// Package full is the end-to-end loom example: generics, an interface binding,
// modules, supply, lifecycle hooks, and constructor cleanups in one graph.
package full

import (
	"context"

	"github.com/Xwudao/loom"
)

// Config holds static configuration.
type Config struct {
	Addr     string
	DSN      string
	LogLevel string
}

// AppName is an existing value supplied to the graph rather than constructed.
type AppName string

// NewConfig loads configuration.
func NewConfig() *Config {
	return &Config{Addr: ":8080", DSN: "postgres://localhost/app", LogLevel: "info"}
}

// Logger is a tiny structured logger.
type Logger struct {
	Level string
}

// NewLogger builds the logger from configuration.
func NewLogger(cfg *Config) *Logger { return &Logger{Level: cfg.LogLevel} }

// Database is a resource owned by a constructor cleanup.
type Database struct {
	DSN    string
	Closed bool
}

// NewDatabase dials the database. It needs a context, which the graph makes
// available with loom.WithContext().
func NewDatabase(cfg *Config) (*Database, loom.Cleanup, error) {
	db := &Database{DSN: cfg.DSN}
	return db, func(context.Context) error {
		db.Closed = true
		return nil
	}, nil
}
