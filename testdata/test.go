package testdata

type Fooer interface {
	// Foo does some foo stuff
	Foo()
	// Bar is different
	Bar()
}

var _ Fooer = (*Concrete)(nil)

type Concrete struct {
}

func (c *Concrete) Foo() {}
func (c *Concrete) Bar() {}
