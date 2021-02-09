package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSerialization(t *testing.T) {
	t.Run("uint16", func(t *testing.T) {
		s := NewSerializer()
		s.Uint16(10)

		d := NewDeserializer(s.Source())
		rs, _ := d.Uint16()
		require.Equal(t, uint16(10), rs)
	})
	t.Run("string", func(t *testing.T) {
		s := NewSerializer()
		_ = s.RideString("some string")

		d := NewDeserializer(s.Source())
		rs, _ := d.RideString()
		require.Equal(t, "some string", rs)
	})
	t.Run("entrypoint", func(t *testing.T) {
		e := Entrypoint{
			name: "name",
			at:   1050,
			argn: 2,
		}
		s := NewSerializer()
		err := e.Serialize(s)
		require.NoError(t, err)

		d := NewDeserializer(s.Source())
		e2, err := deserializeEntrypoint(d)
		require.NoError(t, err)
		require.Equal(t, e, e2)
	})
	t.Run("point", func(t *testing.T) {
		e := point{
			position: 1050,
			value:    rideString("ss"),
			fn:       1,
		}
		s := NewSerializer()
		err := e.Serialize(s)
		require.NoError(t, err)

		d := NewDeserializer(s.Source())
		e2, err := deserializePoint(d)
		require.NoError(t, err)
		require.Equal(t, e, e2)
	})
	t.Run("address", func(t *testing.T) {
		addr := rideAddress(proto.MustAddressFromString("3PKRnKhEmJbhGLDszhLmK8tptzBNjbw6wi4"))
		s := NewSerializer()
		require.NoError(t, addr.Serialize(s))

		d := NewDeserializer(s.Source())
		v, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, addr, v)
	})
	t.Run("NamedType", func(t *testing.T) {
		namedType := newDown(nil)
		s := NewSerializer()
		require.NoError(t, namedType.Serialize(s))

		d := NewDeserializer(s.Source())
		v, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, namedType, v)
	})
	t.Run("Unit", func(t *testing.T) {
		v1 := newUnit(nil)
		s := NewSerializer()
		require.NoError(t, v1.Serialize(s))

		d := NewDeserializer(s.Source())
		v2, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	})
	t.Run("List", func(t *testing.T) {
		v1 := rideList{rideInt(1), rideInt(2)}
		s := NewSerializer()
		require.NoError(t, v1.Serialize(s))

		d := NewDeserializer(s.Source())
		v2, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	})
	t.Run("Object", func(t *testing.T) {
		v1 := rideObject{"key": rideString("value")}
		s := NewSerializer()
		require.NoError(t, v1.Serialize(s))

		d := NewDeserializer(s.Source())
		v2, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	})
	t.Run("bytes", func(t *testing.T) {
		v1 := rideBytes(nil)
		s := NewSerializer()
		require.NoError(t, v1.Serialize(s))

		d := NewDeserializer(s.Source())
		v2, err := d.RideValue()
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	})
}

func TestSerdeTransaction(t *testing.T) {
	obj := testExchangeWithProofsToObject()

	s := NewSerializer()
	require.NoError(t, obj.Serialize(s))

	d := NewDeserializer(s.Source())
	v, err := d.RideValue()
	require.NoError(t, err)

	require.Equal(t, obj, v)
}
