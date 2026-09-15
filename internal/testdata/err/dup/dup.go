package errdup

import "github.com/Xwudao/loom"

type DB struct{}
type App struct{ DB *DB }

func NewMySQL() *DB      { return &DB{} }
func NewPostgres() *DB   { return &DB{} }
func NewApp(db *DB) *App { return &App{DB: db} }

var AppGraph = loom.Graph[*App](
	loom.Provide(NewMySQL),
	loom.Provide(NewPostgres),
	loom.Provide(NewApp),
)
