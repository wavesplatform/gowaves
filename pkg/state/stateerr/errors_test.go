package stateerr_test

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
)

func TestIsTxCommitmentError(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{stateerr.NewStateError(stateerr.TxValidationError, nil), false},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.TxValidationError, nil)), false},
		{errors.Wrap(stateerr.NewStateError(stateerr.TxValidationError, nil), "errors wrapped"), false},

		{stateerr.NewStateError(stateerr.TxCommitmentError, nil), true},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.TxCommitmentError, nil)), true},
		{errors.Wrap(stateerr.NewStateError(stateerr.TxCommitmentError, nil), "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, stateerr.IsTxCommitmentError(test.err))
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.TxValidationError, nil)), false},
		{errors.Wrap(stateerr.NewStateError(stateerr.TxValidationError, nil), "errors wrapped"), false},

		{proto.ErrNotFound, true},
		{fmt.Errorf("wrapped: %w", proto.ErrNotFound), true},
		{errors.Wrap(proto.ErrNotFound, "errors wrapped"), true},

		{keyvalue.ErrNotFound, true},
		{fmt.Errorf("wrapped: %w", keyvalue.ErrNotFound), true},
		{errors.Wrap(keyvalue.ErrNotFound, "errors wrapped"), true},

		{stateerr.NewStateError(stateerr.NotFoundError, nil), true},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.NotFoundError, nil)), true},
		{errors.Wrap(stateerr.NewStateError(stateerr.NotFoundError, nil), "errors wrapped"), true},

		{stateerr.NewStateError(stateerr.RetrievalError, nil), true},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.RetrievalError, nil)), true},
		{errors.Wrap(stateerr.NewStateError(stateerr.RetrievalError, nil), "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, stateerr.IsNotFound(test.err))
	}
}

func TestIsInvalidInput(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{stateerr.NewStateError(stateerr.TxValidationError, nil), false},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.TxValidationError, nil)), false},
		{errors.Wrap(stateerr.NewStateError(stateerr.TxValidationError, nil), "errors wrapped"), false},

		{stateerr.NewStateError(stateerr.InvalidInputError, nil), true},
		{fmt.Errorf("wrapped: %w", stateerr.NewStateError(stateerr.InvalidInputError, nil)), true},
		{errors.Wrap(stateerr.NewStateError(stateerr.InvalidInputError, nil), "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, stateerr.IsInvalidInput(test.err))
	}
}
