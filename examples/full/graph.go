//go:generate go run github.com/Xwudao/loom/cmd/loom generate

package full

import "github.com/Xwudao/loom"

// DatabaseModule groups everything that talks to storage. Modules are sets:
// including one twice has no effect.
var DatabaseModule = loom.Module(
	loom.Provide(NewDatabase),
	loom.Provide(NewRepository[User]),
	loom.Provide(NewRepository[Article]),
)

// ServiceModule groups the service layer. It nests DatabaseModule.
var ServiceModule = loom.Module(
	DatabaseModule,
	loom.As[UserStore](NewUserStore),
	loom.Provide(NewUserService),
	loom.Provide(NewArticleService),
)

// AppGraph wires the application.
var AppGraph = loom.Graph[*Application](
	loom.WithContext(),
	loom.Provide(NewConfig),
	loom.Provide(NewLogger),
	ServiceModule,
	loom.Provide(NewHTTPServer),
	loom.Provide(NewApplication),
	loom.Supply(AppName("full")),
)

// WorkerGraph is a second target in the same package. It produces InitWorker.
var WorkerGraph = loom.Graph[*Worker](
	loom.Provide(NewConfig),
	loom.Provide(NewLogger),
	loom.Provide(NewDatabase),
	loom.Provide(NewWorker),
)
