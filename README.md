cpdocs2impl
=========

cpdocs2impl is used to copy method comments from Go interfaces to 
implementations. It can replace existing comments, or only add 
missing comments.


Install
=======

```sh
go get -u github.com/tzneal/cpdocs2impl/cmd/cpdocs2impl
```

Usage
=====

```
Usage: ./cpdocs2impl [OPTION]... [PACKAGE]...
Copy docs from interfaces to implementations
  -imports
    	consider interfaces from imports as well (slower)
  -replace
    	replace any existing documentation
```  

cpdocs2iml uses the common technique of unnamed variable declarations that ensure 
an implementation implements an interface to identify cases where documentation 
should be copied from an interface to an implementation.  As an example it can modify 
source from:

```go
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
```

into this:

```go
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


// Foo does some foo stuff
func (c *Concrete) Foo() {} 
// Bar is different
func (c *Concrete) Bar() {}
```

