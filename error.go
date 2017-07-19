package bugsnack

import (
	"fmt"

	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func NewError(e interface{}) error {
	var err error

	switch e := e.(type) {
	case string:
		err = errors.New(e)
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	return errors.WithStack(err)
}
