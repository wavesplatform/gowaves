// Benchmarks of the native Ride functions from the functions catalogues of all library versions.
//
// Every distinct native function of the catalogues has a benchmark case (or an explicit skip with a reason,
// see nativeBenchSkips), the coverage is checked by TestNativeFunctionBenchmarkCoverage.
// Sub-benchmarks are named after the Go identifier of the function and the variant of the arguments,
// for example "BenchmarkNativeFunctions/toBase58/large".
//
// Run the benchmarks and save the results for comparison with benchstat:
//
//	go test -run '^$' -bench BenchmarkNativeFunctions -benchmem -count 10 ./pkg/ride/ > new.txt
//	go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
//
// Check the coverage of the catalogues and run every case once as a quick smoke test:
//
//	go test -run TestNativeFunctionBenchmarkCoverage -bench BenchmarkNativeFunctions -benchtime 1x ./pkg/ride/
//
// Run a single function or a family of functions with a sub-benchmark filter:
//
//	go test -run '^$' -bench 'BenchmarkNativeFunctions/(split|makeString)' -benchmem ./pkg/ride/
//
// Collect CPU and allocation profiles and open the flame graphs in a browser (View -> Flame Graph):
//
//	go test -run '^$' -bench 'BenchmarkNativeFunctions/median' -benchmem -cpuprofile cpu.out -memprofile mem.out \
//		./pkg/ride/
//	go tool pprof -http=:8080 cpu.out
//	go tool pprof -http=:8081 -sample_index=alloc_space mem.out

package ride //nolint:testpackage // White-box benchmarks of the unexported native functions.

import (
	"bytes"
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	sh256 "crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// nativeBenchSkips lists the native functions that have no benchmark case, with the reason.
var nativeBenchSkips = map[string]string{ //nolint:gochecknoglobals // Read-only table of the benchmark skips.
	"invoke":          "calls another DApp script, covered by the evaluator benchmarks",
	"reentrantInvoke": "calls another DApp script, covered by the evaluator benchmarks",
	"throw":           "always fails, there is no success path to measure",
	"throw0":          "always fails, there is no success path to measure",

	"assetV3Constructor":                       nativeBenchObjectConstructorSkip,
	"assetV4Constructor":                       nativeBenchObjectConstructorSkip,
	"blockInfoV3Constructor":                   nativeBenchObjectConstructorSkip,
	"blockInfoV4Constructor":                   nativeBenchObjectConstructorSkip,
	"blockInfoV7Constructor":                   nativeBenchObjectConstructorSkip,
	"invocationV3Constructor":                  nativeBenchObjectConstructorSkip,
	"invocationV4Constructor":                  nativeBenchObjectConstructorSkip,
	"invocationV5Constructor":                  nativeBenchObjectConstructorSkip,
	"orderV1Constructor":                       nativeBenchObjectConstructorSkip,
	"orderV8Constructor":                       nativeBenchObjectConstructorSkip,
	"burnTransactionConstructor":               nativeBenchObjectConstructorSkip,
	"commitToGenerationTransactionConstructor": nativeBenchObjectConstructorSkip,
	"createAliasTransactionConstructor":        nativeBenchObjectConstructorSkip,
	"dataTransactionConstructor":               nativeBenchObjectConstructorSkip,
	"exchangeTransactionConstructor":           nativeBenchObjectConstructorSkip,
	"genesisTransactionConstructor":            nativeBenchObjectConstructorSkip,
	"invokeExpressionTransactionConstructor":   nativeBenchObjectConstructorSkip,
	"invokeScriptTransactionV3Constructor":     nativeBenchObjectConstructorSkip,
	"invokeScriptTransactionV4Constructor":     nativeBenchObjectConstructorSkip,
	"issueTransactionConstructor":              nativeBenchObjectConstructorSkip,
	"leaseCancelTransactionConstructor":        nativeBenchObjectConstructorSkip,
	"leaseTransactionConstructor":              nativeBenchObjectConstructorSkip,
	"massTransferTransactionConstructor":       nativeBenchObjectConstructorSkip,
	"paymentTransactionConstructor":            nativeBenchObjectConstructorSkip,
	"reissueTransactionConstructor":            nativeBenchObjectConstructorSkip,
	"setAssetScriptTransactionConstructor":     nativeBenchObjectConstructorSkip,
	"setScriptTransactionConstructor":          nativeBenchObjectConstructorSkip,
	"sponsorFeeTransactionConstructor":         nativeBenchObjectConstructorSkip,
	"transferTransactionConstructor":           nativeBenchObjectConstructorSkip,
	"updateAssetInfoTransactionConstructor":    nativeBenchObjectConstructorSkip,
}

const nativeBenchObjectConstructorSkip = "generated constructor of a state or transaction object, only checks " +
	"the types of arguments like the benchmarked action constructors, not used by real scripts"

const (
	nativeBenchScheme  = proto.TestNetScheme
	nativeBenchHeight  = 1_000_000
	nativeBenchBalance = 100_000_000_000
	nativeBenchKey     = "key"
	nativeBenchAlias   = "benchmark"
)

// nativeBenchState is a minimal types.EnrichedSmartState for the native functions benchmarks.
// It embeds the interface and overrides only the methods used by the native functions, so mock reflection
// doesn't show up in the profiles. All results are prepared in advance, so the state doesn't allocate.
// Calls of other methods panic on the nil embedded interface.
type nativeBenchState struct {
	types.EnrichedSmartState

	address       proto.WavesAddress
	script        proto.Script
	tx            proto.Transaction
	integerEntry  *proto.IntegerDataEntry
	booleanEntry  *proto.BooleanDataEntry
	stringEntry   *proto.StringDataEntry
	binaryEntry   *proto.BinaryDataEntry
	fullBalance   *proto.FullWavesBalance
	assetInfo     *proto.AssetInfo
	fullAssetInfo *proto.FullAssetInfo
	blockInfo     *proto.BlockInfo
}

func (s *nativeBenchState) AddingBlockHeight() (uint64, error) { return nativeBenchHeight, nil }

func (s *nativeBenchState) IsNotFound(err error) bool { return errors.Is(err, proto.ErrNotFound) }

func (s *nativeBenchState) NewestAddrByAlias(proto.Alias) (proto.WavesAddress, error) {
	return s.address, nil
}

func (s *nativeBenchState) NewestScriptBytesByAccount(proto.Recipient) (proto.Script, error) {
	return s.script, nil
}

func (s *nativeBenchState) IsStateUntouched(proto.Recipient) (bool, error) { return true, nil }

func (s *nativeBenchState) NewestTransactionByID([]byte) (proto.Transaction, error) { return s.tx, nil }

func (s *nativeBenchState) NewestTransactionHeightByID([]byte) (uint64, error) {
	return nativeBenchHeight - 1, nil
}

func (s *nativeBenchState) NewestWavesBalance(proto.Recipient) (uint64, error) {
	return nativeBenchBalance, nil
}

func (s *nativeBenchState) NewestAssetBalance(proto.Recipient, crypto.Digest) (uint64, error) {
	return nativeBenchBalance, nil
}

func (s *nativeBenchState) NewestFullWavesBalance(proto.Recipient) (*proto.FullWavesBalance, error) {
	return s.fullBalance, nil
}

func (s *nativeBenchState) RetrieveNewestIntegerEntry(proto.Recipient, string) (*proto.IntegerDataEntry, error) {
	return s.integerEntry, nil
}

func (s *nativeBenchState) RetrieveNewestBooleanEntry(proto.Recipient, string) (*proto.BooleanDataEntry, error) {
	return s.booleanEntry, nil
}

func (s *nativeBenchState) RetrieveNewestStringEntry(proto.Recipient, string) (*proto.StringDataEntry, error) {
	return s.stringEntry, nil
}

func (s *nativeBenchState) RetrieveNewestBinaryEntry(proto.Recipient, string) (*proto.BinaryDataEntry, error) {
	return s.binaryEntry, nil
}

func (s *nativeBenchState) NewestAssetInfo(crypto.Digest) (*proto.AssetInfo, error) {
	return s.assetInfo, nil
}

func (s *nativeBenchState) NewestFullAssetInfo(crypto.Digest) (*proto.FullAssetInfo, error) {
	return s.fullAssetInfo, nil
}

func (s *nativeBenchState) NewestBlockInfoByHeight(proto.Height) (*proto.BlockInfo, error) {
	return s.blockInfo, nil
}

// nativeBenchFixture holds the accounts, transactions and the environment shared by the benchmark cases.
type nativeBenchFixture struct {
	accounts evaluatorBenchAccounts
	transfer *proto.TransferWithProofs
	state    *nativeBenchState
	env      *EvaluationEnvironment
}

func newNativeBenchFixture(tb testing.TB) *nativeBenchFixture {
	tb.Helper()
	a := newEvaluatorBenchAccounts(tb)
	tx := a.transfer(tb)
	generatorPK := a.senderPK
	assetInfo := proto.AssetInfo{
		ID: a.asset, IssueHeight: nativeBenchHeight - 10, Issuer: a.sender, Decimals: 8,
		Quantity:        1_000_000_000_000,
		IssuerPublicKey: a.senderPK,
		Reissuable:      true,
	}
	st := &nativeBenchState{
		address:      a.dApp,
		script:       proto.Script(bytes.Repeat([]byte{0x06, 0x01, 0x02}, 100)),
		tx:           tx,
		integerEntry: &proto.IntegerDataEntry{Key: nativeBenchKey, Value: 12345},
		booleanEntry: &proto.BooleanDataEntry{Key: nativeBenchKey, Value: true},
		stringEntry:  &proto.StringDataEntry{Key: nativeBenchKey, Value: "native functions benchmark"},
		binaryEntry:  &proto.BinaryDataEntry{Key: nativeBenchKey, Value: bytes.Repeat([]byte{0xab}, 32)},
		fullBalance: &proto.FullWavesBalance{
			Regular: nativeBenchBalance, Generating: nativeBenchBalance, Available: nativeBenchBalance,
			Effective: nativeBenchBalance,
		},
		assetInfo: &assetInfo,
		fullAssetInfo: &proto.FullAssetInfo{
			AssetInfo: assetInfo, Name: "Benchmark", Description: "Native functions benchmark asset",
		},
		blockInfo: &proto.BlockInfo{
			Version:             proto.ProtobufBlockVersion,
			Timestamp:           1_700_000_000_000,
			Height:              nativeBenchHeight - 1,
			BaseTarget:          142244892,
			Generator:           a.sender,
			GeneratorPublicKey:  generatorPK,
			GenerationSignature: bytes.Repeat([]byte{0x01}, crypto.DigestSize),
			VRF:                 crypto.MustDigestFromBase58("5AFgQTfL1GhVUZr64N6tkmF8usX9QZsPcJbZmsX32VgK").Bytes(),
		},
	}
	env, err := NewEnvironment(nativeBenchScheme, st, 0, 0, true, true, true, true, true)
	require.NoError(tb, err)
	v := ast.CurrentMaxLibraryVersion()
	env.setLibVersion(v)
	env.SetThisFromAddress(a.dApp)
	env.ChooseSizeCheck(v)
	env.SetTimestamp(1_700_000_000_000)
	env.ChooseTakeString(true)
	env.ChooseMaxDataEntriesSize(true)
	require.NoError(tb, env.SetTransaction(tx))
	require.NoError(tb, env.SetLastBlockFromBlockInfo(st.blockInfo))
	return &nativeBenchFixture{accounts: a, transfer: tx, state: st, env: env}
}

// nativeBenchCase is a single call of a native function with the fixed arguments.
type nativeBenchCase struct {
	fn      rideFunction
	variant string // Optional variant of the arguments, e.g. "small" or "large".
	args    []rideType
}

func (c nativeBenchCase) name() string {
	if c.variant == "" {
		return rideFunctionName(c.fn)
	}
	return rideFunctionName(c.fn) + "/" + c.variant
}

// rideFunctionName returns the Go identifier of the native function.
func rideFunctionName(fn rideFunction) string {
	n := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return n[strings.LastIndexByte(n, '.')+1:]
}

// nativeBenchArgs is a set of prepared argument values shared by the benchmark cases.
type nativeBenchArgs struct {
	address    rideAddress
	alias      rideAlias
	recipients []rideType
	asset      rideByteVector
	halfUp     rideType
	down       rideType
	sha256Alg  rideType
}

func mustCallRideFunction(tb testing.TB, fn rideFunction, args ...rideType) rideType {
	tb.Helper()
	r, err := fn(nil, args...)
	require.NoError(tb, err, rideFunctionName(fn))
	return r
}

func repeatedBytes(n int) rideByteVector {
	r := make([]byte, n)
	for i := range r {
		r[i] = byte(i*7 + 3)
	}
	return r
}

func intList(n int) rideList {
	r := make(rideList, n)
	for i := range r {
		r[i] = rideInt((i*7919 + 13) % 100_003)
	}
	return r
}

func bigIntList(n int) rideList {
	base := new(big.Int).Lsh(big.NewInt(1), 200)
	r := make(rideList, n)
	for i := range r {
		v := new(big.Int).Mul(base, big.NewInt(int64((i*7919+13)%100_003)))
		r[i] = rideBigInt{v: v}
	}
	return r
}

func stringList(n int, item string) rideList {
	r := make(rideList, n)
	for i := range r {
		r[i] = rideString(item + strconv.Itoa(i))
	}
	return r
}

func bigIntValue(s string) rideBigInt {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big integer " + s)
	}
	return rideBigInt{v: v}
}

