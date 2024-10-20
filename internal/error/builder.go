package error

import (
	"errors"
	"fmt"
	"sync"
)

type Builder struct {
	*builder
}

type builder struct {
	message string
	errors  []Error
	sync.Mutex
}

func NewBuilder(format string, args ...any) Builder {
	if len(args) > 0 {
		return Builder{&builder{message: fmt.Sprintf(format, args...)}}
	}
	return Builder{&builder{message: format}}
}

// Add adds an error to the Builder.
//
// adding nil is no-op,
//
// flatten is a boolean flag to flatten the NestedError.
func (b Builder) Add(err Error, flatten ...bool) {
	if err != nil {
		b.Lock()
		if len(flatten) > 0 && flatten[0] {
			for _, e := range err.extras {
				b.errors = append(b.errors, &e)
			}
		} else {
			b.errors = append(b.errors, err)
		}
		b.Unlock()
	}
}

func (b Builder) AddE(err error) {
	b.Add(From(err))
}

func (b Builder) Addf(format string, args ...any) {
	if len(args) > 0 {
		b.Add(errorf(format, args...))
	} else {
		b.AddE(errors.New(format))
	}
}

func (b Builder) AddRange(errs ...Error) {
	b.Lock()
	defer b.Unlock()
	for _, err := range errs {
		b.errors = append(b.errors, err)
	}
}

func (b Builder) AddRangeE(errs ...error) {
	b.Lock()
	defer b.Unlock()
	for _, err := range errs {
		b.errors = append(b.errors, From(err))
	}
}

// Build builds a NestedError based on the errors collected in the Builder.
//
// If there are no errors in the Builder, it returns a Nil() NestedError.
// Otherwise, it returns a NestedError with the message and the errors collected.
//
// Returns:
//   - NestedError: the built NestedError.
func (b Builder) Build() Error {
	if len(b.errors) == 0 {
		return nil
	}
	return Join(b.message, b.errors...)
}

func (b Builder) To(ptr *Error) {
	switch {
	case ptr == nil:
		return
	case *ptr == nil:
		*ptr = b.Build()
	default:
		(*ptr).extras = append((*ptr).extras, *b.Build())
	}
}

func (b Builder) String() string {
	return b.Build().String()
}

func (b Builder) HasError() bool {
	return len(b.errors) > 0
}
