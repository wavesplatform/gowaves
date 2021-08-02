package fourbyte

import (
	"encoding"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metamask"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"strings"
)

const (
	selectorLen = 4

	addressSize = 20
	uint256Size = 256
)

const (
	erc20TransferSignature     Signature = "transfer(address,uint256)"
	erc20TransferFromSignature Signature = "transferFrom(address,address,uint256)"
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

type Selector [selectorLen]byte

func NewSelector(sig Signature) Selector {
	var selector Selector
	copy(selector[:], metamask.Keccak256([]byte(sig)))
	return selector
}

func (s Selector) String() string {
	return s.Hex()
}

func (s Selector) Hex() string {
	return hex.EncodeToString(s[:])
}

func (s *Selector) FromHex(hexSelector string) error {
	bts, err := hex.DecodeString(hexSelector)
	if err != nil {
		return errors.Wrap(err, "failed to decode hex string for selector")
	}
	if len(bts) != len(s) {
		return errors.Errorf("invalid hex selector bytes, expected %d, received %d", len(s), len(bts))
	}
	copy(s[:], bts)
	return nil
}

func rideMetaTypeToTextMarshaler(metaT meta.Type) (encoding.TextMarshaler, error) {
	switch t := metaT.(type) {
	case meta.SimpleType:
		switch t {
		case meta.Int:
			// nickeskov: it's RideInt type
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
		return sliceTextBuilder{t: inner}, nil
	case meta.UnionType:
		marshalers := make([]encoding.TextMarshaler, 0, len(t))
		for _, unionUnitT := range t {
			unionUnitMarshaler, err := rideMetaTypeToTextMarshaler(unionUnitT)
			if err != nil {
				return nil, errors.Wrapf(err,
					"failed to create text marshaler for union type, inner type %T", unionUnitT,
				)
			}
			marshalers = append(marshalers, unionUnitMarshaler)
		}
		return unionTextBuilder(marshalers), nil
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
	t encoding.TextMarshaler
}

func (stb sliceTextBuilder) MarshalText() (text []byte, err error) {
	marshaled, err := stb.t.MarshalText()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal %T", stb.t)
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

type unionTextBuilder []encoding.TextMarshaler

func (utb unionTextBuilder) MarshalText() (text []byte, err error) {
	if len(utb) == 0 {
		return nil, errors.Errorf("can't marshal union with no elements")
	}
	unionElements := make([]encoding.TextMarshaler, len(utb)+1)
	// nickeskov: create index element to represent ride tuple in ethereum abi
	unionElements[0] = intTextBuilder{
		size:     8,
		unsigned: true,
	}
	copy(unionElements[1:], utb)
	text, err = tupleTextBuilder(unionElements).MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal union")
	}
	return text, nil
}

type paymentTextBuilder struct{}

func (ptb paymentTextBuilder) MarshalText() (text []byte, err error) {
	tupleBuilder := tupleTextBuilder{
		bytesTextBuilder{},
		// nickeskov: asset amount field in payment
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
			return nil, errors.Errorf("failed to create function argument text marshaler, type %T", arg)
		}
		marshaled, err := marshaler.MarshalText()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal function argument, type %T", arg)
		}
		elements = append(elements, string(marshaled))
	}
	if ftb.addPayments {
		payments, err := sliceTextBuilder{t: paymentTextBuilder{}}.MarshalText()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal payments")
		}
		elements = append(elements, string(payments))
	}
	return []byte(fmt.Sprintf("%s(%s)", ftb.functionMeta.Name, strings.Join(elements, ","))), nil
}

var Erc20Methods = map[Selector]Method{
	erc20TransferSignature.Selector(): {
		RawName: "transfer",
		Inputs: Arguments{
			Argument{
				Name: "_to",
				Type: Type{
					Size:       addressSize,
					T:          AddressTy,
					stringKind: "address",
				},
			},
			Argument{
				Name: "_value",
				Type: Type{
					Size:       uint256Size,
					T:          UintTy,
					stringKind: "uint256",
				},
			},
		},
		Payments: nil,
		Sig:      erc20TransferSignature,
	},
	erc20TransferFromSignature.Selector(): {
		RawName: "transferFrom",
		Inputs: Arguments{
			Argument{
				Name: "_from",
				Type: Type{
					Size:       addressSize,
					T:          AddressTy,
					stringKind: "address",
				},
			},
			Argument{
				Name: "_to",
				Type: Type{
					Size:       addressSize,
					T:          AddressTy,
					stringKind: "address",
				},
			},
			Argument{
				Name: "_value",
				Type: Type{
					Size:       uint256Size,
					T:          UintTy,
					stringKind: "uint256",
				},
			},
		},
		Payments: nil,
		Sig:      erc20TransferFromSignature,
	},
}
