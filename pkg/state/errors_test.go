package state

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestIsTxCommitmentError(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{StateError{errorType: TxValidationError}, false},
		{fmt.Errorf("wrapped: %w", StateError{errorType: TxValidationError}), false},
		{errors.Wrap(StateError{errorType: TxValidationError}, "errors wrapped"), false},

		{StateError{errorType: TxCommitmentError}, true},
		{fmt.Errorf("wrapped: %w", StateError{errorType: TxCommitmentError}), true},
		{errors.Wrap(StateError{errorType: TxCommitmentError}, "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, IsTxCommitmentError(test.err))
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{fmt.Errorf("wrapped: %w", StateError{errorType: TxValidationError}), false},
		{errors.Wrap(StateError{errorType: TxValidationError}, "errors wrapped"), false},

		{proto.ErrNotFound, true},
		{fmt.Errorf("wrapped: %w", proto.ErrNotFound), true},
		{errors.Wrap(proto.ErrNotFound, "errors wrapped"), true},

		{keyvalue.ErrNotFound, true},
		{fmt.Errorf("wrapped: %w", keyvalue.ErrNotFound), true},
		{errors.Wrap(keyvalue.ErrNotFound, "errors wrapped"), true},

		{StateError{errorType: NotFoundError}, true},
		{fmt.Errorf("wrapped: %w", StateError{errorType: NotFoundError}), true},
		{errors.Wrap(StateError{errorType: NotFoundError}, "errors wrapped"), true},

		{StateError{errorType: RetrievalError}, true},
		{fmt.Errorf("wrapped: %w", StateError{errorType: RetrievalError}), true},
		{errors.Wrap(StateError{errorType: RetrievalError}, "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, IsNotFound(test.err))

	}
}

func TestIsInvalidInput(t *testing.T) {
	tests := []struct {
		err    error
		result bool
	}{
		{nil, false},
		{fmt.Errorf("some err"), false},
		{StateError{errorType: TxValidationError}, false},
		{fmt.Errorf("wrapped: %w", StateError{errorType: TxValidationError}), false},
		{errors.Wrap(StateError{errorType: TxValidationError}, "errors wrapped"), false},

		{StateError{errorType: InvalidInputError}, true},
		{fmt.Errorf("wrapped: %w", StateError{errorType: InvalidInputError}), true},
		{errors.Wrap(StateError{errorType: InvalidInputError}, "errors wrapped"), true},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, IsInvalidInput(test.err))
	}
}
