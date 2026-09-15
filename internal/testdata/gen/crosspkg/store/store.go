// Package store provides types from a different package than the graph.
package store

// Config holds storage configuration.
type Config struct{ DSN string }

// DB is a storage handle.
type DB struct{ DSN string }

// NewConfig returns storage configuration.
func NewConfig() *Config { return &Config{DSN: "store://"} }

// NewDB opens storage.
func NewDB(cfg *Config) *DB { return &DB{DSN: cfg.DSN} }
