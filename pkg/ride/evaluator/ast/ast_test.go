package ast

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestBooleanExpr_Eq(t *testing.T) {
	b1 := NewBoolean(true)
	b2 := NewBoolean(true)
	b3 := NewBoolean(false)
	eq1, _ := b1.Eq(b2)
	assert.True(t, eq1)
	eq2, _ := b1.Eq(b3)
	assert.False(t, eq2)
	eq3, _ := b1.Eq(NewLong(5))
	assert.False(t, eq3)
}

func TestDataEntryListExpr_Get(t *testing.T) {
	var d []proto.DataEntry
	d = append(d, &proto.IntegerDataEntry{
		Key:   "integer",
		Value: 100500,
	})
	lst := NewDataEntryList(d)
	assert.Equal(t, NewLong(100500), lst.Get("integer", proto.DataInteger))
}
