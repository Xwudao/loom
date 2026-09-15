// Package infra declares a module that is reused from the graph package.
package infra

import "github.com/Xwudao/loom"

// DB is a storage handle.
type DB struct{}

// Cache is an in-memory cache.
type Cache struct{}

// NewDB opens storage.
func NewDB() *DB { return &DB{} }

// NewCache builds a cache.
func NewCache() *Cache { return &Cache{} }

// Module groups the infrastructure providers.
var Module = loom.Module(
	loom.Provide(NewDB),
	loom.Provide(NewCache),
)
