package byte_helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransaction(t *testing.T) {
	assert.NotEmpty(t, TransferV1.TransactionBytes)
	assert.NotEmpty(t, TransferV1.Transaction)
	assert.NotEmpty(t, TransferV1.MessageBytes)
}
