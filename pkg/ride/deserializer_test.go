package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		s.RideString("some string")

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
}
