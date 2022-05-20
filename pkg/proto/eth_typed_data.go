package proto

import (
	"bytes"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type ethereumTypedDataType struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (t *ethereumTypedDataType) isArray() bool {
	return strings.HasSuffix(t.Type, "[]")
}

// typeName returns the canonical name of the type. If the type is 'Person[]', then
// this method returns 'Person'
func (t *ethereumTypedDataType) typeName() string {
	if t.isArray() {
		return strings.TrimSuffix(t.Type, "[]")
	}
	return t.Type
}

func (t *ethereumTypedDataType) isReferenceType() bool {
	if len(t.Type) == 0 {
		return false
	}
	// Reference types must have a leading uppercase character
	return unicode.IsUpper([]rune(t.Type)[0])
}

type ethereumTypedDataTypes map[string][]ethereumTypedDataType

type ethereumTypedDataDomain struct {
	Name    string           `json:"name,omitempty"`
	Version string           `json:"version,omitempty"`
	ChainId *hexOrDecimal256 `json:"chainId,omitempty"`
}

type ethereumTypedDataMessage map[string]interface{}

type ethereumTypedData struct {
	Types       ethereumTypedDataTypes   `json:"types"`
	PrimaryType string                   `json:"primaryType"`
	Domain      ethereumTypedDataDomain  `json:"domain"`
	Message     ethereumTypedDataMessage `json:"message"`
}

func (typedData *ethereumTypedData) RawData() ([]byte, error) {
	domainSeparator, err := typedData.HashStructMap("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStructMap(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	rawData := fmt.Sprintf("\x19\x01%s%s", string(domainSeparator[:]), string(typedDataHash[:]))
	return []byte(rawData), nil
}

func (typedData *ethereumTypedData) Hash() (EthereumHash, error) {
	rawData, err := typedData.RawData()
	if err != nil {
		return EthereumHash{}, nil
	}
	return Keccak256EthereumHash(rawData), nil
}

// HashStructMap generates a keccak256 hash of the encoding of the provided data
func (typedData *ethereumTypedData) HashStructMap(primaryType string,
	data map[string]interface{}) (EthereumHash, error) {

	encodedData, err := typedData.EncodeData(primaryType, data, 1)
	if err != nil {
		return EthereumHash{}, err
	}
	return Keccak256EthereumHash(encodedData), nil
}

// Dependencies returns an array of custom types ordered by their hierarchical reference tree
func (typedData *ethereumTypedData) Dependencies(primaryType string, found []string) []string {
	includes := func(arr []string, str string) bool {
		for _, obj := range arr {
			if obj == str {
				return true
			}
		}
		return false
	}

	if includes(found, primaryType) {
		return found
	}
	if typedData.Types[primaryType] == nil {
		return found
	}
	found = append(found, primaryType)
	for _, field := range typedData.Types[primaryType] {
		for _, dep := range typedData.Dependencies(field.Type, found) {
			if !includes(found, dep) {
				found = append(found, dep)
			}
		}
	}
	return found
}

// EncodeType generates the following encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func (typedData *ethereumTypedData) EncodeType(primaryType string) []byte {
	// Get dependencies primary first, then alphabetical
	deps := typedData.Dependencies(primaryType, []string{})
	if len(deps) > 0 {
		slicedDeps := deps[1:]
		sort.Strings(slicedDeps)
		deps = append([]string{primaryType}, slicedDeps...)
	}

	// Format as a string with fields
	var buffer bytes.Buffer
	for _, dep := range deps {
		buffer.WriteString(dep)
		buffer.WriteString("(")
		for _, obj := range typedData.Types[dep] {
			buffer.WriteString(obj.Type)
			buffer.WriteString(" ")
			buffer.WriteString(obj.Name)
			buffer.WriteString(",")
		}
		buffer.Truncate(buffer.Len() - 1)
		buffer.WriteString(")")
	}
	return buffer.Bytes()
}

// TypeHash creates the keccak256 hash  of the data
func (typedData *ethereumTypedData) TypeHash(primaryType string) []byte {
	return crypto.MustKeccak256(typedData.EncodeType(primaryType)).Bytes()
}

// EncodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func (typedData *ethereumTypedData) EncodeData(primaryType string, data map[string]interface{}, depth int) ([]byte, error) {
	if err := typedData.validate(); err != nil {
		return nil, err
	}

	buffer := bytes.Buffer{}

	// Verify extra data
	if exp, got := len(typedData.Types[primaryType]), len(data); exp < got {
		return nil, errors.Errorf("there is extra data provided in the message (%d < %d)", exp, got)
	}

	// Add typehash
	buffer.Write(typedData.TypeHash(primaryType))

	// Add field contents. Structs and arrays have special handlers.
	for _, field := range typedData.Types[primaryType] {
		encType := field.Type
		encValue := data[field.Name]
		if encType[len(encType)-1:] == "]" {
			arrayValue, ok := encValue.([]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}

			arrayBuffer := bytes.Buffer{}
			parsedType := strings.Split(encType, "[")[0]
			for _, item := range arrayValue {
				if typedData.Types[parsedType] != nil {
					mapValue, ok := item.(map[string]interface{})
					if !ok {
						return nil, dataMismatchError(parsedType, item)
					}
					encodedData, err := typedData.EncodeData(parsedType, mapValue, depth+1)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(encodedData)
				} else {
					bytesValue, err := typedData.EncodePrimitiveValue(parsedType, item, depth)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(bytesValue)
				}
			}

			buffer.Write(crypto.MustKeccak256(arrayBuffer.Bytes()).Bytes())
		} else if typedData.Types[field.Type] != nil {
			mapValue, ok := encValue.(map[string]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}
			encodedData, err := typedData.EncodeData(field.Type, mapValue, depth+1)
			if err != nil {
				return nil, err
			}
			buffer.Write(crypto.MustKeccak256(encodedData).Bytes())
		} else {
			byteValue, err := typedData.EncodePrimitiveValue(encType, encValue, depth)
			if err != nil {
				return nil, err
			}
			buffer.Write(byteValue)
		}
	}
	return buffer.Bytes(), nil
}