func mustDecodeBase64(tb testing.TB, s string) rideByteVector {
	tb.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(tb, err)
	return b
}

func mustDecodeBase58(tb testing.TB, s string) rideByteVector {
	tb.Helper()
	b, err := base58.Decode(s)
	require.NoError(tb, err)
	return b
}

func mustDecodeHex(tb testing.TB, s string) rideByteVector {
	tb.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(tb, err)
	return b
}

// nativeBenchCases builds the table of benchmark cases.
func nativeBenchCases(tb testing.TB, f *nativeBenchFixture) []nativeBenchCase {
	tb.Helper()
	address := rideAddress(f.accounts.dApp)
	a := nativeBenchArgs{
		address:   address,
		alias:     rideAlias(*proto.NewAlias(nativeBenchScheme, nativeBenchAlias)),
		asset:     rideByteVector(f.accounts.asset.Bytes()),
		halfUp:    mustCallRideFunction(tb, createHalfUp),
		down:      mustCallRideFunction(tb, createDown),
		sha256Alg: mustCallRideFunction(tb, createSha256),
	}
	a.recipients = []rideType{a.address, a.alias}
	var cases []nativeBenchCase
	cases = append(cases, nativeBenchCommonCases(tb, a)...)
	cases = append(cases, nativeBenchIntCases(a)...)
	cases = append(cases, nativeBenchBigIntCases(a)...)
	cases = append(cases, nativeBenchStringCases()...)
	cases = append(cases, nativeBenchBytesCases()...)
	cases = append(cases, nativeBenchListCases(tb)...)
	cases = append(cases, nativeBenchStateCases(tb, f, a)...)
	cases = append(cases, nativeBenchHashCases()...)
	cases = append(cases, nativeBenchSignatureCases(tb, f, a)...)
	cases = append(cases, nativeBenchCryptoCases(tb)...)
	cases = append(cases, nativeBenchConstructorCases(tb, a)...)
	return cases
}

