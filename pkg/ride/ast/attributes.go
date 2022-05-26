package ast

import "github.com/pkg/errors"

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

const (
	LibV1 LibraryVersion = iota + 1
	LibV2
	LibV3
	LibV4
	LibV5
	LibV6
)

func NewLibraryVersion(b byte) (LibraryVersion, error) {
	lv := LibraryVersion(b)
	switch lv {
	case LibV1, LibV2, LibV3, LibV4, LibV5, LibV6:
		return lv, nil
	default:
		return 0, errors.Errorf("unsupported library version '%d'", b)
	}
}
