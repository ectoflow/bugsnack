package bugsnack

import (
	"errors"
	"fmt"

	"github.com/fromatob/bugsnack/internal/stack"
)

// Error is an error with an attached stacktrace
type Error struct {
	Err        error
	Stacktrace stack.CallStack
}

// Error returns the underlying error's message
func (b *Error) Error() string {
	return b.Err.Error()
}

// New makes an Error from the given value. If it's already an Error
// then it will be used directly. Possible values are: string, error, *Error
// Any other types will be processed by fallback and converted to an error with
// its string representation
func NewError(e interface{}) *Error {
	var err error
	// Arrays start at 1 ¯\_(ツ)_/¯
	stacktrace := stack.Trace()[1:]

	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	case string:
		err = errors.New(e)
	default:
		err = fmt.Errorf("%v", e)
	}

	return &Error{
		Err:        err,
		Stacktrace: stacktrace,
	}
}