func nativeBenchCommonCases(tb testing.TB, a nativeBenchArgs) []nativeBenchCase {
	tb.Helper()
	tuple := mustCallRideFunction(tb, newTuple3, rideInt(1), rideString("two"), rideBoolean(true))
	sameTuple := mustCallRideFunction(tb, newTuple3, rideInt(1), rideString("two"), rideBoolean(true))
	list := intList(100)
	return []nativeBenchCase{
		{fn: eq, variant: "int", args: []rideType{rideInt(12345), rideInt(12345)}},
		{fn: eq, variant: "string", args: []rideType{rideString("native"), rideString("native")}},
		{fn: eq, variant: "list", args: []rideType{list, intList(100)}},
		{fn: eq, variant: "object", args: []rideType{a.address, a.address}},
		{fn: eq, variant: "tuple", args: []rideType{tuple, sameTuple}},
		{fn: neq, args: []rideType{rideInt(12345), rideInt(54321)}},
		{fn: instanceOf, args: []rideType{rideInt(12345), rideString("Int")}},
		{fn: getType, args: []rideType{tuple}},
		{fn: extract, args: []rideType{rideInt(12345)}},
		{fn: isDefined, args: []rideType{rideInt(12345)}},
		{fn: value, args: []rideType{rideInt(12345)}},
		{fn: valueOrErrorMessage, args: []rideType{rideInt(12345), rideString("error message")}},
		{fn: valueOrElse, args: []rideType{rideUnit{}, rideInt(12345)}},
		{fn: sizeTuple, args: []rideType{tuple}},
		{fn: unit},
		{fn: unaryNot, args: []rideType{rideBoolean(false)}},
		{fn: booleanToBytes, args: []rideType{rideBoolean(true)}},
		{fn: booleanToString, args: []rideType{rideBoolean(true)}},
	}
}

func nativeBenchIntCases(a nativeBenchArgs) []nativeBenchCase {
	return []nativeBenchCase{
		{fn: sum, args: []rideType{rideInt(1_234_567), rideInt(7_654_321)}},
		{fn: sub, args: []rideType{rideInt(1_234_567), rideInt(7_654_321)}},
		{fn: mul, args: []rideType{rideInt(1_234_567), rideInt(7_654_321)}},
		{fn: div, args: []rideType{rideInt(7_654_321), rideInt(1_234)}},
		{fn: mod, args: []rideType{rideInt(-7_654_321), rideInt(1_234)}},
		{fn: gt, args: []rideType{rideInt(7_654_321), rideInt(1_234_567)}},
		{fn: ge, args: []rideType{rideInt(7_654_321), rideInt(1_234_567)}},
		{fn: unaryMinus, args: []rideType{rideInt(1_234_567)}},
		{fn: intToString, variant: "small", args: []rideType{rideInt(7)}},
		{fn: intToString, variant: "large", args: []rideType{rideInt(-9_223_372_036_854_775_807)}},
		{fn: intToBytes, args: []rideType{rideInt(1_234_567)}},
		{fn: fraction, variant: "small", args: []rideType{rideInt(1_234), rideInt(5_678), rideInt(91)}},
		{fn: fraction, variant: "large", args: []rideType{
			rideInt(9_000_000_000_000_000_000), rideInt(8_000_000_000_000_000_000), rideInt(9_100_000_000_000_000_000),
		}},
		{fn: fractionIntRounds, args: []rideType{rideInt(1_234_567), rideInt(5_678), rideInt(91), a.halfUp}},
		{fn: pow, variant: "small", args: []rideType{rideInt(12), rideInt(1), rideInt(3456), rideInt(3), rideInt(2),
			a.down}},
		{fn: pow, variant: "large", args: []rideType{rideInt(1_000_000_000_000), rideInt(8), rideInt(50_000_000),
			rideInt(8), rideInt(8), a.halfUp}},
		{fn: log, args: []rideType{rideInt(1_000_000_000_000), rideInt(8), rideInt(2), rideInt(0), rideInt(8),
			a.halfUp}},
		{fn: sqrt, args: []rideType{rideInt(1_000_000_000_000), rideInt(8), rideInt(8), a.halfUp}},
	}
}

const (
	nativeBenchBigIntSmall = "123456789"
	nativeBenchBigIntLarge = "1606938044258990275541962092341162602522202993782792835301376" // 2^200
	nativeBenchBigIntMax   = "6703903964971298549787012499102923063739682910296196688861780721860882015036773488" +
		"400937149083451713845015929093243025426876941405973284973216824503042047"
)

func nativeBenchBigIntCases(a nativeBenchArgs) []nativeBenchCase {
	small := bigIntValue(nativeBenchBigIntSmall)
	large := bigIntValue(nativeBenchBigIntLarge)
	maxBigInt := bigIntValue(nativeBenchBigIntMax)
	maxBytes := rideByteVector(append([]byte{0x7f}, bytes.Repeat([]byte{0xff}, 63)...))
	return []nativeBenchCase{
		{fn: toBigInt, args: []rideType{rideInt(1_234_567)}},
		{fn: sumBigInt, variant: "small", args: []rideType{small, small}},
		{fn: sumBigInt, variant: "large", args: []rideType{large, large}},
		{fn: subtractBigInt, variant: "small", args: []rideType{small, small}},
		{fn: subtractBigInt, variant: "large", args: []rideType{maxBigInt, large}},
		{fn: multiplyBigInt, variant: "small", args: []rideType{small, small}},
		{fn: multiplyBigInt, variant: "large", args: []rideType{large, large}},
		{fn: divideBigInt, variant: "small", args: []rideType{small, rideBigInt{v: big.NewInt(7)}}},
		{fn: divideBigInt, variant: "large", args: []rideType{maxBigInt, large}},
		{fn: moduloBigInt, variant: "small", args: []rideType{small, rideBigInt{v: big.NewInt(7)}}},
		{fn: moduloBigInt, variant: "large", args: []rideType{maxBigInt, small}},
		{fn: fractionBigInt, variant: "small", args: []rideType{small, small, rideBigInt{v: big.NewInt(7)}}},
		{fn: fractionBigInt, variant: "large", args: []rideType{maxBigInt, large, large}},
		{fn: fractionBigIntRounds, variant: "small", args: []rideType{small, small, rideBigInt{v: big.NewInt(7)},
			a.halfUp}},
		{fn: fractionBigIntRounds, variant: "large", args: []rideType{maxBigInt, large, large, a.halfUp}},
		{fn: unaryMinusBigInt, args: []rideType{large}},
		{fn: gtBigInt, args: []rideType{large, small}},
		{fn: geBigInt, args: []rideType{large, small}},
		{fn: maxListBigInt, variant: "small", args: []rideType{bigIntList(10)}},
		{fn: maxListBigInt, variant: "large", args: []rideType{bigIntList(maxListSize)}},
		{fn: minListBigInt, variant: "small", args: []rideType{bigIntList(10)}},
		{fn: minListBigInt, variant: "large", args: []rideType{bigIntList(maxListSize)}},
		{fn: medianListBigInt, variant: "small", args: []rideType{bigIntList(10)}},
		{fn: medianListBigInt, variant: "large", args: []rideType{bigIntList(maxListSize)}},
		{fn: bigIntToBytes, variant: "small", args: []rideType{small}},
		{fn: bigIntToBytes, variant: "large", args: []rideType{maxBigInt}},
		{fn: bytesToBigInt, variant: "small", args: []rideType{rideByteVector{0x07, 0x5b, 0xcd, 0x15}}},
		{fn: bytesToBigInt, variant: "large", args: []rideType{maxBytes}},
		{fn: bytesToBigIntLim, args: []rideType{maxBytes, rideInt(16), rideInt(32)}},
		{fn: bigIntToInt, args: []rideType{small}},
		{fn: bigIntToString, variant: "small", args: []rideType{small}},
		{fn: bigIntToString, variant: "large", args: []rideType{maxBigInt}},
		{fn: stringToBigInt, variant: "small", args: []rideType{rideString(nativeBenchBigIntSmall)}},
		{fn: stringToBigInt, variant: "large", args: []rideType{rideString(nativeBenchBigIntMax)}},
		{fn: stringToBigIntOpt, args: []rideType{rideString(nativeBenchBigIntLarge)}},
		{fn: powBigInt, args: []rideType{bigIntValue("1000000000000"), rideInt(8), bigIntValue("50000000"),
			rideInt(8), rideInt(8), a.halfUp}},
		{fn: logBigInt, args: []rideType{bigIntValue("1000000000000"), rideInt(8), bigIntValue("2"), rideInt(0),
			rideInt(8), a.halfUp}},
		{fn: sqrtBigInt, variant: "small", args: []rideType{small, rideInt(0), rideInt(8), a.halfUp}},
		{fn: sqrtBigInt, variant: "large", args: []rideType{large, rideInt(0), rideInt(18), a.halfUp}},
	}
}

