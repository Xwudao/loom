// Package iface declares the interface a graph exposes, and imports store to
// mirror the real layout that makes a module-level binding impossible: if
// store had to import iface, the two packages would form a cycle.
package iface

import "github.com/Xwudao/loom/internal/testdata/gen/graph_bind/store"

// Finisher is the narrow interface a consumer depends on.
type Finisher interface {
	// Get reads a record.
	Get() string
}

// Consumer needs only the interface.
type Consumer struct{ Store Finisher }

// NewConsumer builds the consumer.
func NewConsumer(f Finisher) *Consumer { return &Consumer{Store: f} }

// UsesStore keeps the import of store meaningful, the way a package that
// defines an interface next to its consumer usually does.
func UsesStore(s *store.Store) string { return s.Get() }
