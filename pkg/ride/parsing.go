package ride

import (
	"bytes"
	sh256 "crypto/sha256"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	scriptApplicationVersion = 1
)

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
	id := sh256.Sum256(script)
	ok, err := verifyCheckSum(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script")
	}
	if !ok {
		return nil, errors.New("invalid script checksum")
	}
	switch script[0] {
	case 0xff:
		p := parserV2{r: bytes.NewReader(script), id: id}
		return p.parse()
	default:
		p := parserV1{r: bytes.NewReader(script), id: id}
		return p.parse()
	}
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
