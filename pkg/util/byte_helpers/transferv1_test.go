package byte_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransferWithSigBuilder(t *testing.T) {
	tr := newTransferWithSigBuilder().MustBuild()
	assert.Equal(t, "9ar4tAzhzw3gt6NPAG4hjb1Y3BeV85DAqfaeQPHmuiNG", tr.ID.String())
}
