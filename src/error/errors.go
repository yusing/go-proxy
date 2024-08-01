package error

var (
	ErrAlreadyStarted = new("already started")
	ErrNotStarted     = new("not started")
	ErrInvalid        = new("invalid")
	ErrUnsupported    = new("unsupported")
	ErrNotExists      = new("does not exist")
	ErrDuplicated     = new("duplicated")
)

func Failure(what string) NestedError {
	return errorf("%s failed", what)
}

func Invalid(subject, what any) NestedError {
	return errorf("%w %s: %q", ErrInvalid, subject, what)
}

func Unsupported(subject, what any) NestedError {
	return errorf("%w %s: %q", ErrUnsupported, subject, what)
}

func NotExists(subject, what any) NestedError {
	return errorf("%s %w: %q", subject, ErrNotExists, what)
}

func Duplicated(subject, what any) NestedError {
	return errorf("%w %s: %q", ErrDuplicated, subject, what)
}
