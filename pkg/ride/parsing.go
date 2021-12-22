package ride

import (
	"bytes"
	sh256 "crypto/sha256"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type contentType byte

const (
	contentTypeExpression contentType = iota + 1
	contentTypeApplication
)

func newContentType(b byte) (contentType, error) {
	ct := contentType(b)
	switch ct {
	case contentTypeExpression, contentTypeApplication:
		return ct, nil
	default:
		return 0, errors.Errorf("unsupported content type '%d'", b)
	}
}

type libraryVersion byte

const (
	libV1 libraryVersion = iota + 1
	libV2
	libV3
	libV4
	libV5
	libV6
)

func newLibraryVersion(b byte) (libraryVersion, error) {
	lv := libraryVersion(b)
	switch lv {
	case libV1, libV2, libV3, libV4, libV5, libV6:
		return lv, nil
	default:
		return 0, errors.Errorf("unsupported library version '%d'", b)
	}
}

const (
	tokenLong byte = iota
	tokenBytes
	tokenString
	tokenIf
	tokenBlockV1
	tokenRef
	tokenTrue
	tokenFalse
	tokenGetter
	tokenFunctionCall
	tokenBlockV2
)

const (
	functionTypeNative byte = iota
	functionTypeUser
)

const (
	declarationTypeLet byte = iota
	declarationTypeFunction
)

func Parse(script []byte) (*Tree, error) {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script")
	}
	if !ok {
		return nil, errors.New("invalid script checksum")
	}
	id := sh256.Sum256(script)
	r := bytes.NewReader(script)
	p, err := parseHeader(r, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script header")
	}
	return p.parse()
}

type parser interface {
	parse() (*Tree, error)
}

type scriptHeader struct {
	content contentType
	library libraryVersion
}

/*
	Serialization mode V1 (LIBRARY_VERSION <= 5):
	00 CONTENT_TYPE LIBRARY_VERSION <DAPP|EXPRESSION> - DApp, Expression
	LIBRARY_VERSION <EXPRESSION> - Expression

	Serialization mode V2 (since LIBRARY_VERSION >= 6):
	LIBRARY_VERSION CONTENT_TYPE <DAPP|EXPRESSION> - DApp, Expression
*/
func parseHeader(r *bytes.Reader, id [32]byte) (parser, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if b < byte(libV6) {
		switch b {
		case 0:
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			ct, err := newContentType(b)
			if err != nil {
				return nil, err
			}
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			lv, err := newLibraryVersion(b)
			if err != nil {
				return nil, err
			}
			return &parserV1{r: r, id: id, header: scriptHeader{content: ct, library: lv}}, nil
		default:
			lv := libraryVersion(b)
			return &parserV1{r: r, id: id, header: scriptHeader{content: contentTypeExpression, library: lv}}, nil
		}
	}
	lv, err := newLibraryVersion(b)
	if err != nil {
		return nil, err
	}
	b, err = r.ReadByte()
	if err != nil {
		return nil, err
	}
	ct, err := newContentType(b)
	if err != nil {
		return nil, err
	}
	return &parserV2{r: r, id: id, header: scriptHeader{content: ct, library: lv}}, nil
}

func verifyCheckSum(scr []byte) (bool, error) {
	size := len(scr) - 4
	if size <= 0 {
		return false, errors.Errorf("invalid source length %d", size)
	}
	body, cs := scr[:size], scr[size:]
	digest, err := crypto.SecureHash(body)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify check sum")
	}
	return bytes.Equal(digest[:4], cs), nil
}
