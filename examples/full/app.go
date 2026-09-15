package full

import (
	"context"

	"github.com/Xwudao/loom"
)

// HTTPServer is a long-lived service with start/stop hooks.
type HTTPServer struct {
	Addr    string
	Log     *Logger
	Users   *UserService
	Running bool
}

// NewHTTPServer builds the server and registers its lifecycle hooks.
func NewHTTPServer(lc *loom.Lifecycle, cfg *Config, log *Logger, users *UserService) *HTTPServer {
	server := &HTTPServer{Addr: cfg.Addr, Log: log, Users: users}
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

// Application is the assembled program.
type Application struct {
	Name     AppName
	Config   *Config
	Logger   *Logger
	Database *Database
	Server   *HTTPServer
	Users    *UserService
	Articles *ArticleService
}

// NewApplication builds the application.
func NewApplication(
	name AppName,
	cfg *Config,
	log *Logger,
	db *Database,
	server *HTTPServer,
	users *UserService,
	articles *ArticleService,
) *Application {
	return &Application{
		Name:     name,
		Config:   cfg,
		Logger:   log,
		Database: db,
		Server:   server,
		Users:    users,
		Articles: articles,
	}
}

// Worker is a second build target that shares providers with the application.
type Worker struct {
	DB  *Database
	Log *Logger
}

// NewWorker builds the worker.
func NewWorker(db *Database, log *Logger) *Worker { return &Worker{DB: db, Log: log} }
