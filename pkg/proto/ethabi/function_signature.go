package ethabi

import (
	"encoding"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

type Signature string

func NewSignatureFromRideFunctionMeta(fn meta.Function, addPayments bool) (Signature, error) {
	builder := functionTextBuilder{
		addPayments:  addPayments,
		functionMeta: fn,
	}
	signature, err := builder.MarshalText()
	if err != nil {
		return "", errors.Errorf("failed to build signature for function %s with %d arguments",
			fn.Name,
			len(fn.Arguments),
		)
	}
	return Signature(signature), nil
}

func (s Signature) String() string {
	return string(s)
}

func (s Signature) Selector() Selector {
	return NewSelector(s)
}

const SelectorSize = 4

type Selector [SelectorSize]byte

func NewSelector(sig Signature) Selector {
	var selector Selector
	hash := crypto.MustKeccak256([]byte(sig))
	copy(selector[:], hash[:])
	return selector
}

func NewSelectorFromBytes(a []byte) (Selector, error) {
	if len(a) != SelectorSize {
		return Selector{}, errors.Errorf("failed to create new selector, invalid selector size: want %d, got %d", SelectorSize, len(a))
	}
	var selector Selector
	copy(selector[:], a[:])
	return selector, nil
}

func (s Selector) String() string {
	return s.Hex()
}

func (s Selector) Hex() string {
	return fmt.Sprintf("0x%s", hex.EncodeToString(s[:]))
}

func (s *Selector) FromHex(hexSelector string) error {
	bts, err := hex.DecodeString(strings.TrimPrefix(hexSelector, "0x"))
	if err != nil {
		return errors.Wrap(err, "failed to decode hex string for selector")
	}
	if len(bts) != SelectorSize {
		return errors.Errorf("invalid hex selector bytes, expected %d, received %d", SelectorSize, len(bts))
	}
	copy(s[:], bts)
	return nil
}

func rideMetaTypeToTextMarshaler(metaT meta.Type) (encoding.TextMarshaler, error) {
	switch t := metaT.(type) {
	case meta.SimpleType:
		switch t {
		case meta.Int:
			//  it's RideInt type
			marshaler := intTextBuilder{
				size:     64,
				unsigned: false,
			}
			return marshaler, nil
		case meta.Bytes:
			return bytesTextBuilder{}, nil
		case meta.Boolean:
			return booleanTextBuilder{}, nil
		case meta.String:
			return stringTextBuilder{}, nil
		default:
			return nil, errors.Errorf("invalid ride simple type (%d)", t)
		}
	case meta.ListType:
		inner, err := rideMetaTypeToTextMarshaler(t.Inner)
		if err != nil {
			return nil, errors.Wrapf(err,
				"failed to create text marshaler for list type, inner type %T", t.Inner,
			)
		}
		return sliceTextBuilder{inner: inner}, nil
	case meta.UnionType:
		return nil, errors.Wrap(UnsupportedType, "UnionType")
	default:
		return nil, errors.Errorf("unsupported ride metadata type, type %T", t)
	}
}

type intTextBuilder struct {
	size     int
	unsigned bool
}

func (itb intTextBuilder) MarshalText() (text []byte, err error) {
	if itb.size%8 != 0 || itb.size <= 0 || itb.size > 256 {
		return nil, errors.Errorf("invalid int type size (%d)", itb.size)
	}
	unsignedPrefix := ""
	if itb.unsigned {
		unsignedPrefix = "u"
	}
	return []byte(fmt.Sprintf("%sint%d", unsignedPrefix, itb.size)), nil
}

type fixedBytesTextBuilder struct {
	size int
}

func (fbtb fixedBytesTextBuilder) MarshalText() (text []byte, err error) {
	if fbtb.size < 1 || fbtb.size > 32 {
		return nil, errors.Errorf("invalid fixed bytes type size (%d)", fbtb.size)
	}
	return []byte(fmt.Sprintf("bytes%d", fbtb.size)), nil
}

type bytesTextBuilder struct{}

func (btb bytesTextBuilder) MarshalText() (text []byte, err error) {
	return []byte("bytes"), nil
}

type booleanTextBuilder struct{}

func (btb booleanTextBuilder) MarshalText() (text []byte, err error) {
	return []byte("bool"), nil
}

type stringTextBuilder struct{}

func (stb stringTextBuilder) MarshalText() (text []byte, err error) {
	return []byte("string"), nil
}

type sliceTextBuilder struct {
	inner encoding.TextMarshaler
}

func (stb sliceTextBuilder) MarshalText() (text []byte, err error) {
	marshaled, err := stb.inner.MarshalText()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal %T", stb.inner)
	}
	return []byte(fmt.Sprintf("%s[]", marshaled)), nil
}

type tupleTextBuilder []encoding.TextMarshaler

func (ttb tupleTextBuilder) MarshalText() (text []byte, err error) {
	elements := make([]string, 0, len(ttb))
	for _, elem := range ttb {
		marshaled, err := elem.MarshalText()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal tuple element, type %T", elem)
		}
		elements = append(elements, string(marshaled))
	}
	return []byte(fmt.Sprintf("(%s)", strings.Join(elements, ","))), nil
}

type paymentTextBuilder struct{}

func (ptb paymentTextBuilder) MarshalText() (text []byte, err error) {
	tupleBuilder := tupleTextBuilder{
		// full asset ID
		fixedBytesTextBuilder{
			size: 32,
		},
		// asset amount field in payment
		intTextBuilder{
			size:     64,
			unsigned: false,
		},
	}
	text, err = tupleBuilder.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payment")
	}
	return text, nil
}

type functionTextBuilder struct {
	addPayments  bool
	functionMeta meta.Function
}

func (ftb functionTextBuilder) MarshalText() (text []byte, err error) {
	sliceLen := len(ftb.functionMeta.Arguments)
	if ftb.addPayments {
		sliceLen += 1
	}
	elements := make([]string, 0, sliceLen)
	for _, arg := range ftb.functionMeta.Arguments {
		marshaler, err := rideMetaTypeToTextMarshaler(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create function argument text marshaler, type %T", arg)
		}
		marshaled, err := marshaler.MarshalText()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal function argument, type %T", arg)
		}
		elements = append(elements, string(marshaled))
	}
	if ftb.addPayments {
		payments, err := sliceTextBuilder{inner: paymentTextBuilder{}}.MarshalText()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal payments")
		}
		elements = append(elements, string(payments))
	}
	return []byte(fmt.Sprintf("%s(%s)", ftb.functionMeta.Name, strings.Join(elements, ","))), nil
}
