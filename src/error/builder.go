package error

import (
	"fmt"
	"sync"
)

type Builder struct {
	*builder
}

type builder struct {
	message string
	errors  []NestedError
	sync.Mutex
}

func NewBuilder(format string, args ...any) Builder {
	return Builder{&builder{message: fmt.Sprintf(format, args...)}}
}

// adding nil / nil is no-op,
// you may safely pass expressions returning error to it
func (b Builder) Add(err NestedError) Builder {
	if err != nil {
		b.Lock()
		// TODO: if err severity is higher than b.severity, update b.severity
		b.errors = append(b.errors, err)
		b.Unlock()
	}
	return b
}

func (b Builder) AddE(err error) Builder {
	return b.Add(From(err))
}

func (b Builder) Addf(format string, args ...any) Builder {
	return b.Add(errorf(format, args...))
}

// Build builds a NestedError based on the errors collected in the Builder.
//
// If there are no errors in the Builder, it returns a Nil() NestedError.
// Otherwise, it returns a NestedError with the message and the errors collected.
//
// Returns:
//   - NestedError: the built NestedError.
func (b Builder) Build() NestedError {
	if len(b.errors) == 0 {
		return nil
	} else if len(b.errors) == 1 {
		return b.errors[0]
	}
	return Join(b.message, b.errors...)
}

func (b Builder) To(ptr *NestedError) {
	if ptr == nil {
		return
	} else if *ptr == nil {
		*ptr = b.Build()
	} else {
		(*ptr).With(b.Build())
	}
}

func (b Builder) HasError() bool {
	return len(b.errors) > 0
}
