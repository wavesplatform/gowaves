package proto

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/errs"
)

func ValidatePositiveAmount(amount int64, of string) error {
	if !(amount > 0) {
		return errs.NewNonPositiveAmount(amount, of)
	}
	return nil
}

func ValidateNonNegativeAmount(amount int64, of string) error {
	if amount < 0 {
		return errs.NewTxValidationError(fmt.Sprintf("negative amount %d %s", amount, of))
	}
	return nil
}
