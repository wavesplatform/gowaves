package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimalString1(t *testing.T) {
	a := NewDecimal(12345, 2)
	assert.Equal(t, "123.45", a.String())
}

func TestDecimalString2(t *testing.T) {
	a := NewDecimal(12345, 0)
	assert.Equal(t, "12345", a.String())
}

func TestDecimalString3(t *testing.T) {
	a := NewDecimal(12345, 8)
	assert.Equal(t, "0.00012345", a.String())
}

func TestDecimalString4(t *testing.T) {
	a := NewDecimal(1234567890, 8)
	assert.Equal(t, "12.34567890", a.String())
}

func TestDecimalString5(t *testing.T) {
	a := NewDecimal(100000000, 8)
	assert.Equal(t, "1.00000000", a.String())
}

func TestNewDecimalFromString1(t *testing.T) {
	if a, err := NewDecimalFromString("12.345"); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 3, a.Scale())
		assert.Equal(t, "12.345", a.String())
	}
}

func TestNewDecimalFromString2(t *testing.T) {
	if a, err := NewDecimalFromString("12345"); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "12345", a.String())
	}
}

func TestNewDecimalFromString3(t *testing.T) {
	if a, err := NewDecimalFromString("12345.0"); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "12345", a.String())
	}
}

func TestNewDecimalFromString4(t *testing.T) {
	if a, err := NewDecimalFromString(".12345"); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 5, a.Scale())
		assert.Equal(t, "0.12345", a.String())
	}
}

func TestNewDecimalFromString5(t *testing.T) {
	if a, err := NewDecimalFromString("0.12345"); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 5, a.Scale())
		assert.Equal(t, "0.12345", a.String())
	}
}

func TestNewDecimalFromString6(t *testing.T) {
	if a, err := NewDecimalFromString("12345."); assert.NoError(t, err) {
		assert.Equal(t, 12345, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "12345", a.String())
	}
}

func TestNewDecimalFromString7(t *testing.T) {
	if a, err := NewDecimalFromString("."); assert.NoError(t, err) {
		assert.Equal(t, 0, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "0", a.String())
	}
}

func TestNewDecimalFromString8(t *testing.T) {
	if a, err := NewDecimalFromString("0"); assert.NoError(t, err) {
		assert.Equal(t, 0, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "0", a.String())
	}
}

func TestNewDecimalFromString9(t *testing.T) {
	if a, err := NewDecimalFromString("0.0"); assert.NoError(t, err) {
		assert.Equal(t, 0, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "0", a.String())
	}
}

func TestNewDecimalFromString10(t *testing.T) {
	if a, err := NewDecimalFromString(".0"); assert.NoError(t, err) {
		assert.Equal(t, 0, a.Value())
		assert.Equal(t, 0, a.Scale())
		assert.Equal(t, "0", a.String())
	}
}

func TestNewDecimalFromString11(t *testing.T) {
	if a, err := NewDecimalFromString("1234.500"); assert.NoError(t, err) {
		assert.Equal(t, 1234500, a.Value())
		assert.Equal(t, 3, a.Scale())
		assert.Equal(t, "1234.500", a.String())
	}
}

func TestNewDecimalFromStringFail1(t *testing.T) {
	_, err := NewDecimalFromString("qwert")
	assert.Error(t, err, "failed to convert 'qwert' to Decimal")
}

func TestNewDecimalFromStringFail2(t *testing.T) {
	_, err := NewDecimalFromString("123.qwert")
	assert.Error(t, err, "failed to convert '123.qwert' to Decimal")
}

func TestNewDecimalFromStringFail3(t *testing.T) {
	_, err := NewDecimalFromString("qwert.456")
	assert.Error(t, err, "failed to convert 'qwert.456' to Decimal")
}

func TestNewDecimalFromStringFail4(t *testing.T) {
	_, err := NewDecimalFromString("123.456.789")
	assert.Error(t, err, "failed to convert '123.456.789' to Decimal")
}

func TestDecimalRescale1(t *testing.T) {
	a, _ := NewDecimalFromString("123.45")
	b := a.Rescale(8)
	assert.Equal(t, "123.45000000", b.String())
}

func TestDecimalRescale2(t *testing.T) {
	a, _ := NewDecimalFromString("0.012345")
	b := a.Rescale(8)
	assert.Equal(t, "0.01234500", b.String())
}

func TestDecimalRescale3(t *testing.T) {
	a, _ := NewDecimalFromString("0.12345678")
	b := a.Rescale(4)
	assert.Equal(t, "0.1234", b.String())
}

func TestDecimalRescale4(t *testing.T) {
	a, _ := NewDecimalFromString("12345.67890")
	b := a.Rescale(5)
	assert.Equal(t, "12345.67890", b.String())
}

func TestDecimalMarshalJSON(t *testing.T) {
	a, err := NewDecimalFromString("0.123456789")
	require.NoError(t, err)
	s, err := a.MarshalJSON()
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte("\"0.12345678\""), s) // Length of the decimal part is limited by 8 symbols
}

func TestDecimalUnmarshalJSON(t *testing.T) {
	js := "\"12345.67890\""
	var a Decimal
	err := a.UnmarshalJSON([]byte(js))
	require.NoError(t, err)
	assert.Equal(t, "12345.67890", a.String())
}

func TestInfiniteDecimalString1(t *testing.T) {
	d, err := NewDecimalFromString("12345.6789")
	require.NoError(t, err)
	a := d.ToInfiniteDecimal(false)
	assert.Equal(t, "12345.6789", a.String())
}

func TestInfiniteDecimalString2(t *testing.T) {
	d, err := NewDecimalFromString("12345.6789")
	require.NoError(t, err)
	a := d.ToInfiniteDecimal(true)
	assert.Equal(t, "infinite", a.String())
}

func TestInfiniteDecimalMarshalJSON1(t *testing.T) {
	d, err := NewDecimalFromString("123.4567890")
	require.NoError(t, err)
	a := d.ToInfiniteDecimal(false)
	js, err := a.MarshalJSON()
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte("\"123.4567890\""), js)
}

func TestInfiniteDecimalMarshalJSON2(t *testing.T) {
	d, err := NewDecimalFromString("123.4567890")
	require.NoError(t, err)
	a := d.ToInfiniteDecimal(true)
	js, err := a.MarshalJSON()
	require.NoError(t, err)
	assert.ElementsMatch(t, []byte("\"infinite\""), js)
}

func TestInfiniteDecimalUnmarshalJSON1(t *testing.T) {
	js := "\"12345.67890\""
	var a InfiniteDecimal
	err := a.UnmarshalJSON([]byte(js))
	require.NoError(t, err)
	assert.Equal(t, "12345.67890", a.String())
}

func TestInfiniteDecimalUnmarshalJSON2(t *testing.T) {
	js := "\"infinite\""
	var a InfiniteDecimal
	err := a.UnmarshalJSON([]byte(js))
	require.NoError(t, err)
	assert.Equal(t, "infinite", a.String())
}
