package util

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestTransferWithSigBuilder(t *testing.T) {
	tr := NewTransferWithSigBuilder().MustBuild()
	assert.Equal(t, "9ar4tAzhzw3gt6NPAG4hjb1Y3BeV85DAqfaeQPHmuiNG", tr.ID.String())
}
