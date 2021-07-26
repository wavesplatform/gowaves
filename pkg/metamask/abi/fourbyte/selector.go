package fourbyte

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

// DecodedCallData is an internal type to represent a method call parsed according
// to an ABI method signature.
type DecodedCallData struct {
	Signature string
	Name      string
	Inputs    []DecodedArg
	Payments  []Payment
}

// String implements stringer interface for decodedCallData
func (cd DecodedCallData) String() string {
	args := make([]string, len(cd.Inputs))
	for i, arg := range cd.Inputs {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", cd.Name, strings.Join(args, ","))
}

// Database is a 4byte database with the possibility of maintaining an immutable
// set (embedded) into the process and a mutable set (loaded and written to file).
type Database struct {
	embedded map[string]string
	custom   map[string]string
}

// New loads the standard signature database embedded in the package.
func NewDatabase() (*Database, error) {
	db := &Database{make(map[string]string), make(map[string]string)}
	db.embedded = __4byteJson

	return db, nil
}

// This method does not validate the match, it's assumed the caller will do.
func (db *Database) Selector(id []byte) (string, error) {
	if len(id) < 4 {
		return "", fmt.Errorf("expected 4-byte id, got %d", len(id))
	}
	sig := hex.EncodeToString(id[:4])
	if selector, exists := db.embedded[sig]; exists {
		return selector, nil
	}
	if selector, exists := db.custom[sig]; exists {
		return selector, nil
	}
	return "", fmt.Errorf("signature %v not found", sig)
}

func (db *Database) MethodBySelector(id Selector) (Method, error) {
	if method, ok := erc20Methods[id]; ok {
		return method, nil
	}
	// TODO(nickeskov): support ride scripts metadata
	return Method{}, fmt.Errorf("signature %v not found", id.String())
}

func (db *Database) ParseCallDataNew(data []byte) (*DecodedCallData, error) {
	// If the data is empty, we have a plain value transfer, nothing more to do
	if len(data) == 0 {
		return nil, errors.New("transaction doesn't contain data")
	}
	// Validate the call data that it has the 4byte prefix and the rest divisible by 32 bytes
	if len(data) < 4 {
		return nil, errors.New("transaction data is not valid ABI: missing the 4 byte call prefix")
	}
	if n := len(data) - 4; n%32 != 0 {
		return nil, errors.Errorf("transaction data is not valid ABI (length should be a multiple of 32 (was %d))", n)
	}
	var selector Selector
	copy(selector[:], data[:len(selector)])
	method, err := db.MethodBySelector(selector)
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but the ABI signature could not be found: %v", err)
	}

	info, err := parseArgData(&method, data[len(selector):])
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but provided ABI signature could not be verified: %v", err)
	}
	return info, nil
}

func (db *Database) ParseCallDataRide(data []byte) (*DecodedCallData, error) {
	// If the data is empty, we have a plain value transfer, nothing more to do
	if len(data) == 0 {
		return nil, errors.New("transaction doesn't contain data")
	}
	// Validate the call data that it has the 4byte prefix and the rest divisible by 32 bytes
	if len(data) < 4 {
		return nil, errors.New("transaction data is not valid ABI: missing the 4 byte call prefix")
	}
	if n := len(data) - 4; n%32 != 0 {
		return nil, errors.Errorf("transaction data is not valid ABI (length should be a multiple of 32 (was %d))", n)
	}
	var selector Selector
	copy(selector[:], data[:len(selector)])
	method, err := db.MethodBySelector(selector)
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but the ABI signature could not be found: %v", err)
	}

	info, err := parseArgDataToRideTypes(&method, data[len(selector):])
	if err != nil {
		return nil, errors.Errorf("Transaction contains data, but provided ABI signature could not be verified: %v", err)
	}
	return info, nil
}

type DecodedArg struct {
	Soltype Argument
	Value   interface{}
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

func parseArgData(method *Method, argData []byte) (*DecodedCallData, error) {
	values, paymentsABI, err := method.Inputs.UnpackValues(argData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack Inputs arguments ABI data")
	}
	payments, err := unpackPayments(paymentsABI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack payments")
	}

	decoded := DecodedCallData{Signature: method.Sig.String(), Name: method.RawName, Payments: payments}
	for i := 0; i < len(method.Inputs); i++ {
		decoded.Inputs = append(decoded.Inputs, DecodedArg{
			Soltype: method.Inputs[i],
			Value:   values[i],
		})
	}
	return &decoded, nil
}

func parseArgDataToRideTypes(method *Method, argData []byte) (*DecodedCallData, error) {
	values, paymentsABI, err := method.Inputs.UnpackRideValues(argData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack Inputs arguments ABI data")
	}
	payments, err := unpackPayments(paymentsABI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack payments")
	}

	decoded := DecodedCallData{Signature: method.Sig.String(), Name: method.RawName, Payments: payments}
	for i := 0; i < len(method.Inputs); i++ {
		decoded.Inputs = append(decoded.Inputs, DecodedArg{
			Soltype: method.Inputs[i],
			Value:   values[i],
		})
	}

	return &decoded, nil
}
