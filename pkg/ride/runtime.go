package ride

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	byteVectorBase58Prefix = "base58"
	byteVectorBase64Prefix = "base64"

	addressTypeName         = "Address"
	aliasTypeName           = "Alias"
	assetPairTypeName       = "AssetPair"
	assetTypeName           = "Asset"
	attachedPaymentTypeName = "AttachedPayment"
	balanceDetailsTypeName  = "BalanceDetails"
	bigIntTypeName          = "BigInt"
	binaryEntryTypeName     = "BinaryEntry"
	blockInfoTypeName       = "BlockInfo"
	booleanEntryTypeName    = "BooleanEntry"
	booleanTypeName         = "Boolean"
	burnTypeName            = "Burn"
	bytesTypeName           = "ByteVector"
	dataEntryTypeName       = "DataEntry"
	deleteEntryTypeName     = "DeleteEntry"
	intTypeName             = "Int"
	integerEntryTypeName    = "IntegerEntry"
	invocationTypeName      = "Invocation"
	issueTypeName           = "Issue"
	leaseCancelTypeName     = "LeaseCancel"
	leaseTypeName           = "Lease"
	listTypeName            = "List[Any]"
	orderTypeName           = "Order"
	recipientTypeName       = "Recipient"
	reissueTypeName         = "Reissue"
	scriptResultTypeName    = "ScriptResult"
	scriptTransferTypeName  = "ScriptTransfer"
	sponsorFeeTypeName      = "SponsorFee"
	stringEntryTypeName     = "StringEntry"
	stringTypeName          = "String"
	transferTypeName        = "Transfer"
	transferSetTypeName     = "TransferSet"
	unitTypeName            = "Unit"
	writeSetTypeName        = "WriteSet"

	burnTransactionTypeName             = "BurnTransaction"
	createAliasTransactionTypeName      = "CreateAliasTransaction"
	dataTransactionTypeName             = "DataTransaction"
	exchangeTransactionTypeName         = "ExchangeTransaction"
	genesisTransactionTypeName          = "GenesisTransaction"
	invokeExpressionTransactionTypeName = "InvokeExpressionTransaction"
	invokeScriptTransactionTypeName     = "InvokeScriptTransaction"
	issueTransactionTypeName            = "IssueTransaction"
	leaseCancelTransactionTypeName      = "LeaseCancelTransaction"
	leaseTransactionTypeName            = "LeaseTransaction"
	massTransferTransactionTypeName     = "MassTransferTransaction"
	paymentTransactionTypeName          = "PaymentTransaction"
	reissueTransactionTypeName          = "ReissueTransaction"
	setAssetScriptTransactionTypeName   = "SetAssetScriptTransaction"
	setScriptTransactionTypeName        = "SetScriptTransaction"
	sponsorFeeTransactionTypeName       = "SponsorFeeTransaction"
	transferTransactionTypeName         = "TransferTransaction"
	updateAssetInfoTransactionTypeName  = "UpdateAssetInfoTransaction"

	aliasField                 = "alias"
	amountAssetField           = "amountAsset"
	amountField                = "amount"
	argsField                  = "args"
	assetField                 = "asset"
	assetIDField               = "assetId"
	assetPairField             = "assetPair"
	attachmentField            = "attachment"
	availableField             = "available"
	baseTargetField            = "baseTarget"
	bodyBytesField             = "bodyBytes"
	buyMatcherFeeField         = "buyMatcherFee"
	buyOrderField              = "buyOrder"
	bytesField                 = "bytes"
	callerField                = "caller"
	callerPublicKeyField       = "callerPublicKey"
	compiledScriptField        = "compiledScript"
	dAppField                  = "dApp"
	dataField                  = "data"
	decimalsField              = "decimals"
	descriptionField           = "description"
	effectiveField             = "effective"
	expirationField            = "expiration"
	expressionField            = "expression"
	feeAssetIDField            = "feeAssetId"
	feeField                   = "fee"
	functionField              = "function"
	generatingField            = "generating"
	generationSignatureField   = "generationSignature"
	generatorField             = "generator"
	generatorPublicKeyField    = "generatorPublicKey"
	heightField                = "height"
	idField                    = "id"
	instanceField              = "$instance"
	isReissuableField          = "isReissuable"
	issuePublicKeyField        = "issuerPublicKey"
	issuerField                = "issuer"
	keyField                   = "key"
	leaseIDField               = "leaseId"
	matcherFeeAssetIDField     = "matcherFeeAssetId"
	matcherFeeField            = "matcherFee"
	matcherPublicKeyField      = "matcherPublicKey"
	minSponsoredAssetFeeField  = "minSponsoredAssetFee"
	minSponsoredFeeField       = "minSponsoredFee"
	nameField                  = "name"
	nonceField                 = "nonce"
	orderTypeField             = "orderType"
	originCallerField          = "originCaller"
	originCallerPublicKeyField = "originCallerPublicKey"
	paymentField               = "payment"
	paymentsField              = "payments"
	priceAssetField            = "priceAsset"
	priceField                 = "price"
	proofsField                = "proofs"
	quantityField              = "quantity"
	recipientField             = "recipient"
	regularField               = "regular"
	reissuableField            = "reissuable"
	scriptField                = "script"
	scriptedField              = "scripted"
	sellMatcherFeeField        = "sellMatcherFee"
	sellOrderField             = "sellOrder"
	senderField                = "sender"
	senderPublicKeyField       = "senderPublicKey"
	sponsoredField             = "sponsored"
	timestampField             = "timestamp"
	totalAmountField           = "totalAmount"
	transactionIDField         = "transactionId"
	transferSetField           = "transferSet"
	transfersCountField        = "transferCount"
	transfersField             = "transfers"
	valueField                 = "value"
	versionField               = "version"
	vrfField                   = "vrf"
	writeSetField              = "writeSet"
)