const (
	nativeBenchASCII = "The quick brown fox jumps over the lazy dog. "
	nativeBenchUTF8  = "Съешь же ещё этих мягких французских булок, да выпей чаю. ✓ "
)

func nativeBenchStringCases() []nativeBenchCase {
	small := rideString("native, functions, benchmark")
	large := rideString(strings.Repeat(nativeBenchASCII, 500))                // 22500 bytes.
	utf8 := rideString(strings.Repeat(nativeBenchUTF8, 150))                  // ~16 KB, 9150 runes.
	csv := rideString(strings.TrimSuffix(strings.Repeat("item,", 1000), ",")) // 1000 items.
	return []nativeBenchCase{
		{fn: concatStrings, variant: "small", args: []rideType{small, small}},
		{fn: concatStrings, variant: "large", args: []rideType{large, small}},
		{fn: sizeString, variant: "ascii", args: []rideType{large}},
		{fn: sizeString, variant: "utf8", args: []rideType{utf8}},
		{fn: takeString, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: takeString, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: takeStringV6, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: takeStringV6, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: dropString, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: dropString, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: dropStringV6, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: dropStringV6, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: takeRightString, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: takeRightString, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: takeRightStringV6, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: takeRightStringV6, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: dropRightString, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: dropRightString, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: dropRightStringV6, variant: "ascii", args: []rideType{large, rideInt(10_000)}},
		{fn: dropRightStringV6, variant: "utf8", args: []rideType{utf8, rideInt(5_000)}},
		{fn: indexOfSubstring, variant: "ascii", args: []rideType{large, rideString("lazy cat")}},
		{fn: indexOfSubstring, variant: "utf8", args: []rideType{utf8, rideString("✓ ✓")}},
		{fn: indexOfSubstringWithOffset, variant: "ascii", args: []rideType{large, rideString("dog"), rideInt(10_000)}},
		{fn: indexOfSubstringWithOffset, variant: "utf8", args: []rideType{utf8, rideString("чаю"), rideInt(5_000)}},
		{fn: lastIndexOfSubstring, variant: "ascii", args: []rideType{large, rideString("quick")}},
		{fn: lastIndexOfSubstring, variant: "utf8", args: []rideType{utf8, rideString("булок")}},
		{fn: lastIndexOfSubstringWithOffset, variant: "ascii", args: []rideType{large, rideString("quick"),
			rideInt(10_000)}},
		{fn: lastIndexOfSubstringWithOffset, variant: "utf8", args: []rideType{utf8, rideString("булок"),
			rideInt(5_000)}},
		{fn: contains, variant: "ascii", args: []rideType{large, rideString("lazy cat")}},
		{fn: contains, variant: "utf8", args: []rideType{utf8, rideString("✓ ✓")}},
		{fn: replaceFirst, args: []rideType{large, rideString("dog"), rideString("cat")}},
		{fn: replaceAll, args: []rideType{large, rideString("dog"), rideString("cat")}},
		{fn: stringToBytes, variant: "small", args: []rideType{small}},
		{fn: stringToBytes, variant: "large", args: []rideType{large}},
		{fn: parseInt, args: []rideType{rideString("-9223372036854775807")}},
		{fn: parseIntValue, args: []rideType{rideString("1234567890")}},
		{fn: splitString, variant: "small", args: []rideType{small, rideString(", ")}},
		{fn: splitString, variant: "large", args: []rideType{csv, rideString(",")}},
		{fn: splitStringV6, variant: "small", args: []rideType{small, rideString(", ")}},
		{fn: splitStringV6, variant: "large", args: []rideType{
			rideString(strings.TrimSuffix(strings.Repeat("item,", 20), ",")), rideString(","),
		}},
		{fn: splitString4C, variant: "small", args: []rideType{small, rideString(", ")}},
		{fn: splitString4C, variant: "large", args: []rideType{
			rideString(strings.TrimSuffix(strings.Repeat("item,", 100), ",")), rideString(","),
		}},
		{fn: splitString51C, variant: "small", args: []rideType{small, rideString(", ")}},
		{fn: splitString51C, variant: "large", args: []rideType{csv, rideString(",")}},
		{fn: makeString, variant: "small", args: []rideType{stringList(5, "item"), rideString(", ")}},
		{fn: makeString, variant: "large", args: []rideType{stringList(maxListSize, "item"), rideString(", ")}},
		{fn: makeStringV6, variant: "small", args: []rideType{stringList(5, "item"), rideString(", ")}},
		{fn: makeStringV6, variant: "large", args: []rideType{stringList(70, "i"), rideString(",")}},
		{fn: makeString2C, variant: "small", args: []rideType{stringList(5, "item"), rideString(", ")}},
		{fn: makeString2C, variant: "large", args: []rideType{stringList(100, "item"), rideString(", ")}},
		{fn: makeString11C, variant: "small", args: []rideType{stringList(5, "item"), rideString(", ")}},
		{fn: makeString11C, variant: "large", args: []rideType{stringList(maxListSize, "item"), rideString(", ")}},
	}
}

