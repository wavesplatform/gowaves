package errs

import (
	"fmt"

	"github.com/pkg/errors"
)

type IExtend interface {
	Extend(message string) error
}

func Extend(err error, message string) error {
	if ex, ok := err.(IExtend); ok {
		return ex.Extend(message)
	}
	return errors.Wrap(err, message)
}

func fmtExtend(self error, message string) string {
	return fmt.Sprintf("%s: %s", message, self)
}