// Attempt to parse bytes in different formats: byte array, hex string, []byte.
func parseBytes(encType interface{}) ([]byte, bool) {
	switch v := encType.(type) {
	case []byte:
		return v, true
	case string:
		b, err := DecodeFromHexString(v)
		if err != nil {
			return nil, false
		}
		return b, true
	default:
		return nil, false
	}
}

func bigUint64(x uint64) *big.Int {
	i := big.Int{}
	return i.SetUint64(x)
}

func parseInteger(encType string, encValue interface{}) (*big.Int, error) {
	var (
		length int
		signed = strings.HasPrefix(encType, "int")
		b      *big.Int
	)

	if encType == "int" || encType == "uint" {
		length = 256
	} else {
		lengthStr := ""
		if strings.HasPrefix(encType, "uint") {
			lengthStr = strings.TrimPrefix(encType, "uint")
		} else {
			lengthStr = strings.TrimPrefix(encType, "int")
		}
		atoiSize, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, errors.Errorf("invalid size on integer: %v", lengthStr)
		}
		length = atoiSize
	}

	switch v := encValue.(type) {
	case *big.Int:
		b = v
	case *hexOrDecimal256:
		b = (*big.Int)(v)
	case string:
		var hexIntValue hexOrDecimal256
		if err := hexIntValue.UnmarshalText([]byte(v)); err != nil {
			return nil, err
		}
		b = (*big.Int)(&hexIntValue)

	case float64:
		// standard JSON unmarshal parses non-strings as float64. Fail if we cannot
		// convert it losslessly
		if float64(int64(v)) == v {
			b = big.NewInt(int64(v))
		} else {
			return nil, fmt.Errorf("invalid float value %v for type %v", v, encType)
		}
	case float32:
		// Fail if we cannot convert it losslessly
		if float32(int64(v)) == v {
			b = big.NewInt(int64(v))
		} else {
			return nil, fmt.Errorf("invalid float value %v for type %v", v, encType)
		}

	case int:
		b = big.NewInt(int64(v))
	case uint:
		b = bigUint64(uint64(v))

	case int64:
		b = big.NewInt(v)
	case uint64:
		b = bigUint64(v)

	case int32:
		b = big.NewInt(int64(v))
	case uint32:
		b = bigUint64(uint64(v))

	case int16:
		b = big.NewInt(int64(v))
	case uint16:
		b = bigUint64(uint64(v))

	case int8:
		b = big.NewInt(int64(v))
	case byte:
		b = bigUint64(uint64(v))
	}

	if b == nil {
		return nil, errors.Errorf("invalid integer value %v/%T for type %v", encValue, encValue, encType)
	}
	if b.BitLen() > length {
		return nil, errors.Errorf("integer larger than '%v'", encType)
	}
	if !signed && b.Sign() == -1 {
		return nil, errors.Errorf("invalid negative value for unsigned type %v", encType)
	}
	return b, nil
}