func nativeBenchBytesCases() []nativeBenchCase {
	small := repeatedBytes(32)
	large := repeatedBytes(16 * 1024)
	oneKB := repeatedBytes(1024)
	b58 := rideString(base58.Encode(repeatedBytes(64))) // 87 characters.
	b64Small := rideString(base64.StdEncoding.EncodeToString(small))
	b64Large := rideString(base64.StdEncoding.EncodeToString(large))
	b64OneKB := rideString(base64.StdEncoding.EncodeToString(oneKB))
	b16Small := rideString(hex.EncodeToString(small))
	b16Large := rideString(hex.EncodeToString(large))
	b16OneKB := rideString(hex.EncodeToString(oneKB))
	utf8 := rideByteVector(strings.Repeat(nativeBenchUTF8, 150))
	return []nativeBenchCase{
		{fn: sizeBytes, args: []rideType{large}},
		{fn: takeBytes, args: []rideType{large, rideInt(10_000)}},
		{fn: takeBytesV6, args: []rideType{large, rideInt(10_000)}},
		{fn: dropBytes, args: []rideType{large, rideInt(10_000)}},
		{fn: dropBytesV6, args: []rideType{large, rideInt(10_000)}},
		{fn: takeRightBytes, args: []rideType{large, rideInt(10_000)}},
		{fn: takeRightBytesV6, args: []rideType{large, rideInt(10_000)}},
		{fn: dropRightBytes, args: []rideType{large, rideInt(10_000)}},
		{fn: dropRightBytesV6, args: []rideType{large, rideInt(10_000)}},
		{fn: concatBytes, variant: "small", args: []rideType{small, small}},
		{fn: concatBytes, variant: "large", args: []rideType{large, large}},
		{fn: toBase58, variant: "small", args: []rideType{small}},
		{fn: toBase58, variant: "large", args: []rideType{repeatedBytes(64)}},
		{fn: toBase58V4, variant: "small", args: []rideType{small}},
		{fn: toBase58V4, variant: "large", args: []rideType{repeatedBytes(64)}},
		{fn: fromBase58, variant: "small", args: []rideType{rideString(base58.Encode(small))}},
		{fn: fromBase58, variant: "large", args: []rideType{b58}},
		{fn: toBase64, variant: "small", args: []rideType{small}},
		{fn: toBase64, variant: "large", args: []rideType{large}},
		{fn: toBase64V4, variant: "small", args: []rideType{small}},
		{fn: toBase64V4, variant: "large", args: []rideType{large}},
		{fn: toBase641C, variant: "small", args: []rideType{small}},
		{fn: toBase641C, variant: "large", args: []rideType{oneKB}},
		{fn: fromBase64, variant: "small", args: []rideType{b64Small}},
		{fn: fromBase64, variant: "large", args: []rideType{b64Large}},
		{fn: fromBase641C, variant: "small", args: []rideType{b64Small}},
		{fn: fromBase641C, variant: "large", args: []rideType{b64OneKB}},
		{fn: toBase16, variant: "small", args: []rideType{small}},
		{fn: toBase16, variant: "large", args: []rideType{repeatedBytes(8 * 1024)}},
		{fn: toBase16V4, variant: "small", args: []rideType{small}},
		{fn: toBase16V4, variant: "large", args: []rideType{repeatedBytes(8 * 1024)}},
		{fn: toBase161C, variant: "small", args: []rideType{small}},
		{fn: toBase161C, variant: "large", args: []rideType{oneKB}},
		{fn: fromBase16, variant: "small", args: []rideType{b16Small}},
		{fn: fromBase16, variant: "large", args: []rideType{b16Large}},
		{fn: fromBase16V4, variant: "small", args: []rideType{b16Small}},
		{fn: fromBase16V4, variant: "large", args: []rideType{b16Large}},
		{fn: fromBase161C, variant: "small", args: []rideType{b16Small}},
		{fn: fromBase161C, variant: "large", args: []rideType{b16OneKB}},
		{fn: bytesToUTF8String, variant: "small", args: []rideType{rideByteVector("native functions")}},
		{fn: bytesToUTF8String, variant: "large", args: []rideType{utf8}},
		{fn: bytesToUTF8StringV4, variant: "small", args: []rideType{rideByteVector("native functions")}},
		{fn: bytesToUTF8StringV4, variant: "large", args: []rideType{utf8}},
		{fn: bytesToInt, args: []rideType{small}},
		{fn: bytesToIntWithOffset, args: []rideType{small, rideInt(16)}},
	}
}

func nativeBenchListCases(tb testing.TB) []nativeBenchCase {
	tb.Helper()
	small := intList(10)
	large := intList(maxListSize)
	half := intList(maxListSize / 2)
	last := large[len(large)-1]
	entries := make(rideList, 0, 100)
	for i := range 25 {
		k := rideString(nativeBenchKey + strconv.Itoa(i))
		entries = append(entries,
			mustCallRideFunction(tb, integerEntryConstructor, k, rideInt(i)),
			mustCallRideFunction(tb, booleanEntryConstructor, k, rideBoolean(i%2 == 0)),
			mustCallRideFunction(tb, binaryEntryConstructor, k, repeatedBytes(32)),
			mustCallRideFunction(tb, stringEntryConstructor, k, rideString("value"+strconv.Itoa(i))),
		)
	}
	lastKey := rideString(nativeBenchKey + "24")
	dataEntries := make(rideList, 0, 100)
	for i := range 100 {
		k := rideString(nativeBenchKey + strconv.Itoa(i))
		dataEntries = append(dataEntries, mustCallRideFunction(tb, dataEntryConstructor, k, rideInt(i)))
	}
	return []nativeBenchCase{
		{fn: sizeList, args: []rideType{large}},
		{fn: getList, args: []rideType{large, rideInt(500)}},
		{fn: createList, variant: "small", args: []rideType{rideInt(1), small}},
		{fn: createList, variant: "large", args: []rideType{rideInt(1), half}},
		{fn: limitedCreateList, variant: "small", args: []rideType{rideInt(1), small}},
		{fn: limitedCreateList, variant: "large", args: []rideType{rideInt(1), half}},
		{fn: appendToList, variant: "small", args: []rideType{small, rideInt(1)}},
		{fn: appendToList, variant: "large", args: []rideType{half, rideInt(1)}},
		{fn: concatList, variant: "small", args: []rideType{small, small}},
		{fn: concatList, variant: "large", args: []rideType{half, half}},
		{fn: indexOfList, variant: "small", args: []rideType{small, small[len(small)-1]}},
		{fn: indexOfList, variant: "large", args: []rideType{large, last}},
		{fn: lastIndexOfList, variant: "small", args: []rideType{small, small[0]}},
		{fn: lastIndexOfList, variant: "large", args: []rideType{large, large[0]}},
		{fn: containsElement, variant: "small", args: []rideType{small, small[len(small)-1]}},
		{fn: containsElement, variant: "large", args: []rideType{large, last}},
		{fn: listRemoveByIndex, variant: "small", args: []rideType{small, rideInt(5)}},
		{fn: listRemoveByIndex, variant: "large", args: []rideType{large, rideInt(500)}},
		{fn: listReplaceByIndex, variant: "small", args: []rideType{small, rideInt(5), rideInt(1)}},
		{fn: listReplaceByIndex, variant: "large", args: []rideType{large, rideInt(500), rideInt(1)}},
		{fn: fillList, variant: "small", args: []rideType{rideInt(10), rideInt(1)}},
		{fn: fillList, variant: "large", args: []rideType{rideInt(maxListSize), rideInt(1)}},
		{fn: median, variant: "small", args: []rideType{small}},
		{fn: median, variant: "large", args: []rideType{large}},
		{fn: listMax, variant: "small", args: []rideType{small}},
		{fn: listMax, variant: "large", args: []rideType{large}},
		{fn: listMin, variant: "small", args: []rideType{small}},
		{fn: listMin, variant: "large", args: []rideType{large}},
		{fn: intFromArray, variant: "entries", args: []rideType{entries, lastKey}},
		{fn: intFromArray, variant: "dataEntries", args: []rideType{dataEntries, rideString(nativeBenchKey + "99")}},
		{fn: booleanFromArray, args: []rideType{entries, lastKey}},
		{fn: bytesFromArray, args: []rideType{entries, lastKey}},
		{fn: stringFromArray, args: []rideType{entries, lastKey}},
		{fn: intValueFromArray, args: []rideType{entries, lastKey}},
		{fn: booleanValueFromArray, args: []rideType{entries, lastKey}},
		{fn: bytesValueFromArray, args: []rideType{entries, lastKey}},
		{fn: stringValueFromArray, args: []rideType{entries, lastKey}},
		{fn: intFromArrayByIndex, variant: "entries", args: []rideType{entries, rideInt(96)}},
		{fn: intFromArrayByIndex, variant: "dataEntries", args: []rideType{dataEntries, rideInt(99)}},
		{fn: booleanFromArrayByIndex, args: []rideType{entries, rideInt(97)}},
		{fn: bytesFromArrayByIndex, args: []rideType{entries, rideInt(98)}},
		{fn: stringFromArrayByIndex, args: []rideType{entries, rideInt(99)}},
		{fn: intValueFromArrayByIndex, args: []rideType{entries, rideInt(96)}},
		{fn: booleanValueFromArrayByIndex, args: []rideType{entries, rideInt(97)}},
		{fn: bytesValueFromArrayByIndex, args: []rideType{entries, rideInt(98)}},
		{fn: stringValueFromArrayByIndex, args: []rideType{entries, rideInt(99)}},
	}
}

