package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

func ctx(s stack) Context {
	return NewContext(s, nil, 'W')
}

func TestGteLong(t *testing.T) {
	s := NewStack()
	s.PushL(10)
	s.PushL(6)

	err := GteLong(ctx(s))
	require.NoError(t, err)

	require.NoError(t, with(ctx(s), func(i bool) error {
		require.True(t, i)
		return nil
	}))

	s.PushL(5)
	s.PushL(5)
	_ = GteLong(ctx(s))
	require.Equal(t, ast.NewBoolean(true), s.Pop())

	s.PushL(4)
	s.PushL(5)
	_ = GteLong(ctx(s))
	require.Equal(t, ast.NewBoolean(false), s.Pop())
}

func TestIsInstanceOf(t *testing.T) {
	s := NewStack()
	s.Push(ast.NewLong(5))
	s.Push(ast.NewString(ast.NewLong(5).InstanceOf()))
	require.NoError(t, IsInstanceOf(ctx(s)))
	require.Equal(t, ast.NewBoolean(true), s.Pop())

	s.Push(ast.NewString(""))
	s.Push(ast.NewString(ast.NewLong(5).InstanceOf()))
	require.NoError(t, IsInstanceOf(ctx(s)))
	require.Equal(t, ast.NewBoolean(false), s.Pop())
}

func TestUserAddress(t *testing.T) {
	a := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	s := NewStack()
	addr, err := proto.NewAddressFromString(a)
	require.NoError(t, err)
	s.Push(ast.NewBytes(addr.Bytes()))

	err = UserAddress(ctx(s))
	require.NoError(t, err)
	require.Equal(t, ast.NewAddressFromProtoAddress(addr), s.Pop())
}

func TestUserAddressFromString(t *testing.T) {
	t.Run("error if empty", func(t *testing.T) {
		s := NewStack()
		sc := ctx(s)
		err := UserAddressFromString(sc)
		require.Error(t, err)
	})
	t.Run("error if wrong type, passed Long instead String", func(t *testing.T) {
		s := NewStack()
		s.Push(ast.NewLong(12345))
		sc := ctx(s)
		err := UserAddressFromString(sc)
		require.Error(t, err)
	})
	t.Run("invalid string passed, expecting Unit", func(t *testing.T) {
		s := NewStack()
		s.Push(ast.NewString("fake address"))
		sc := ctx(s)
		err := UserAddressFromString(sc)
		require.NoError(t, err)
		require.Equal(t, sc.Pop(), ast.NewUnit())
	})
	t.Run("correct address provided, but wrong chainID. Current mainnet, provided stagenet", func(t *testing.T) {
		s := NewStack()
		s.Push(ast.NewString("3MpV2xvvcWUcv8FLDKJ9ZRrQpEyF8nFwRUM"))
		sc := NewContext(s, nil, proto.MainNetScheme)
		err := UserAddressFromString(sc)
		require.NoError(t, err)
		require.Equal(t, sc.Pop(), ast.NewUnit())
	})
	t.Run("correct address provided", func(t *testing.T) {
		s := NewStack()
		s.Push(ast.NewString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"))
		sc := NewContext(s, nil, proto.MainNetScheme)
		err := UserAddressFromString(sc)
		require.NoError(t, err)
		require.Equal(t,
			sc.Pop(),
			ast.NewAddressFromProtoAddress(proto.MustAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")))
	})
}

func TestNativeSumLong(t *testing.T) {
	s := NewStack()
	s.PushL(5)
	s.PushL(6)
	c := ctx(s)
	err := NativeSumLong(c)
	require.NoError(t, err)
	require.Equal(t, ast.NewLong(11), s.Pop())
}
