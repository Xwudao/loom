package full

import (
	"context"
	"errors"
)

// User is a domain entity.
type User struct {
	ID   int64
	Name string
}

// Article is a domain entity.
type Article struct {
	ID    int64
	Title string
}

// Repository is generic over the entity it stores.
type Repository[T any] struct {
	DB *Database
}

// NewRepository builds a repository for T.
func NewRepository[T any](db *Database) *Repository[T] { return &Repository[T]{DB: db} }

// UserStore is the persistence boundary the services depend on.
type UserStore interface {
	Find(ctx context.Context, id int64) (*User, error)
}

type userStore struct {
	repo *Repository[User]
	log  *Logger
}

// Find loads a user.
func (s *userStore) Find(_ context.Context, id int64) (*User, error) {
	if id != 1 {
		return nil, errors.New("user not found")
	}
	return &User{ID: 1, Name: "ada"}, nil
}

// NewUserStore returns the concrete implementation of UserStore.
func NewUserStore(repo *Repository[User], log *Logger) *userStore {
	return &userStore{repo: repo, log: log}
}

// UserService contains user business logic.
type UserService struct {
	Store UserStore
	Log   *Logger
}

// NewUserService builds the user service.
func NewUserService(store UserStore, log *Logger) *UserService {
	return &UserService{Store: store, Log: log}
}

// ArticleService contains article business logic.
type ArticleService struct {
	Repo *Repository[Article]
	Log  *Logger
}

// NewArticleService builds the article service.
func NewArticleService(repo *Repository[Article], log *Logger) *ArticleService {
	return &ArticleService{Repo: repo, Log: log}
}