func nativeBenchStateCases(tb testing.TB, f *nativeBenchFixture, a nativeBenchArgs) []nativeBenchCase {
	tb.Helper()
	key := rideString(nativeBenchKey)
	txID := rideByteVector(f.transfer.ID.Bytes())
	pb, err := f.transfer.MarshalSignedToProtobuf(nativeBenchScheme)
	require.NoError(tb, err)
	ethPK := mustDecodeHex(tb, "0c9af283046995d88527c7acc82dc7f7e5a29a3119d68b8903789541348e008f"+
		"4d0b8d7d8047c23818ec2063f6299ba469f79245d07d78f2b55f500f5d953e4f")
	cases := []nativeBenchCase{
		{fn: addressFromString, args: []rideType{rideString(f.accounts.dApp.String())}},
		{fn: addressValueFromString, args: []rideType{rideString(f.accounts.dApp.String())}},
		{fn: addressToString, args: []rideType{a.address}},
		{fn: addressFromPublicKey, args: []rideType{rideByteVector(f.accounts.senderPK.Bytes())}},
		{fn: addressFromPublicKeyStrict, variant: "waves", args: []rideType{rideByteVector(f.accounts.senderPK.Bytes())}},
		{fn: addressFromPublicKeyStrict, variant: "ethereum", args: []rideType{ethPK}},
		{fn: transactionByID, args: []rideType{txID}},
		{fn: transactionHeightByID, args: []rideType{txID}},
		{fn: transferByID, args: []rideType{txID}},
		{fn: transferFromProtobuf, args: []rideType{rideByteVector(pb)}},
		{fn: assetBalanceV3, variant: "waves", args: []rideType{a.address, rideUnit{}}},
		{fn: assetBalanceV3, variant: "asset", args: []rideType{a.address, a.asset}},
		{fn: assetBalanceV4, args: []rideType{a.address, a.asset}},
		{fn: assetInfoV3, args: []rideType{a.asset}},
		{fn: assetInfoV4, args: []rideType{a.asset}},
		{fn: blockInfoByHeight, args: []rideType{rideInt(nativeBenchHeight - 1)}},
		{fn: calculateDelay, args: []rideType{a.address, rideInt(nativeBenchBalance)}},
		{fn: intFromSelfState, args: []rideType{key}},
		{fn: booleanFromSelfState, args: []rideType{key}},
		{fn: bytesFromSelfState, args: []rideType{key}},
		{fn: stringFromSelfState, args: []rideType{key}},
		{fn: intValueFromSelfState, args: []rideType{key}},
		{fn: booleanValueFromSelfState, args: []rideType{key}},
		{fn: bytesValueFromSelfState, args: []rideType{key}},
		{fn: stringValueFromSelfState, args: []rideType{key}},
	}
	for _, r := range a.recipients {
		variant := r.instanceOf()
		cases = append(cases,
			nativeBenchCase{fn: addressFromRecipient, variant: variant, args: []rideType{r}},
			nativeBenchCase{fn: hashScriptAtAddress, variant: variant, args: []rideType{r}},
			nativeBenchCase{fn: isDataStorageUntouched, variant: variant, args: []rideType{r}},
			nativeBenchCase{fn: wavesBalanceV3, variant: variant, args: []rideType{r}},
			nativeBenchCase{fn: wavesBalanceV4, variant: variant, args: []rideType{r}},
			nativeBenchCase{fn: intFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: booleanFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: bytesFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: stringFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: intValueFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: booleanValueFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: bytesValueFromState, variant: variant, args: []rideType{r, key}},
			nativeBenchCase{fn: stringValueFromState, variant: variant, args: []rideType{r, key}},
		)
	}
	return cases
}

func nativeBenchHashCases() []nativeBenchCase {
	small := repeatedBytes(32)
	large := repeatedBytes(32 * 1024)
	cases := []nativeBenchCase{
		{fn: keccak256, variant: "small", args: []rideType{small}},
		{fn: keccak256, variant: "large", args: []rideType{large}},
		{fn: keccak256, variant: "string", args: []rideType{rideString(nativeBenchASCII)}},
		{fn: blake2b256, variant: "small", args: []rideType{small}},
		{fn: blake2b256, variant: "large", args: []rideType{large}},
		{fn: blake2b256, variant: "string", args: []rideType{rideString(nativeBenchASCII)}},
		{fn: sha256, variant: "small", args: []rideType{small}},
		{fn: sha256, variant: "large", args: []rideType{large}},
		{fn: sha256, variant: "string", args: []rideType{rideString(nativeBenchASCII)}},
	}
	families := []struct {
		size int
		fns  []rideFunction
	}{
		{size: 16, fns: []rideFunction{keccak256_16, blake2b256_16, sha256_16}},
		{size: 32, fns: []rideFunction{keccak256_32, blake2b256_32, sha256_32}},
		{size: 64, fns: []rideFunction{keccak256_64, blake2b256_64, sha256_64}},
		{size: 128, fns: []rideFunction{keccak256_128, blake2b256_128, sha256_128}},
	}
	for _, fm := range families {
		data := repeatedBytes(fm.size * 1024)
		for _, fn := range fm.fns {
			cases = append(cases, nativeBenchCase{fn: fn, args: []rideType{data}})
		}
	}
	return cases
}

