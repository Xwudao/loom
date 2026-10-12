package interfaceambiguous

import "github.com/Xwudao/loom"

type Store interface{ Get() int }
type first struct{}
type second struct{}

func (*first) Get() int   { return 1 }
func (*second) Get() int  { return 2 }
func NewFirst() *first    { return &first{} }
func NewSecond() *second  { return &second{} }
func NewApp(s Store) *App { return &App{Store: s} }

type App struct{ Store Store }

var AppGraph = loom.Graph[*App](
	loom.Provide(NewFirst),
	loom.Provide(NewSecond),
	loom.Provide(NewApp),
)
