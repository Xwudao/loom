// Package multigraph declares two graphs in one package.
package multigraph

import "github.com/Xwudao/loom"

// Config holds configuration.
type Config struct{ Name string }

// Server is one build target.
type Server struct{ Config *Config }

// Worker is another build target.
type Worker struct{ Config *Config }

// NewConfig loads configuration.
func NewConfig() *Config { return &Config{Name: "multi"} }

// NewServer builds a server.
func NewServer(cfg *Config) *Server { return &Server{Config: cfg} }

// NewWorker builds a worker.
func NewWorker(cfg *Config) *Worker { return &Worker{Config: cfg} }

// ServerGraph produces InitServer.
var ServerGraph = loom.Graph[*Server](
	loom.Provide(NewConfig),
	loom.Provide(NewServer),
)

// WorkerGraph produces InitWorker.
var WorkerGraph = loom.Graph[*Worker](
	loom.Provide(NewConfig),
	loom.Provide(NewWorker),
)