func nativeBenchSignatureCases(tb testing.TB, f *nativeBenchFixture, a nativeBenchArgs) []nativeBenchCase {
	tb.Helper()
	pk := rideByteVector(f.accounts.senderPK.Bytes())
	sign := func(msg []byte) rideByteVector {
		sig, err := crypto.Sign(f.accounts.senderSK, msg)
		require.NoError(tb, err)
		return sig.Bytes()
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(tb, err)
	rsaPK, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	require.NoError(tb, err)
	rsaSign := func(msg []byte) rideByteVector {
		d := sh256.Sum256(msg)
		sig, sErr := rsa.SignPKCS1v15(rand.Reader, rsaKey, stdcrypto.SHA256, d[:])
		require.NoError(tb, sErr)
		return sig
	}
	small := repeatedBytes(32)
	large := repeatedBytes(32 * 1024)
	cases := []nativeBenchCase{
		{fn: sigVerify, variant: "small", args: []rideType{small, sign(small), pk}},
		{fn: sigVerify, variant: "large", args: []rideType{large, sign(large), pk}},
		{fn: rsaVerify, variant: "small", args: []rideType{a.sha256Alg, small, rsaSign(small), rideByteVector(rsaPK)}},
		{fn: rsaVerify, variant: "large", args: []rideType{a.sha256Alg, large, rsaSign(large), rideByteVector(rsaPK)}},
	}
	for _, s := range []struct {
		size int
		fn   rideFunction
	}{
		{size: 8, fn: sigVerify_8}, {size: 16, fn: sigVerify_16}, {size: 32, fn: sigVerify_32},
		{size: 64, fn: sigVerify_64}, {size: 128, fn: sigVerify_128},
	} {
		msg := repeatedBytes(s.size * 1024)
		cases = append(cases, nativeBenchCase{fn: s.fn, args: []rideType{msg, sign(msg), pk}})
	}
	for _, s := range []struct {
		size int
		fn   rideFunction
	}{
		{size: 16, fn: rsaVerify_16}, {size: 32, fn: rsaVerify_32}, {size: 64, fn: rsaVerify_64},
		{size: 128, fn: rsaVerify_128},
	} {
		msg := repeatedBytes(s.size * 1024)
		cases = append(cases, nativeBenchCase{fn: s.fn,
			args: []rideType{a.sha256Alg, msg, rsaSign(msg), rideByteVector(rsaPK)}})
	}
	return cases
}

const (
	nativeBenchBLSKey = "hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5" +
		"Lv7KN2tLKnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRs" +
		"PnEvKvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pG" +
		"i8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8Wwa" +
		"XsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W4WvRyZ" +
		"hmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r"
	nativeBenchBLSProof = "lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+W" +
		"vb7GlYlHpxCysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZV" +
		"mh1rlyqznIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF"
	nativeBenchBLSInputs = "LcMT3OOlkHLzJBKCKjjzzVMg+r+FVgd52LlhZPB4RFg="
	nativeBenchBNKey     = "LDCJzjgi5HtcHEXHfU8TZz+ZUHD2ZwsQ7JIEvzdMPYKYs9SoGkKUmg1yya4TE0Ms7x+KOJ4Ze/CPfKp2s5jbniF" +
		"NM71N/YlHVbNkytLtQi1DzReSh9SNBsvskdY5mavQJe+67PuPVEYnx+lJ97qIG8243njZbGWPqUJ2Vqj49NAunhqX+eIkK3zAB3IPWls" +
		"3gruzX2t9wrmyE9cVVvf1kgWx63PsQV37qdH0KcFRpCH89k4TPS6fLmqdFxX3YGHCGFTpr6tLogvjbUFJPT98kJ/xck0C0B/s8PTVKd" +
		"ao4VQHT4DBIO8+GB3CQVh6VV4EcMLtDWWNxF4yloAlKcFT0Q4AzJSimpFqd/SwSz9Pb7uk5srte3nwphVamC+fHlJt"
	nativeBenchBNProof = "GQPBoHuCPcIosF+WZKE5jZV13Ib4EdjLnABncpSHcMKBZl0LhllnPxcuzExIQwhxcfXvFFAjlnDGpKauQ9OQsjB" +
		"KUBsdBZnGiV2Sg4TSdyHuLo2AbRRqJN0IV3iH3On8I4ngnL30ZAxVyGQH2EK58aUZGxMbbXGR9pQdh99QaiE="
	nativeBenchBNInputs = "IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQo="
)

func nativeBenchCryptoCases(tb testing.TB) []nativeBenchCase {
	tb.Helper()
	blsArgs := []rideType{
		mustDecodeBase64(tb, nativeBenchBLSKey),
		mustDecodeBase64(tb, nativeBenchBLSProof),
		mustDecodeBase64(tb, nativeBenchBLSInputs),
	}
	bnArgs := []rideType{
		mustDecodeBase64(tb, nativeBenchBNKey),
		mustDecodeBase64(tb, nativeBenchBNProof),
		mustDecodeBase64(tb, nativeBenchBNInputs),
	}
	merkleProofs := make(rideList, 0, 10)
	for _, p := range []string{
		"D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu", "DqspFkHCwkUdN8FsHkzVEGtfzhycFPgNNyi7YeMQunpR",
		"9YapWwCMpJaytFUaSnBwGpHsuGuixtnChpPyzSZeQCC7", "9CorA9cjXNdDQ3dxMk5aL4myMBELVdX1FH5RrJ6RTtG8",
		"J1ZLoKt7wsX2oCXtWYrtaCxKJZyL1ZyZXYgVBXPhXtKh", "Fm8onvGicJFTfPcBgRXMHY863HhPHHi3huHKCoBeyBFC",
		"9jvzHEcg5NTgXAxyxtbSS3Qq9Zp84gcZ5WJTJWSZeGNr", "32XGrpXv46NtBcHjaygGdwn1KqHqen3oNJSmRCAt7waN",
		"2FM86QERU97ewCicP3NiYPKEDYe7jrriHFn9NSKgo3mE", "6ze4HCcxj7gpjzAuE9Tco3nLU186mC6FAUZFbyuSVjaj",
	} {
		merkleProofs = append(merkleProofs, mustDecodeBase58(tb, p))
	}
	merkleLeaf := mustDecodeBase58(tb, "DcasUHxyPk3bYLZs5h17SjZJAP4uUEzjkycboi4YAXGD")
	chain := loadCertificates(tb, "../crypto/testdata/tdx-cert-chain.pem")
	cases := []nativeBenchCase{
		{fn: bls12Groth16Verify, args: blsArgs},
		{fn: bn256Groth16Verify, args: bnArgs},
		{fn: checkMerkleProof, args: []rideType{
			mustDecodeBase64(tb, "eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk="),
			mustDecodeBase64(tb, "ACCP8jyg8Rv62mE4IMD4FGATnUXEIoCIK0LMoQCjAGpl5AEg16lhBiAz+xB8hwUs8U7dTJeGmJQyWVfXm"+
				"HqzA+b2YuUBICJEors9RDiMZNeWp2yIlJrpf/a4rZxTvI7yIx3D5pihACAaVrwYIveDbOb3uE+Hj1w+Tl0vornHqPT9pCja/TmfPg"+
				"AgxGoHWeIYY3RDkfAyYD99LA6OXdiXaB9a86EifTMS728AINbkCaDKCXEc5i61+c3ewBPFoCCYMCyvIrDbmHAThKt4ACAUkUex5yc"+
				"LaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw=="),
			mustDecodeBase64(tb, "AAAdIQ=="),
		}},
		{fn: rebuildMerkleRoot, args: []rideType{merkleProofs, merkleLeaf, rideInt(0)}},
		{fn: ecRecover, args: []rideType{
			mustDecodeHex(tb, "da74793f1299abeb213430596f281261355e29af0fdf5d359fe23cd9aca824c8"),
			mustDecodeHex(tb, "a57deea68952929239bd764d1f6966ea982af65fa6305f3bb71819a0376bd0ff42887b4496780434bd"+
				"954af05f2b24ab54f10d63ba11e3ce0a2c73c6e25a77cd1c"),
		}},
		{fn: secP256Verify, args: []rideType{
			mustDecodeBase64(tb,
				"AgIZGwP/AAYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFQAAAAAAAADnAAAAAAAAALeumraedvd5"+
					"Slaw2xkVKB1DXUiMkdQG7TOnk5yvhzD4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADcnip8b5SPF0dONKf8Q+0DD"+
					"3wVY/G6vd9jQMguDlSoxQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"+
					"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIABwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"+
					"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADQ0PMxCMjQs6H9Ericuy2oMAj6fPa7h5C7H86EurUIgwAAAAAAAAAAAAAAAAA"+
					"AAAAAAAAAAAAAAAAAAAAAAAAA"),
			mustDecodeBase64(tb, "90ctulEo2RFhfKMLLgT9WHnx+TnmytOCWNSNwEWsVTjkEhNEMU0lyOtP2XESdwTFUAlRJwryIkWjYZR53"+
				"H4FyQ=="),
			mustDecodeBase64(tb, "KdU/1vG5aM0TC1WRHYmV8ByD6oabSRj7vHVvqIWYn0h60Ihc/FT/NvVgBTMG8rnVnEF+AeojruMo22LjhGDo7A=="),
		}},
		{fn: validateCertChain, args: []rideType{
			toRideByteVectorList(tb, chain), toRideByteVectorList(tb, loadRevocations(tb)), rideInt(1770106534615),
		}},
	}
	for _, fn := range []rideFunction{
		bls12Groth16Verify_1, bls12Groth16Verify_2, bls12Groth16Verify_3, bls12Groth16Verify_4,
		bls12Groth16Verify_5, bls12Groth16Verify_6, bls12Groth16Verify_7, bls12Groth16Verify_8,
		bls12Groth16Verify_9, bls12Groth16Verify_10, bls12Groth16Verify_11, bls12Groth16Verify_12,
		bls12Groth16Verify_13, bls12Groth16Verify_14, bls12Groth16Verify_15,
	} {
		cases = append(cases, nativeBenchCase{fn: fn, args: blsArgs})
	}
	for _, fn := range []rideFunction{
		bn256Groth16Verify_1, bn256Groth16Verify_2, bn256Groth16Verify_3, bn256Groth16Verify_4,
		bn256Groth16Verify_5, bn256Groth16Verify_6, bn256Groth16Verify_7, bn256Groth16Verify_8,
		bn256Groth16Verify_9, bn256Groth16Verify_10, bn256Groth16Verify_11, bn256Groth16Verify_12,
		bn256Groth16Verify_13, bn256Groth16Verify_14, bn256Groth16Verify_15,
	} {
		cases = append(cases, nativeBenchCase{fn: fn, args: bnArgs})
	}
	return cases
}

func nativeBenchConstructorCases(tb testing.TB, a nativeBenchArgs) []nativeBenchCase {
	tb.Helper()
	key := rideString(nativeBenchKey)
	issueArgs := []rideType{rideString("Benchmark"), rideString("Native functions benchmark asset"),
		rideInt(1_000_000_000_000), rideInt(8), rideBoolean(true), rideUnit{}, rideInt(0)}
	issue := mustCallRideFunction(tb, issueConstructor, issueArgs...)
	lease := mustCallRideFunction(tb, leaseConstructor, a.address, rideInt(1_000_000), rideInt(0))
	transfer := mustCallRideFunction(tb, scriptTransferConstructor, a.address, rideInt(1_000_000), a.asset)
	dataEntry := mustCallRideFunction(tb, dataEntryConstructor, key, rideInt(1))
	writeSet := mustCallRideFunction(tb, writeSetConstructor, rideList{dataEntry, dataEntry})
	transferSet := mustCallRideFunction(tb, transferSetConstructor, rideList{transfer, transfer})
	cases := []nativeBenchCase{
		{fn: address, args: []rideType{rideByteVector(proto.WavesAddress(a.address).Bytes())}},
		{fn: alias, args: []rideType{rideString(nativeBenchAlias)}},
		{fn: assetPairConstructor, args: []rideType{a.asset, rideUnit{}}},
		{fn: attachedPaymentConstructor, args: []rideType{a.asset, rideInt(1_000_000)}},
		{fn: balanceDetailsConstructor, args: []rideType{rideInt(1), rideInt(2), rideInt(3), rideInt(4)}},
		{fn: integerEntryConstructor, args: []rideType{key, rideInt(1)}},
		{fn: booleanEntryConstructor, args: []rideType{key, rideBoolean(true)}},
		{fn: binaryEntryConstructor, args: []rideType{key, a.asset}},
		{fn: stringEntryConstructor, args: []rideType{key, rideString("value")}},
		{fn: deleteEntryConstructor, args: []rideType{key}},
		{fn: dataEntryConstructor, args: []rideType{key, rideInt(1)}},
		{fn: issueConstructor, args: issueArgs},
		{fn: simplifiedIssue, args: issueArgs[:5]},
		{fn: fullIssue, args: issueArgs},
		{fn: calculateAssetID, args: []rideType{issue}},
		{fn: reissueConstructor, args: []rideType{a.asset, rideInt(1_000), rideBoolean(true)}},
		{fn: burnConstructor, args: []rideType{a.asset, rideInt(1_000)}},
		{fn: sponsorFeeConstructor, args: []rideType{a.asset, rideInt(1_000)}},
		{fn: leaseConstructor, args: []rideType{a.address, rideInt(1_000_000), rideInt(0)}},
		{fn: simplifiedLease, args: []rideType{a.address, rideInt(1_000_000)}},
		{fn: fullLease, args: []rideType{a.address, rideInt(1_000_000), rideInt(0)}},
		{fn: calculateLeaseID, variant: "address", args: []rideType{lease}},
		{fn: calculateLeaseID, variant: "alias",
			args: []rideType{mustCallRideFunction(tb, leaseConstructor, a.alias, rideInt(1_000_000), rideInt(0))}},
		{fn: leaseCancelConstructor, args: []rideType{a.asset}},
		{fn: scriptTransferConstructor, args: []rideType{a.address, rideInt(1_000_000), a.asset}},
		{fn: transferConstructor, args: []rideType{a.address, rideInt(1_000_000)}},
		{fn: writeSetConstructor, args: []rideType{rideList{dataEntry, dataEntry}}},
		{fn: transferSetConstructor, args: []rideType{rideList{transfer, transfer}}},
		{fn: scriptResultConstructor, args: []rideType{writeSet, transferSet}},
	}
	for _, fn := range []rideFunction{
		createBuy, createSell, createCeiling, createDown, createFloor, createHalfDown, createHalfEven, createHalfUp,
		createUp, createNoAlg, createMd5, createSha1, createSha224, createSha256, createSha384, createSha512,
		createSha3224, createSha3256, createSha3384, createSha3512,
	} {
		cases = append(cases, nativeBenchCase{fn: fn})
	}
	tuples := []rideFunction{
		newTuple2, newTuple3, newTuple4, newTuple5, newTuple6, newTuple7, newTuple8, newTuple9, newTuple10,
		newTuple11, newTuple12, newTuple13, newTuple14, newTuple15, newTuple16, newTuple17, newTuple18, newTuple19,
		newTuple20, newTuple21, newTuple22,
	}
	for i, fn := range tuples {
		const minTupleSize = 2
		args := make([]rideType, i+minTupleSize)
		for j := range args {
			args[j] = rideInt(j)
		}
		cases = append(cases, nativeBenchCase{fn: fn, args: args})
	}
	return cases
}

// nativeBenchmarkResult keeps the results reachable, so the compiler can't drop the function calls.
var nativeBenchmarkResult rideType //nolint:gochecknoglobals // Benchmark sink.

func BenchmarkNativeFunctions(b *testing.B) {
	f := newNativeBenchFixture(b)
	for _, c := range nativeBenchCases(b, f) {
		b.Run(c.name(), func(b *testing.B) {
			_, err := c.fn(f.env, c.args...)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				nativeBenchmarkResult, _ = c.fn(f.env, c.args...)
			}
		})
	}
}

