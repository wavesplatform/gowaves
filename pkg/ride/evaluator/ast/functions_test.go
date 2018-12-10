package ast

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNativeSumLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeSumLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(9), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeSumLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeSubLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeSubLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(1), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeSubLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeMulLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(2)}
	rs, err := NativeMulLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(10), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeMulLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeGeLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(5)}
	rs, err := NativeGeLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeGeLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeGtLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeGtLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeGtLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeDivLong(t *testing.T) {
	params1 := Exprs{NewLong(9), NewLong(2)}
	rs, err := NativeDivLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(4), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeDivLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestUserAddressFromString(t *testing.T) {
	params1 := NewString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	rs, err := UserAddressFromString(newEmptyScope(), NewExprs(params1))
	require.NoError(t, err)
	addr, _ := NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	assert.Equal(t, addr, rs)
}