// EncodePrimitiveValue deals with the primitive values found
// while searching through the typed data
func (typedData *ethereumTypedData) EncodePrimitiveValue(encType string, encValue interface{}, depth int) ([]byte, error) {
	switch encType {
	case "address":
		stringValue, ok := encValue.(string)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}

		b, err := DecodeFromHexString(stringValue)
		if err != nil {
			return nil, dataMismatchError(encType, encValue)
		}

		retval := make([]byte, 32)
		copy(retval[12:], BytesToEthereumAddress(b).Bytes())

		return retval, nil
	case "bool":
		boolValue, ok := encValue.(bool)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		if boolValue {
			return paddedEthereumBigIntToBytes(big1, 32), nil
		}
		return paddedEthereumBigIntToBytes(big0, 32), nil
	case "string":
		strVal, ok := encValue.(string)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.MustKeccak256([]byte(strVal)).Bytes(), nil
	case "bytes":
		bytesValue, ok := parseBytes(encValue)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.MustKeccak256(bytesValue).Bytes(), nil
	}
	if strings.HasPrefix(encType, "bytes") {
		lengthStr := strings.TrimPrefix(encType, "bytes")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, errors.Errorf("invalid size on bytes: %v", lengthStr)
		}
		if length < 0 || length > 32 {
			return nil, errors.Errorf("invalid size on bytes: %d", length)
		}
		if byteValue, ok := parseBytes(encValue); !ok || len(byteValue) != length {
			return nil, dataMismatchError(encType, encValue)
		} else {
			// Right-pad the bits
			dst := make([]byte, 32)
			copy(dst, byteValue)
			return dst, nil
		}
	}
	if strings.HasPrefix(encType, "int") || strings.HasPrefix(encType, "uint") {
		b, err := parseInteger(encType, encValue)
		if err != nil {
			return nil, err
		}
		return ethereumU256ToBytes(b), nil
	}
	return nil, errors.Errorf("unrecognized type '%s'", encType)

}

// dataMismatchError generates an error for a mismatch between
// the provided type and data
func dataMismatchError(encType string, encValue interface{}) error {
	return errors.Errorf("provided data '%v' doesn't match type '%s'", encValue, encType)
}

// validate makes sure the types are sound
func (typedData *ethereumTypedData) validate() error {
	if err := typedData.Types.validate(); err != nil {
		return err
	}
	if err := typedData.Domain.validate(); err != nil {
		return err
	}
	return nil
}

// Map generates a map version of the typed data
func (typedData *ethereumTypedData) Map() map[string]interface{} {
	dataMap := map[string]interface{}{
		"types":       typedData.Types,
		"domain":      typedData.Domain.Map(),
		"primaryType": typedData.PrimaryType,
		"message":     typedData.Message,
	}
	return dataMap
}