func TestNativeFunctionBenchmarkCases(t *testing.T) {
	f := newNativeBenchFixture(t)
	names := make(map[string]struct{})
	for _, c := range nativeBenchCases(t, f) {
		n := c.name()
		_, dup := names[n]
		require.False(t, dup, "duplicate benchmark case %q", n)
		names[n] = struct{}{}
		t.Run(n, func(t *testing.T) {
			r, err := c.fn(f.env, c.args...)
			require.NoError(t, err)
			require.NotNil(t, r)
			if fn := rideFunctionName(c.fn); strings.Contains(fn, "Verify") || fn == "checkMerkleProof" {
				// All verification functions must succeed with the prepared arguments.
				require.Equal(t, rideBoolean(true), r)
			}
			// Repeated call with the same arguments must give the same result, the arguments must stay intact.
			again, err := c.fn(f.env, c.args...)
			require.NoError(t, err)
			require.Equal(t, r, again)
		})
	}
}

// catalogueFunctions returns the Go identifiers of all distinct native functions of all library versions.
func catalogueFunctions(t *testing.T) map[string]struct{} {
	t.Helper()
	r := make(map[string]struct{})
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		functions, err := selectFunctions(v)
		require.NoError(t, err)
		for id := 0; ; id++ {
			fn := functions(id)
			if fn == nil {
				break
			}
			r[rideFunctionName(fn)] = struct{}{}
		}
	}
	return r
}

func TestNativeFunctionBenchmarkCoverage(t *testing.T) {
	catalogue := catalogueFunctions(t)
	require.NotEmpty(t, catalogue)
	benchmarked := make(map[string]struct{})
	for _, c := range nativeBenchCases(t, newNativeBenchFixture(t)) {
		benchmarked[rideFunctionName(c.fn)] = struct{}{}
	}
	for name := range catalogue {
		_, ok := benchmarked[name]
		reason, skipped := nativeBenchSkips[name]
		switch {
		case ok && skipped:
			t.Errorf("function %q has both a benchmark case and a skip entry", name)
		case !ok && !skipped:
			t.Errorf("function %q has neither a benchmark case nor a skip entry", name)
		case skipped && reason == "":
			t.Errorf("skip entry of function %q has no reason", name)
		}
	}
	for name := range benchmarked {
		if _, ok := catalogue[name]; !ok {
			t.Errorf("benchmarked function %q is not in the catalogues", name)
		}
	}
	for name := range nativeBenchSkips {
		if _, ok := catalogue[name]; !ok {
			t.Errorf("skipped function %q is not in the catalogues", name)
		}
	}
	require.Len(t, catalogue, len(benchmarked)+len(nativeBenchSkips), fmt.Sprintf("catalogue %d functions",
		len(catalogue)))
}
