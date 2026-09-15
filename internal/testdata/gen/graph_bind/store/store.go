// Package store owns the concrete storage provider.
package store

import "github.com/Xwudao/loom"

// Store is the implementation.
type Store struct{ Name string }

// NewStore builds the implementation.
func NewStore() *Store { return &Store{Name: "store"} }

// Get reads a record.
func (s *Store) Get() string { return s.Name }

// Module is the storage provider set.
var Module = loom.Module(
	loom.Provide(NewStore),
)