var (
	typedDataReferenceTypeRegexp = regexp.MustCompile(`^[A-Z](\w*)(\[\])?$`)

	validPrimitiveTypes = map[string]struct{}{
		"address":   {},
		"address[]": {},
		"bool":      {},
		"bool[]":    {},
		"string":    {},
		"string[]":  {},

		"bytes":     {},
		"bytes[]":   {},
		"bytes1":    {},
		"bytes1[]":  {},
		"bytes2":    {},
		"bytes2[]":  {},
		"bytes3":    {},
		"bytes3[]":  {},
		"bytes4":    {},
		"bytes4[]":  {},
		"bytes5":    {},
		"bytes5[]":  {},
		"bytes6":    {},
		"bytes6[]":  {},
		"bytes7":    {},
		"bytes7[]":  {},
		"bytes8":    {},
		"bytes8[]":  {},
		"bytes9":    {},
		"bytes9[]":  {},
		"bytes10":   {},
		"bytes10[]": {},
		"bytes11":   {},
		"bytes11[]": {},
		"bytes12":   {},
		"bytes12[]": {},
		"bytes13":   {},
		"bytes13[]": {},
		"bytes14":   {},
		"bytes14[]": {},
		"bytes15":   {},
		"bytes15[]": {},
		"bytes16":   {},
		"bytes16[]": {},
		"bytes17":   {},
		"bytes17[]": {},
		"bytes18":   {},
		"bytes18[]": {},
		"bytes19":   {},
		"bytes19[]": {},
		"bytes20":   {},
		"bytes20[]": {},
		"bytes21":   {},
		"bytes21[]": {},
		"bytes22":   {},
		"bytes22[]": {},
		"bytes23":   {},
		"bytes23[]": {},
		"bytes24":   {},
		"bytes24[]": {},
		"bytes25":   {},
		"bytes25[]": {},
		"bytes26":   {},
		"bytes26[]": {},
		"bytes27":   {},
		"bytes27[]": {},
		"bytes28":   {},
		"bytes28[]": {},
		"bytes29":   {},
		"bytes29[]": {},
		"bytes30":   {},
		"bytes30[]": {},
		"bytes31":   {},
		"bytes31[]": {},
		"bytes32":   {},
		"bytes32[]": {},

		"int":      {},
		"int[]":    {},
		"int8":     {},
		"int8[]":   {},
		"int16":    {},
		"int16[]":  {},
		"int32":    {},
		"int32[]":  {},
		"int64":    {},
		"int64[]":  {},
		"int128":   {},
		"int128[]": {},
		"int256":   {},
		"int256[]": {},

		"uint":      {},
		"uint[]":    {},
		"uint8":     {},
		"uint8[]":   {},
		"uint16":    {},
		"uint16[]":  {},
		"uint32":    {},
		"uint32[]":  {},
		"uint64":    {},
		"uint64[]":  {},
		"uint128":   {},
		"uint128[]": {},
		"uint256":   {},
		"uint256[]": {},
	}
)

// Checks if the primitive value is valid
func isPrimitiveTypeValid(primitiveType string) bool {
	_, ok := validPrimitiveTypes[primitiveType]
	return ok
}

// validate checks if the types object is conformant to the specs
func (t ethereumTypedDataTypes) validate() error {
	for typeKey, typeArr := range t {
		if len(typeKey) == 0 {
			return errors.Errorf("empty type key")
		}
		for i, typeObj := range typeArr {
			if len(typeObj.Type) == 0 {
				return errors.Errorf("type %q:%d: empty Type", typeKey, i)
			}
			if len(typeObj.Name) == 0 {
				return errors.Errorf("type %q:%d: empty Name", typeKey, i)
			}
			if typeKey == typeObj.Type {
				return errors.Errorf("type %q cannot reference itself", typeObj.Type)
			}
			if typeObj.isReferenceType() {
				if _, exist := t[typeObj.typeName()]; !exist {
					return errors.Errorf("reference type %q is undefined", typeObj.Type)
				}
				if !typedDataReferenceTypeRegexp.MatchString(typeObj.Type) {
					return errors.Errorf("unknown reference type %q", typeObj.Type)
				}
			} else if !isPrimitiveTypeValid(typeObj.Type) {
				return errors.Errorf("unknown type %q", typeObj.Type)
			}
		}
	}
	return nil
}

// validate checks if the given domain is valid, i.e. contains at least
// the minimum viable keys and values
func (domain *ethereumTypedDataDomain) validate() error {
	if domain.ChainId == nil && len(domain.Name) == 0 && len(domain.Version) == 0 {
		return errors.New("domain is undefined")
	}
	return nil
}

// Map is a helper function to generate a map version of the domain
func (domain *ethereumTypedDataDomain) Map() map[string]interface{} {
	dataMap := make(map[string]interface{}, 3)

	if domain.ChainId != nil {
		dataMap["chainId"] = domain.ChainId
	}

	if len(domain.Name) > 0 {
		dataMap["name"] = domain.Name
	}

	if len(domain.Version) > 0 {
		dataMap["version"] = domain.Version
	}

	return dataMap
}
