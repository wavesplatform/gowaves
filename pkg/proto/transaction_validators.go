package proto

import "github.com/wavesplatform/gowaves/pkg/errs"

func ValidatePositiveAmount(amount int64, of string) error {
	if !(amount > 0) {
		return errs.NewNonPositiveAmount(amount, of)
	}
	return nil
}
