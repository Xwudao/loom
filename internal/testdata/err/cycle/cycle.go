package errcycle

import "github.com/Xwudao/loom"

type A struct{}
type B struct{}
type C struct{}

func NewA(*B) *A { return &A{} }
func NewB(*C) *B { return &B{} }
func NewC(*A) *C { return &C{} }

var AGraph = loom.Graph[*A](
	loom.Provide(NewA),
	loom.Provide(NewB),
	loom.Provide(NewC),
)
