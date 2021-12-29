package data

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	delimiter    = "."
	infiniteText = "\"infinite\""
)

type Decimal struct {
	value uint64
	scale uint
}

func NewDecimal(value uint64, scale uint) *Decimal {
	return &Decimal{value, scale}
}

func NewDecimalFromString(s string) (*Decimal, error) {
	p := strings.Split(s, delimiter)
	switch len(p) {
	case 1:
		i, err := parsePart(p[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert '%s' to Decimal", s)
		}
		return &Decimal{i, 0}, nil
	case 2:
		i1, err := parsePart(p[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert '%s' to Decimal", s)
		}
		i2, err := parsePart(p[1])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert '%s' to Decimal", s)
		}
		if i2 > 0 {
			s := len(p[1])
			v := i1*uint64(math.Pow10(s)) + i2
			return &Decimal{v, uint(s)}, nil
		}
		return &Decimal{i1, 0}, nil
	default:
		return nil, errors.Errorf("failed to convert '%s' to Decimal", s)
	}
}

func (d *Decimal) Scale() int {
	return int(d.scale)
}

func (d *Decimal) Value() int {
	return int(d.value)
}

func (d *Decimal) String() string {
	var sb strings.Builder
	str := strconv.Itoa(d.Value())
	s := d.Scale()
	l := len(str)

	var a, b string
	if l >= s {
		a = str[:l-s]
		b = str[l-s:]
	} else {
		b = strings.Repeat("0", s-l) + str
	}
	if a == "" {
		a = "0"
	}
	sb.WriteString(a)
	if len(b) > 0 {
		sb.WriteString(delimiter)
		if len(b) > 8 {
			b = b[:8]
		}
		sb.WriteString(b)
	}
	return sb.String()
}

func (d *Decimal) Rescale(scale uint) *Decimal {
	switch {
	case d.scale < scale:
		diff := int(scale - d.scale)
		v := new(big.Int).SetUint64(d.value)
		y := new(big.Int).SetUint64(uint64(math.Pow10(diff)))
		v = v.Mul(v, y)
		return &Decimal{v.Uint64(), scale}
	case d.scale > scale:
		diff := int(d.scale - scale)
		v := new(big.Int).SetUint64(d.value)
		y := new(big.Int).SetUint64(uint64(math.Pow10(diff)))
		v = v.Quo(v, y)
		return &Decimal{v.Uint64(), scale}
	default:
		return &Decimal{d.value, d.scale}
	}
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(d.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (d *Decimal) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Decimal from JSON")
	}
	v, err := NewDecimalFromString(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Decimal from JSON")
	}
	d.value = v.value
	d.scale = v.scale
	return nil
}

func parsePart(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func (d *Decimal) ToInfiniteDecimal(infinite bool) InfiniteDecimal {
	return InfiniteDecimal{Decimal: *d, Infinite: infinite}
}

type InfiniteDecimal struct {
	Decimal
	Infinite bool
}

func (d InfiniteDecimal) MarshalJSON() ([]byte, error) {
	if d.Infinite {
		return []byte(infiniteText), nil
	}
	return d.Decimal.MarshalJSON()
}

func (d *InfiniteDecimal) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == infiniteText {
		d.Infinite = true
		return nil
	}
	return d.Decimal.UnmarshalJSON(value)
}

func (d *InfiniteDecimal) String() string {
	if d.Infinite {
		s, err := strconv.Unquote(infiniteText)
		if err != nil {
			return infiniteText
		}
		return s
	}
	return d.Decimal.String()
}
