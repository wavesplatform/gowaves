package ethabi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

// DecodedCallData is an internal type to represent a method call parsed according
// to an ABI method signature.
type DecodedCallData struct {
	Signature Signature
	Name      string
	Inputs    []DecodedArg
	Payments  []Payment
}

// String implements stringer interface for DecodedCallData
func (cd DecodedCallData) String() string {
	args := make([]string, len(cd.Inputs))
	for i, arg := range cd.Inputs {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", cd.Name, strings.Join(args, ","))
}

// IsERC20TransferSelector checks that selector is an ERC20Transfer function selector
func IsERC20TransferSelector(id Selector) bool {
	return id == erc20TransferSelector
}

type MethodsMap struct {
	methods       map[Selector]Method
	parsePayments bool
}

func NewErc20MethodsMap() MethodsMap {
	return MethodsMap{
		methods:       erc20Methods,
		parsePayments: false,
	}
}

func NewMethodsMapFromRideDAppMeta(dApp meta.DApp) (MethodsMap, error) {
	return newMethodsMapFromRideDAppMeta(dApp, true)
}

func newMethodsMapFromRideDAppMeta(dApp meta.DApp, parsePayments bool) (MethodsMap, error) {
	methods := make(map[Selector]Method, len(dApp.Functions))
	for _, fn := range dApp.Functions {
		method, err := NewMethodFromRideFunctionMeta(fn, parsePayments)
		if err != nil {
			if errors.Is(err, UnsupportedType) {
				continue // ignore ride callable with unsupported argument type
			}
			return MethodsMap{}, errors.Wrapf(err,
				"failed to build ABI db from DApp metadata, verison %d", dApp.Version,
			)
		}
		methods[method.Sig.Selector()] = method
	}
	db := MethodsMap{
		methods:       methods,
		parsePayments: parsePayments,
	}
	return db, nil
}

func (mm MethodsMap) MethodBySelector(id Selector) (Method, error) {
	if method, ok := mm.methods[id]; ok {
		return method, nil
	}
	return Method{}, fmt.Errorf("signature %q not found", id.String())
}

func (mm MethodsMap) ParseCallDataRide(data []byte) (*DecodedCallData, error) {
	// If the data is empty, we have a plain value transfer, nothing more to do
	if len(data) == 0 {
		return nil, errors.New("transaction doesn't contain data")
	}
	// Validate the call data that it has the 4byte prefix and the rest divisible by 32 bytes
	if len(data) < SelectorSize {
		return nil, errors.New("transaction data is not valid ABI: missing the 4 byte call prefix")
	}
	if n := len(data) - SelectorSize; n%32 != 0 {
		return nil, errors.Errorf("transaction data is not valid ABI (length should be a multiple of 32 (was %d))", n)
	}
	var selector Selector
	copy(selector[:], data[:SelectorSize])
	method, err := mm.MethodBySelector(selector)
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but the ABI signature could not be found: %v", err)
	}

	info, err := parseArgDataToRideTypes(&method, data[SelectorSize:], mm.parsePayments)
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but provided ABI signature could not be verified: %v", err)
	}
	return info, nil
}

func (mm MethodsMap) MarshalJSON() ([]byte, error) {
	abiResult := make([]abi, 0, len(mm.methods))

	for _, method := range mm.methods {
		methodABI, err := makeJSONABIForMethod(method)
		if err != nil {
			return nil, err
		}
		abiResult = append(abiResult, methodABI)
	}

	return json.Marshal(abiResult)
}

type DecodedArg struct {
	Soltype Argument
	Value   DataType
}

func (da *DecodedArg) String() string {
	var value string
	switch val := da.Value.(type) {
	case fmt.Stringer:
		value = val.String()
	default:
		value = fmt.Sprintf("%v", val)
	}
	return fmt.Sprintf("%v: %v", da.Soltype.Type.String(), value)
}

func (da *DecodedArg) DecodedValue() interface{} {
	return da.Value
}

func (da *DecodedArg) InternalType() byte {
	return byte(da.Soltype.Type.T)
}

func parseArgDataToRideTypes(method *Method, argData []byte, parsePayments bool) (*DecodedCallData, error) {
	values, paymentsOffset, err := method.Inputs.UnpackRideValues(argData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack Inputs arguments ABI data")
	}

	var payments []Payment
	if parsePayments {
		payments, err = unpackPayments(paymentsOffset, argData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack payments")
		}
	}

	decodedInputs := make([]DecodedArg, len(method.Inputs))
	for i := range method.Inputs {
		decodedInputs[i] = DecodedArg{Soltype: method.Inputs[i], Value: values[i]}
	}
	decoded := DecodedCallData{
		Signature: method.Sig,
		Name:      method.RawName,
		Inputs:    decodedInputs,
		Payments:  payments,
	}
	return &decoded, nil
}
