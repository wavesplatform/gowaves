package ast

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type ContentType byte

const (
	ContentTypeExpression ContentType = iota + 1
	ContentTypeApplication
)

func NewContentType(b byte) (ContentType, error) {
	ct := ContentType(b)
	switch ct {
	case ContentTypeExpression, ContentTypeApplication:
		return ct, nil
	default:
		return 0, errors.Errorf("unsupported content type '%d'", b)
	}
}

type LibraryVersion byte

func (lv *LibraryVersion) UnmarshalJSON(data []byte) error {
	var version byte
	if err := json.Unmarshal(data, &version); err != nil {
		return errors.Wrap(err, "unmarshal LibraryVersion failed")
	}

	v, err := NewLibraryVersion(version)
	if err != nil {
		return errors.Wrap(err, "unmarshal LibraryVersion failed")
	}

	*lv = v
	return nil
}

const (
	LibV1 LibraryVersion = iota + 1
	LibV2
	LibV3
	LibV4
	LibV5
	LibV6
)

// CurrentMaxLibraryVersion reports the max lib version. Update it when a new version was added.
func CurrentMaxLibraryVersion() LibraryVersion {
	return LibV6
}

func NewLibraryVersion(b byte) (LibraryVersion, error) {
	lv := LibraryVersion(b)
	switch lv {
	case LibV1, LibV2, LibV3, LibV4, LibV5, LibV6:
		return lv, nil
	default:
		return 0, errors.Errorf("unsupported library version '%d'", b)
	}
}
