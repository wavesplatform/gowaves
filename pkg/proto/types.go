package proto

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"strings"
)

// B58Bytes represents bytes as Base58 string in JSON
type B58Bytes []byte

// String represents underlying bytes as Base58 string
func (b B58Bytes) String() string {
	return base58.Encode(b)
}

// MarshalJSON writes B58Bytes Value as JSON string
func (b B58Bytes) MarshalJSON() ([]byte, error) {
	s := base58.Encode(b)
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads B58Bytes from JSON string
func (b *B58Bytes) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s = strings.TrimSuffix(strings.TrimPrefix(s, "\""), "\"")
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode B58Bytes")
	}
	*b = B58Bytes(v)
	return nil
}

// Asset represents an optional asset identification
type Asset struct {
	Present bool
	ID      B58Bytes
}

func NewAssetFromString(s string) (*Asset, error) {
	if strings.ToUpper(s) == WavesAssetName {
		return &Asset{Present: false}, nil
	} else {
		a, err := base58.Decode(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create Asset from string")
		}
		return &Asset{Present: true, ID: a}, nil
	}
}

// String method converts Asset to its text representation
func (a Asset) String() string {
	if a.Present {
		return a.ID.String()
	} else {
		return WavesAssetName
	}
}

// MarshalJSON writes Asset as a JSON string Value
func (a Asset) MarshalJSON() ([]byte, error) {
	if a.Present {
		return a.ID.MarshalJSON()
	} else {
		return []byte("null"), nil
	}
}

// UnmarshalJSON reads Asset from a JSON string Value
func (a *Asset) UnmarshalJSON(value []byte) error {
	s := strings.ToUpper(string(value))
	if s == "NULL" || s == WavesAssetName || s == "" {
		*a = Asset{Present: false}
	} else {
		var b B58Bytes
		err := b.UnmarshalJSON(value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Asset.ID")
		}
		*a = Asset{Present: true, ID: b}
	}
	return nil
}

type Attachment string

// String returns Attachment's string representation
func (a Attachment) String() string {
	return string(a)
}

// MarshalJSON writes Attachment as a JSON string Value
func (a Attachment) MarshalJSON() ([]byte, error) {
	b := []byte(a)
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(base58.Encode(b))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads Attachment from a JSON string Value
func (a *Attachment) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s = strings.TrimSuffix(strings.TrimPrefix(s, "\""), "\"")
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode Attachment from JSON Value")
	}
	*a = Attachment(string(v))
	return nil
}
