// Package db is named after its own type, which is the case that pushes the
// generator to a package-qualified variable name.
package db

// DB is a database handle.
type DB struct{}

// NewDB opens the database.
func NewDB() *DB { return &DB{} }
