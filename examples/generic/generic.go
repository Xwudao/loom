// Package generic shows that type parameters are first-class dependency keys:
// Repository[User] and Repository[Article] are distinct types that share one
// database handle.

//go:generate go run github.com/Xwudao/loom/cmd/loom generate
package generic

import "github.com/Xwudao/loom"

// User is a domain entity.
type User struct{ Name string }

// Article is a domain entity.
type Article struct{ Title string }

// DB is shared by both repositories.
type DB struct {
	Opens int
}

// Repository is generic over the entity it stores.
type Repository[T any] struct {
	DB *DB
}

// UserService depends on Repository[User] specifically.
type UserService struct{ Repo *Repository[User] }

// ArticleService depends on Repository[Article] specifically.
type ArticleService struct{ Repo *Repository[Article] }

// App depends on both services.
type App struct {
	Users    *UserService
	Articles *ArticleService
}

// NewDB opens a database.
func NewDB() *DB { return &DB{Opens: 1} }

// NewRepository builds a repository for T.
func NewRepository[T any](db *DB) *Repository[T] { return &Repository[T]{DB: db} }

// NewUserService builds the user service.
func NewUserService(repo *Repository[User]) *UserService { return &UserService{Repo: repo} }

// NewArticleService builds the article service.
func NewArticleService(repo *Repository[Article]) *ArticleService {
	return &ArticleService{Repo: repo}
}

// NewApp builds the application.
func NewApp(users *UserService, articles *ArticleService) *App {
	return &App{Users: users, Articles: articles}
}

// AppGraph wires the application.
var AppGraph = loom.Graph[*App](
	loom.Provide(NewDB),
	loom.Provide(NewRepository[User]),
	loom.Provide(NewRepository[Article]),
	loom.Provide(NewUserService),
	loom.Provide(NewArticleService),
	loom.Provide(NewApp),
)