type rideType interface {
	instanceOf() string
	eq(other rideType) bool
	get(prop string) (rideType, error)
	lines() []string
	fmt.Stringer
}

type rideProven interface {
	getProofs() rideList
	setProofs(proofs rideList) rideProven
}

type rideBoolean bool

func (b rideBoolean) instanceOf() string {
	return booleanTypeName
}

func (b rideBoolean) eq(other rideType) bool {
	if o, ok := other.(rideBoolean); ok {
		return b == o
	}
	return false
}

func (b rideBoolean) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
}

func (b rideBoolean) lines() []string {
	return []string{strconv.FormatBool(bool(b))}
}

func (b rideBoolean) String() string {
	return strconv.FormatBool(bool(b))
}

type rideInt int64

func (l rideInt) instanceOf() string {
	return intTypeName
}

func (l rideInt) eq(other rideType) bool {
	if o, ok := other.(rideInt); ok {
		return l == o
	}
	return false
}

func (l rideInt) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

func (l rideInt) lines() []string {
	return []string{strconv.FormatInt(int64(l), 10)}
}

func (l rideInt) String() string {
	return strconv.FormatInt(int64(l), 10)
}

type rideBigInt struct {
	v *big.Int
}

func (l rideBigInt) instanceOf() string {
	return bigIntTypeName
}

func (l rideBigInt) eq(other rideType) bool {
	if o, ok := other.(rideBigInt); ok {
		return l.v.Cmp(o.v) == 0
	}
	return false
}

func (l rideBigInt) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

func (l rideBigInt) lines() []string {
	return []string{l.v.String()}
}

func (l rideBigInt) String() string {
	return l.v.String()
}

type rideString string

func (s rideString) instanceOf() string {
	return stringTypeName
}

func (s rideString) eq(other rideType) bool {
	if o, ok := other.(rideString); ok {
		return s == o
	}
	return false
}

func (s rideString) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", s.instanceOf(), prop)
}

func (s rideString) lines() []string {
	return []string{strconv.Quote(string(s))}
}

func (s rideString) String() string {
	return strconv.Quote(string(s))
}

type rideBytes []byte

func (b rideBytes) instanceOf() string {
	return bytesTypeName
}

func (b rideBytes) eq(other rideType) bool {
	if o, ok := other.(rideBytes); ok {
		return bytes.Equal(b, o)
	}
	return false
}

func (b rideBytes) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
}

func (b rideBytes) lines() []string {
	return []string{b.String()}
}

func (b rideBytes) String() string {
	str, prefix := b.stringAndPrefix()
	return prefix + "'" + str + "'"
}

// stringAndPrefix function return string representation of byte slice and the name of encoding used to produce it.
// In Scala implementation the string representation of byte arrays switches from Base58 to Base64 for arrays of size
// bigger than 1024 bytes.
func (b rideBytes) stringAndPrefix() (string, string) {
	if len(b) > 1024 {
		return base64.StdEncoding.EncodeToString(b), byteVectorBase64Prefix
	}
	return base58.Encode(b), byteVectorBase58Prefix
}

func (b rideBytes) scalaString() string {
	str, prefix := b.stringAndPrefix()
	if prefix == byteVectorBase58Prefix {
		return str
	}
	return prefix + ":" + str
}

type rideAddress proto.WavesAddress

func (a rideAddress) instanceOf() string {
	return addressTypeName
}

func (a rideAddress) eq(other rideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case rideBytes:
		return bytes.Equal(a[:], o[:])
	default:
		return false
	}
}

