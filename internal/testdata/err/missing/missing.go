package errmissing

import "github.com/Xwudao/loom"

type Config struct{}
type DB struct{}
type UserStore interface{ Find(int64) error }
type UserService struct{}
type App struct{}

func NewConfig() *Config                    { return &Config{} }
func NewDB(*Config) *DB                     { return &DB{} }
func NewUserService(UserStore) *UserService { return &UserService{} }
func NewApp(*UserService) *App              { return &App{} }

var AppGraph = loom.Graph[*App](
	loom.Provide(NewConfig),
	loom.Provide(NewDB),
	loom.Provide(NewUserService),
	loom.Provide(NewApp),
)
