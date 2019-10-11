package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBooleanExpr_Eq(t *testing.T) {
	b1 := NewBoolean(true)
	b2 := NewBoolean(true)
	b3 := NewBoolean(false)
	eq1 := b1.Eq(b2)
	assert.True(t, eq1)
	eq2 := b1.Eq(b3)
	assert.False(t, eq2)
	eq3 := b1.Eq(NewLong(5))
	assert.False(t, eq3)
}

func TestBuyExpr_Eq(t *testing.T) {
	be := &BuyExpr{}
	eq := be.Eq(&BuyExpr{})
	require.True(t, eq)
}

func TestSellExpr_Eq(t *testing.T) {
	se := &SellExpr{}
	eq := se.Eq(&SellExpr{})
	require.True(t, eq)
}

func TestBlockHeaderExprIsObject(t *testing.T) {
	var e Expr = NewBlockHeader(nil)
	_ = e.(Getable)
}

func TestAttachedPaymentExprIsObject(t *testing.T) {
	var e Expr = NewAttachedPaymentExpr(nil, nil)
	_ = e.(Getable)
}