func (a rideAddress) get(prop string) (rideType, error) {
	switch prop {
	case bytesField:
		return rideBytes(a[:]), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

func makeLinesForAddressBytes(b []byte) []string {
	return []string{
		addressTypeName + "(",
		"\t" + bytesField + " = " + rideBytes(b).String(),
		")",
	}
}

func (a rideAddress) lines() []string {
	return makeLinesForAddressBytes(a[:])
}

func (a rideAddress) String() string {
	return strings.Join(a.lines(), "\n")
}

type rideAddressLike []byte

func (a rideAddressLike) instanceOf() string {
	return addressTypeName
}

func (a rideAddressLike) eq(other rideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case rideBytes:
		return bytes.Equal(a[:], o[:])
	case rideAddressLike:
		return bytes.Equal(a[:], o[:])
	default:
		return false
	}
}

func (a rideAddressLike) get(prop string) (rideType, error) {
	switch prop {
	case bytesField:
		return rideBytes(a[:]), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

func (a rideAddressLike) lines() []string {
	return makeLinesForAddressBytes(a[:])
}

func (a rideAddressLike) String() string {
	return strings.Join(a.lines(), "\n")
}

type rideAlias proto.Alias

func (a rideAlias) instanceOf() string {
	return aliasTypeName
}

func (a rideAlias) eq(other rideType) bool {
	switch o := other.(type) {
	case rideAlias:
		return a.Alias == o.Alias
	default:
		return false
	}
}

func (a rideAlias) get(prop string) (rideType, error) {
	switch prop {
	case aliasField:
		return rideString(a.Alias), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

func (a rideAlias) String() string {
	return strings.Join(a.lines(), "\n")
}

func (a rideAlias) lines() []string {
	return []string{
		aliasTypeName + "(",
		"\t" + aliasField + " = " + strconv.Quote(a.Alias),
		")",
	}
}

type rideUnit struct{}

func (a rideUnit) instanceOf() string {
	return unitTypeName
}

func (a rideUnit) eq(other rideType) bool {
	return a.instanceOf() == other.instanceOf()
}

func (a rideUnit) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

func (a rideUnit) lines() []string {
	return []string{unitTypeName}
}

func (a rideUnit) String() string {
	return unitTypeName
}

type rideNamedType struct {
	name string
}

func (a rideNamedType) instanceOf() string {
	return a.name
}

func (a rideNamedType) eq(other rideType) bool {
	return a.instanceOf() == other.instanceOf()
}

func (a rideNamedType) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

func (a rideNamedType) lines() []string {
	return []string{a.name}
}

func (a rideNamedType) String() string {
	return a.name
}

type rideList []rideType

func (a rideList) instanceOf() string {
	return listTypeName
}

func (a rideList) eq(other rideType) bool {
	if a.instanceOf() != other.instanceOf() {
		return false
	}
	o, ok := other.(rideList)
	if !ok {
		return false
	}
	if len(a) != len(o) {
		return false
	}
	for i, item := range a {
		if !item.eq(o[i]) {
			return false
		}
	}
	return true
}

func (a rideList) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

func (a rideList) lines() []string {
	return []string{a.String()}
}

func (a rideList) String() string {
	ss := make([]string, len(a))
	for i, e := range a {
		ss[i] = e.String()
	}
	return "[" + strings.Join(ss, ", ") + "]"
}

type (
	rideFunction    func(env environment, args ...rideType) (rideType, error)
	rideConstructor func(environment) rideType
)

func fieldLines(key string, valueLines []string) []string {
	l := len(valueLines)
	r := make([]string, l)
	r[0] = "\t" + key + " = " + valueLines[0]
	for i := 1; i < l; i++ {
		r[i] = "\t" + valueLines[i]
	}
	return r
}

//go:generate moq -out runtime_moq_test.go . environment:mockRideEnvironment
type environment interface {
	scheme() byte
	height() rideInt
	transaction() rideType
	this() rideType
	block() rideType
	txID() rideType // Invoke transaction ID
	state() types.SmartState
	timestamp() uint64
	setNewDAppAddress(address proto.WavesAddress)
	checkMessageLength(int) bool
	takeString(s string, n int) rideString
	invocation() rideType // Invocation object made of invoke transaction
	setInvocation(inv rideType)
	setLibVersion(v ast.LibraryVersion)
	libVersion() (ast.LibraryVersion, error)
	validateInternalPayments() bool
	blockV5Activated() bool
	rideV6Activated() bool
	consensusImprovementsActivated() bool
	invokeExpressionActivated() bool // TODO: check naming
	internalPaymentsValidationHeight() uint64
	maxDataEntriesSize() int
	isProtobufTx() bool
	complexityCalculator() complexityCalculator
}
