package error

import (
	"fmt"
	"sync"
)

type Builder struct {
	message string
	errors  []error
	sync.Mutex
}

func NewBuilder(format string, args ...any) *Builder {
	return &Builder{message: fmt.Sprintf(format, args...)}
}

func (b *Builder) Add(err error) *Builder {
	if err != nil {
		b.Lock()
		b.errors = append(b.errors, err)
		b.Unlock()
	}
	return b
}

func (b *Builder) Addf(format string, args ...any) *Builder {
	return b.Add(fmt.Errorf(format, args...))
}

// Build builds a NestedError based on the errors collected in the Builder.
//
// If there are no errors in the Builder, it returns a Nil() NestedError.
// Otherwise, it returns a NestedError with the message and the errors collected.
//
// Returns:
//   - NestedError: the built NestedError.
func (b *Builder) Build() NestedError {
	if len(b.errors) == 0 {
		return Nil()
	}
	return Join(b.message, b.errors...)
}
