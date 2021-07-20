package fourbyte

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"strings"
)

var __4byteJson = map[string]string{
	"a9059cbb": "transfer(address,uint256)",
	"23b872dd": "transferFrom(address,address,uint256)",
}

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

func NewSignature(funcName string, inputArgs Arguments) Signature {
	typeStrings := make([]string, len(inputArgs))
	for i := range inputArgs {
		typeStrings[i] = inputArgs[i].Type.String()
	}
	sig := fmt.Sprintf("%s(%s)", funcName, strings.Join(typeStrings, ","))
	return Signature(sig)
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
	copy(selector[:], crypto.Keccak256([]byte(sig)))
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

var erc20Methods = map[Selector]Method{
	erc20TransferSignature.Selector(): {
		RawName: "transfer",
		Type:    Callable,
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
		Sig: erc20TransferSignature,
	},
	erc20TransferFromSignature.Selector(): {
		RawName: "transferFrom",
		Type:    Callable,
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
		Sig: erc20TransferFromSignature,
	},
}
