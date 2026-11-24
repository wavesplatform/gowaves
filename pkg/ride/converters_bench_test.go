// Benchmarks of the conversions between the Ride objects and the Go representations of transactions,
// script actions and other blockchain entities.
//
//   - BenchmarkConvertToAction: conversion of a single Ride action object to a script action (convertToAction).
//   - BenchmarkObjectToActions: conversion of the V3 results WriteSet, TransferSet and ScriptResult.
//   - BenchmarkListToActions: conversion of the list of actions returned by V4+ callables.
//   - BenchmarkScriptActionToObject: conversion of a script action to a Ride transaction object.
//   - BenchmarkTransactionToObject: conversion of every transaction type to a Ride transaction object.
//   - BenchmarkObjectConverters: invocation, Ethereum invocation, block info, asset info, order and other objects.
//
// Every case of the switches in convertToAction, objectToActions, scriptActionToObject, transactionToObject and
// ethereumTransactionToObject must have a benchmark case, the coverage is checked by
// TestConversionBenchmarkCoverage.
//
// Run the benchmarks and save the results for comparison with benchstat:
//
//	go test -run '^$' \
//		-bench 'Benchmark(ConvertToAction|.*ToActions|ScriptActionToObject|TransactionToObject|ObjectConverters)$' \
//		-benchmem -count 10 ./pkg/ride/ > new.txt
//	go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
//
// Check the coverage of the switches and run every case once as a quick smoke test:
//
//	go test -run TestConversionBenchmarkCoverage \
//		-bench 'Benchmark(ConvertToAction|.*ToActions|ScriptActionToObject|TransactionToObject|ObjectConverters)$' \
//		-benchtime 1x ./pkg/ride/
//
// Collect CPU and allocation profiles and open the flame graphs in a browser (View -> Flame Graph):
//
//	go test -run '^$' -bench 'BenchmarkTransactionToObject' -benchmem -cpuprofile cpu.out -memprofile mem.out \
//		./pkg/ride/
//	go tool pprof -http=:8080 cpu.out
//	go tool pprof -http=:8081 -sample_index=alloc_space mem.out

package ride //nolint:testpackage // White-box benchmarks of the unexported converters.

import (
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	rideast "github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

const (
	convBenchMaxEntries   = 100 // Maximum number of data entries in WriteSet and in the list of actions.
	convBenchMaxTransfers = 10  // Maximum number of transfers in TransferSet.
	convBenchMassTxs      = 100 // Maximum number of transfers in MassTransfer transaction.
	convBenchTimestamp    = 1_700_000_000_000
	convBenchFee          = 500_000

	convBenchERC20HexData = "a9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c" +
		"00000000000000000000000000000000000000000000000000001cc92ad60000"
	convBenchInvokeHexData = "" +
		"3e08c2280000000000000000000000000000000000000000000000000000000000000040000000000000000000000000" +
		"000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000" +
		"000000057361666473000000000000000000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000"
	convBenchEthPK = "c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036d" +
		"b47fac027f7883108cbc7b2953539b9cb53"
)

// convBenchState extends the native functions benchmark state with the methods used to resolve the kind of
// Ethereum transactions.
type convBenchState struct {
	*nativeBenchState

	tree           *rideast.Tree
	assetConstInfo *proto.AssetConstInfo
}

func (s *convBenchState) NewestScriptByAccount(proto.Recipient) (*rideast.Tree, error) {
	return s.tree, nil
}

func (s *convBenchState) NewestAssetConstInfo(proto.AssetID) (*proto.AssetConstInfo, error) {
	return s.assetConstInfo, nil
}

// convBenchFixture holds the environments and the prepared values shared by the conversion benchmarks.
type convBenchFixture struct {
	native *nativeBenchFixture
	state  *convBenchState
	envs   map[rideast.LibraryVersion]*EvaluationEnvironment
}

func newConvBenchFixture(tb testing.TB) *convBenchFixture {
	tb.Helper()
	native := newNativeBenchFixture(tb)
	st := &convBenchState{
		nativeBenchState: native.state,
		tree: &rideast.Tree{Meta: meta.DApp{Functions: []meta.Function{
			{Name: "call", Arguments: []meta.Type{meta.String}},
		}}},
		assetConstInfo: &native.state.assetInfo.AssetConstInfo,
	}
	f := &convBenchFixture{native: native, state: st, envs: make(map[rideast.LibraryVersion]*EvaluationEnvironment)}
	for v := rideast.LibV1; v <= rideast.CurrentMaxLibraryVersion(); v++ {
		env, err := NewEnvironment(nativeBenchScheme, st, 0, 0, true, true, true, true, true)
		require.NoError(tb, err)
		env.setLibVersion(v)
		env.SetThisFromAddress(native.accounts.dApp)
		env.ChooseSizeCheck(v)
		env.SetTimestamp(convBenchTimestamp)
		env.ChooseTakeString(true)
		env.ChooseMaxDataEntriesSize(true)
		require.NoError(tb, env.SetTransaction(native.transfer))
		require.NoError(tb, env.SetLastBlockFromBlockInfo(native.state.blockInfo))
		f.envs[v] = env
	}
	return f
}

func (f *convBenchFixture) env() *EvaluationEnvironment {
	return f.envs[rideast.CurrentMaxLibraryVersion()]
}

// convBenchActionCase is a single Ride action object. The key is the instance name of the object.
type convBenchActionCase struct {
	variant string
	obj     rideType
}

func (c convBenchActionCase) name() string {
	if c.variant == "" {
		return c.obj.instanceOf()
	}
	return c.obj.instanceOf() + "/" + c.variant
}

func convBenchActionCases(f *convBenchFixture) []convBenchActionCase {
	a := f.native.accounts
	asset := rideByteVector(a.asset.Bytes())
	address := rideAddress(a.dApp)
	alias := rideAlias(*proto.NewAlias(nativeBenchScheme, nativeBenchAlias))
	value := repeatedBytes(32)
	return []convBenchActionCase{
		{obj: newRideBinaryEntry(nativeBenchKey, value)},
		{obj: newRideBooleanEntry(nativeBenchKey, true)},
		{obj: newRideDeleteEntry(nativeBenchKey)},
		{obj: newRideIntegerEntry(nativeBenchKey, 12345)},
		{obj: newRideStringEntry(nativeBenchKey, "conversion benchmark")},
		{variant: "integer", obj: newRideDataEntry(rideInt(12345), nativeBenchKey)},
		{variant: "boolean", obj: newRideDataEntry(rideBoolean(true), nativeBenchKey)},
		{variant: "string", obj: newRideDataEntry(rideString("conversion benchmark"), nativeBenchKey)},
		{variant: "binary", obj: newRideDataEntry(value, nativeBenchKey)},
		{obj: newRideIssue(rideUnit{}, "Benchmark", "Conversion benchmark asset", 1, 8, 1_000_000_000, true)},
		{obj: newRideReissue(asset, 1_000_000, true)},
		{obj: newRideBurn(asset, 1_000_000)},
		{obj: newRideSponsorFee(asset, 100_000)},
		{variant: "waves", obj: newRideScriptTransfer(rideUnit{}, address, 1_000_000)},
		{variant: "asset", obj: newRideScriptTransfer(asset, address, 1_000_000)},
		{variant: "alias", obj: newRideScriptTransfer(asset, alias, 1_000_000)},
		{variant: "address", obj: newRideLease(address, 1_000_000, 1)},
		{variant: "alias", obj: newRideLease(alias, 1_000_000, 1)},
		{obj: newRideLeaseCancel(repeatedBytes(crypto.DigestSize))},
	}
}

// convBenchDataEntries returns n V3 DataEntry objects of all value types.
func convBenchDataEntries(n int) rideList {
	r := make(rideList, n)
	for i := range r {
		key := rideString(nativeBenchKey + strconv.Itoa(i))
		switch i % 4 {
		case 0:
			r[i] = newRideDataEntry(rideInt(i), key)
		case 1:
			r[i] = newRideDataEntry(rideBoolean(i%3 == 0), key)
		case 2:
			r[i] = newRideDataEntry(rideString("value "+strconv.Itoa(i)), key)
		default:
			r[i] = newRideDataEntry(repeatedBytes(32), key)
		}
	}
	return r
}

// convBenchEntries returns n V4+ data entry actions of all types.
func convBenchEntries(n int) rideList {
	r := make(rideList, n)
	for i := range r {
		key := rideString(nativeBenchKey + strconv.Itoa(i))
		switch i % 5 {
		case 0:
			r[i] = newRideIntegerEntry(key, rideInt(i))
		case 1:
			r[i] = newRideBooleanEntry(key, i%3 == 0)
		case 2:
			r[i] = newRideStringEntry(key, rideString("value "+strconv.Itoa(i)))
		case 3:
			r[i] = newRideBinaryEntry(key, repeatedBytes(32))
		default:
			r[i] = newRideDeleteEntry(key)
		}
	}
	return r
}

func convBenchTransfers(f *convBenchFixture, n int) rideList {
	asset := rideByteVector(f.native.accounts.asset.Bytes())
	r := make(rideList, n)
	for i := range r {
		var a rideType = rideUnit{}
		if i%2 == 1 {
			a = asset
		}
		r[i] = newRideScriptTransfer(a, rideAddress(f.native.accounts.dApp), rideInt(1_000_000+i))
	}
	return r
}

// convBenchResultCase is a single V3 result object or the V4+ list of actions.
type convBenchResultCase struct {
	name string
	obj  rideType
}

func convBenchV3ResultCases(f *convBenchFixture) []convBenchResultCase {
	ws := func(n int) rideWriteSet { return newRideWriteSet(convBenchDataEntries(n)) }
	ts := func(n int) rideTransferSet { return newRideTransferSet(convBenchTransfers(f, n)) }
	return []convBenchResultCase{
		{name: writeSetTypeName + "/small", obj: ws(1)},
		{name: writeSetTypeName + "/large", obj: ws(convBenchMaxEntries)},
		{name: transferSetTypeName + "/small", obj: ts(1)},
		{name: transferSetTypeName + "/large", obj: ts(convBenchMaxTransfers)},
		{name: scriptResultTypeName + "/small", obj: newRideScriptResult(ws(1), ts(1))},
		{name: scriptResultTypeName + "/large", obj: newRideScriptResult(ws(convBenchMaxEntries),
			ts(convBenchMaxTransfers))},
	}
}

func convBenchListCases(f *convBenchFixture) []convBenchResultCase {
	small := rideList{
		newRideStringEntry(nativeBenchKey, "conversion benchmark"),
		newRideScriptTransfer(rideUnit{}, rideAddress(f.native.accounts.dApp), 1_000_000),
	}
	large := append(rideList{}, convBenchEntries(convBenchMaxEntries)...)
	large = append(large, convBenchTransfers(f, convBenchMaxTransfers)...)
	for _, c := range convBenchActionCases(f) {
		switch c.obj.(type) {
		case rideIssue, rideReissue, rideBurn, rideSponsorFee, rideLease, rideLeaseCancel:
			large = append(large, c.obj)
		}
	}
	return []convBenchResultCase{
		{name: "empty", obj: rideList{}},
		{name: "small", obj: small},
		{name: "large", obj: large},
	}
}

// convBenchScriptActionCase is a single script action. The key is the Go type of the action.
type convBenchScriptActionCase struct {
	action proto.ScriptAction
}

func (c convBenchScriptActionCase) name() string { return fmt.Sprintf("%T", c.action) }

func convBenchScriptActionCases(f *convBenchFixture) []convBenchScriptActionCase {
	asset := f.native.accounts.asset
	return []convBenchScriptActionCase{
		{action: &proto.ReissueScriptAction{AssetID: asset, Quantity: 1_000_000, Reissuable: true}},
		{action: &proto.BurnScriptAction{AssetID: asset, Quantity: 1_000_000}},
	}
}

// convBenchTxCase is a single transaction converted to the Ride object with the library version.
// The key is the Go type of the transaction.
type convBenchTxCase struct {
	variant string
	version rideast.LibraryVersion
	tx      proto.Transaction
}

func (c convBenchTxCase) key() string { return fmt.Sprintf("%T", c.tx) }

func (c convBenchTxCase) name() string {
	n := strings.TrimPrefix(c.key(), "*proto.")
	if c.variant != "" {
		n += "/" + c.variant
	}
	return n + "/V" + strconv.Itoa(int(c.version))
}

func convBenchSetScript(tb testing.TB, f *convBenchFixture) *proto.SetScriptWithProofs {
	tb.Helper()
	a := f.native.accounts
	tx := proto.NewUnsignedSetScriptWithProofs(2, a.senderPK, f.state.script, convBenchFee, convBenchTimestamp)
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchUpdateAssetInfo(tb testing.TB, f *convBenchFixture) *proto.UpdateAssetInfoWithProofs {
	tb.Helper()
	a := f.native.accounts
	tx := proto.NewUnsignedUpdateAssetInfoWithProofs(1, a.asset, a.senderPK, "Benchmark",
		"Conversion benchmark asset", convBenchTimestamp, proto.NewOptionalAssetWaves(), convBenchFee)
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchInvokeExpression(tb testing.TB, f *convBenchFixture) *proto.InvokeExpressionTransactionWithProofs {
	tb.Helper()
	a := f.native.accounts
	tx := proto.NewUnsignedInvokeExpressionWithProofs(1, a.senderPK, proto.B64Bytes(f.state.script),
		proto.NewOptionalAssetWaves(), convBenchFee, convBenchTimestamp)
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchCommitToGeneration(tb testing.TB, f *convBenchFixture) *proto.CommitToGenerationWithProofs {
	tb.Helper()
	const periodStart = 11
	a := f.native.accounts
	blsSK, err := bls.GenerateSecretKey([]byte("conversion benchmark"))
	require.NoError(tb, err)
	blsPK, err := blsSK.PublicKey()
	require.NoError(tb, err)
	_, sig, err := bls.ProvePoP(blsSK, blsPK, periodStart)
	require.NoError(tb, err)
	tx := proto.NewUnsignedCommitToGenerationWithProofs(1, a.senderPK, periodStart, blsPK, sig, convBenchFee,
		convBenchTimestamp)
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchData(tb testing.TB, f *convBenchFixture, n int) *proto.DataWithProofs {
	tb.Helper()
	a := f.native.accounts
	tx := proto.NewUnsignedDataWithProofs(2, a.senderPK, convBenchFee, convBenchTimestamp)
	for i := range n {
		key := nativeBenchKey + strconv.Itoa(i)
		var e proto.DataEntry
		switch i % 5 {
		case 0:
			e = &proto.IntegerDataEntry{Key: key, Value: int64(i)}
		case 1:
			e = &proto.BooleanDataEntry{Key: key, Value: i%3 == 0}
		case 2:
			e = &proto.StringDataEntry{Key: key, Value: "value " + strconv.Itoa(i)}
		case 3:
			e = &proto.BinaryDataEntry{Key: key, Value: repeatedBytes(32)}
		default:
			e = &proto.DeleteDataEntry{Key: key}
		}
		require.NoError(tb, tx.AppendEntry(e))
	}
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchMassTransfer(tb testing.TB, f *convBenchFixture, n int) *proto.MassTransferWithProofs {
	tb.Helper()
	a := f.native.accounts
	transfers := make([]proto.MassTransferEntry, n)
	for i := range transfers {
		transfers[i] = proto.MassTransferEntry{Recipient: proto.NewRecipientFromAddress(a.dApp), Amount: uint64(i + 1)}
	}
	tx := proto.NewUnsignedMassTransferWithProofs(2, a.senderPK, *proto.NewOptionalAssetFromDigest(a.asset),
		transfers, convBenchFee, convBenchTimestamp, proto.Attachment("conversion benchmark"))
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

func convBenchExchangeV4Orders(tb testing.TB, f *convBenchFixture) *proto.ExchangeWithProofs {
	tb.Helper()
	a := f.native.accounts
	asset := *proto.NewOptionalAssetFromDigest(a.asset)
	waves := proto.NewOptionalAssetWaves()
	order := func(t proto.OrderType) *proto.OrderV4 {
		o := proto.NewUnsignedOrderV4(a.senderPK, a.senderPK, asset, waves, t, 100_000_000, 1_000_000,
			convBenchTimestamp, convBenchTimestamp+86_400_000, 300_000, waves, proto.OrderPriceModeDefault,
			proto.Attachment("conversion benchmark"))
		require.NoError(tb, o.Sign(nativeBenchScheme, a.senderSK))
		return o
	}
	tx := proto.NewUnsignedExchangeWithProofs(3, order(proto.Buy), order(proto.Sell), 100_000_000, 1_000_000,
		300_000, 300_000, convBenchFee, convBenchTimestamp)
	require.NoError(tb, tx.Sign(nativeBenchScheme, a.senderSK))
	return tx
}

// convBenchEthereum returns the Ethereum transactions of all kinds.
func convBenchEthereum(tb testing.TB, f *convBenchFixture) []*proto.EthereumTransaction {
	tb.Helper()
	senderPK, err := proto.NewEthereumPublicKeyFromHexString(convBenchEthPK)
	require.NoError(tb, err)
	id := crypto.MustDigestFromBase58("GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9")
	assetTo := proto.EthereumAddress(f.native.accounts.asset[:proto.EthereumAddressSize])
	dAppTo := proto.EthereumAddress(f.native.accounts.dApp.Body())
	legacy := func(to *proto.EthereumAddress, value int64, data []byte) *proto.EthereumLegacyTx {
		return &proto.EthereumLegacyTx{
			Value: big.NewInt(value), To: to, Data: data, GasPrice: big.NewInt(1), Nonce: convBenchTimestamp,
			Gas: convBenchFee, V: big.NewInt(2*int64(nativeBenchScheme) + 35),
		}
	}
	erc20Data, err := hex.DecodeString(convBenchERC20HexData)
	require.NoError(tb, err)
	erc20CallData, err := ethabi.NewErc20MethodsMap().ParseCallDataRide(erc20Data, true)
	require.NoError(tb, err)
	erc20Args, err := ethabi.GetERC20TransferArguments(erc20CallData)
	require.NoError(tb, err)
	invokeData, err := hex.DecodeString(convBenchInvokeHexData)
	require.NoError(tb, err)
	mm, err := ethabi.NewMethodsMapFromRideDAppMeta(f.state.tree.Meta)
	require.NoError(tb, err)
	invokeCallData, err := mm.ParseCallDataRide(invokeData, true)
	require.NoError(tb, err)
	transferWaves := proto.NewEthereumTransaction(legacy(&dAppTo, 1_000_000_000_000_000, nil),
		proto.NewEthereumTransferWavesTxKind(), &id, &senderPK, 0)
	transferAsset := proto.NewEthereumTransaction(legacy(&assetTo, 0, erc20Data),
		proto.NewEthereumTransferAssetsErc20TxKind(*erc20CallData, *proto.NewOptionalAssetFromDigest(f.native.accounts.asset),
			erc20Args), &id, &senderPK, 0)
	invoke := proto.NewEthereumTransaction(legacy(&dAppTo, 0, invokeData),
		proto.NewEthereumInvokeScriptTxKind(*invokeCallData), &id, &senderPK, 0)
	return []*proto.EthereumTransaction{&transferWaves, &transferAsset, &invoke}
}

func ethereumKindVariant(tx *proto.EthereumTransaction) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", tx.TxKind), "*proto.")
}

func convBenchTxCases(tb testing.TB, f *convBenchFixture) []convBenchTxCase {
	tb.Helper()
	maxV := rideast.CurrentMaxLibraryVersion()
	cases := []convBenchTxCase{
		{tx: byte_helpers.Genesis.Transaction},
		{tx: byte_helpers.Payment.Transaction},
		{tx: byte_helpers.IssueWithSig.Transaction},
		{tx: byte_helpers.IssueWithProofs.Transaction},
		{tx: byte_helpers.TransferWithSig.Transaction},
		{tx: byte_helpers.TransferWithProofs.Transaction},
		{tx: byte_helpers.ReissueWithSig.Transaction},
		{tx: byte_helpers.ReissueWithProofs.Transaction},
		{tx: byte_helpers.BurnWithSig.Transaction},
		{tx: byte_helpers.BurnWithProofs.Transaction},
		{tx: byte_helpers.ExchangeWithSig.Transaction, version: rideast.LibV7},
		{tx: byte_helpers.ExchangeWithSig.Transaction},
		{tx: byte_helpers.ExchangeWithProofs.Transaction, version: rideast.LibV7},
		{tx: byte_helpers.ExchangeWithProofs.Transaction},
		{variant: "orderV4", tx: convBenchExchangeV4Orders(tb, f)},
		{tx: byte_helpers.LeaseWithSig.Transaction},
		{tx: byte_helpers.LeaseWithProofs.Transaction},
		{tx: byte_helpers.LeaseCancelWithSig.Transaction},
		{tx: byte_helpers.LeaseCancelWithProofs.Transaction},
		{tx: byte_helpers.CreateAliasWithSig.Transaction},
		{tx: byte_helpers.CreateAliasWithProofs.Transaction},
		{variant: "small", tx: byte_helpers.MassTransferWithProofs.Transaction},
		{variant: "large", tx: convBenchMassTransfer(tb, f, convBenchMassTxs)},
		{variant: "small", tx: byte_helpers.DataWithProofs.Transaction},
		{variant: "large", tx: convBenchData(tb, f, convBenchMaxEntries)},
		{tx: convBenchSetScript(tb, f)},
		{tx: byte_helpers.SponsorshipWithProofs.Transaction},
		{tx: byte_helpers.SetAssetScriptWithProofs.Transaction},
		{tx: byte_helpers.InvokeScriptWithProofs.Transaction, version: rideast.LibV3},
		{tx: byte_helpers.InvokeScriptWithProofs.Transaction},
		{tx: convBenchUpdateAssetInfo(tb, f)},
		{tx: convBenchInvokeExpression(tb, f)},
		{tx: convBenchCommitToGeneration(tb, f)},
	}
	for _, tx := range convBenchEthereum(tb, f) {
		cases = append(cases, convBenchTxCase{variant: ethereumKindVariant(tx), tx: tx})
	}
	for i := range cases {
		if cases[i].version == 0 {
			cases[i].version = maxV
		}
	}
	return cases
}

// convBenchObjectCase is a call of one of the other *ToObject converters.
type convBenchObjectCase struct {
	name string
	fn   func() (rideType, error)
}

func convBenchObjectCases(tb testing.TB, f *convBenchFixture) []convBenchObjectCase {
	tb.Helper()
	st := f.native.state
	invoke := byte_helpers.InvokeScriptWithProofs.Transaction
	expression := convBenchInvokeExpression(tb, f)
	ethInvoke := convBenchEthereum(tb, f)[2]
	payments := f.native.accounts.payments()
	exchangeV4 := convBenchExchangeV4Orders(tb, f)
	orderV1, err := byte_helpers.ExchangeWithProofs.Transaction.GetBuyOrder()
	require.NoError(tb, err)
	orderV4, err := exchangeV4.GetBuyOrder()
	require.NoError(tb, err)
	blockInfo := *st.blockInfo
	blockInfo.Rewards = proto.Rewards{
		proto.NewReward(f.native.accounts.sender, 600_000_000),
		proto.NewReward(f.native.accounts.dApp, 200_000_000),
	}
	transfer := &proto.FullScriptTransfer{
		Amount: 1_000_000, Asset: *proto.NewOptionalAssetFromDigest(f.native.accounts.asset),
		Recipient: proto.NewRecipientFromAddress(f.native.accounts.dApp), Sender: f.native.accounts.sender,
		SenderPK: f.native.accounts.senderPK, Timestamp: convBenchTimestamp, ID: &crypto.Digest{},
	}
	var cases []convBenchObjectCase
	for _, v := range []rideast.LibraryVersion{rideast.LibV3, rideast.LibV4, rideast.LibV5} {
		sv := "/V" + strconv.Itoa(int(v))
		cases = append(cases,
			convBenchObjectCase{name: "invocationToObject/InvokeScriptWithProofs" + sv, fn: func() (rideType, error) {
				return invocationToObject(v, nativeBenchScheme, invoke)
			}},
			convBenchObjectCase{name: "ethereumInvocationToObject" + sv, fn: func() (rideType, error) {
				return ethereumInvocationToObject(v, nativeBenchScheme, ethInvoke, payments)
			}},
		)
	}
	cases = append(cases, convBenchObjectCase{name: "invocationToObject/InvokeExpression/V9",
		fn: func() (rideType, error) {
			return invocationToObject(rideast.LibV9, nativeBenchScheme, expression)
		}})
	for _, v := range []rideast.LibraryVersion{rideast.LibV3, rideast.LibV4, rideast.LibV7} {
		cases = append(cases, convBenchObjectCase{name: "blockInfoToObject/V" + strconv.Itoa(int(v)),
			fn: func() (rideType, error) { return blockInfoToObject(&blockInfo, v) }})
	}
	for _, o := range []struct {
		name  string
		order proto.Order
	}{{"OrderV1", orderV1}, {"OrderV4", orderV4}} {
		for _, v := range []rideast.LibraryVersion{rideast.LibV7, rideast.LibV8} {
			cases = append(cases, convBenchObjectCase{name: "orderToObject/" + o.name + "/V" + strconv.Itoa(int(v)),
				fn: func() (rideType, error) { return orderToObject(v, nativeBenchScheme, o.order) }})
		}
	}
	cases = append(cases,
		convBenchObjectCase{name: "assetInfoToObject", fn: func() (rideType, error) {
			return assetInfoToObject(st.assetInfo), nil
		}},
		convBenchObjectCase{name: "fullAssetInfoToObject", fn: func() (rideType, error) {
			return fullAssetInfoToObject(st.fullAssetInfo), nil
		}},
		convBenchObjectCase{name: "balanceDetailsToObject", fn: func() (rideType, error) {
			return balanceDetailsToObject(st.fullBalance), nil
		}},
		convBenchObjectCase{name: "scriptTransferToObject", fn: func() (rideType, error) {
			return scriptTransferToObject(transfer), nil
		}},
		convBenchObjectCase{name: "scriptTransferToTransferTransactionObject", fn: func() (rideType, error) {
			return scriptTransferToTransferTransactionObject(transfer), nil
		}},
	)
	return cases
}

// Benchmark sinks keep the results reachable, so the compiler can't drop the conversions.
var (
	convBenchActionResult  proto.ScriptAction   //nolint:gochecknoglobals // Benchmark sink.
	convBenchActionsResult []proto.ScriptAction //nolint:gochecknoglobals // Benchmark sink.
	convBenchObjectResult  rideType             //nolint:gochecknoglobals // Benchmark sink.
)

func BenchmarkConvertToAction(b *testing.B) {
	f := newConvBenchFixture(b)
	env := f.env()
	for _, c := range convBenchActionCases(f) {
		b.Run(c.name(), func(b *testing.B) {
			_, err := convertToAction(env, c.obj)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchActionResult, _ = convertToAction(env, c.obj)
			}
		})
	}
}

func BenchmarkObjectToActions(b *testing.B) {
	f := newConvBenchFixture(b)
	env := f.envs[rideast.LibV3]
	for _, c := range convBenchV3ResultCases(f) {
		b.Run(c.name, func(b *testing.B) {
			_, err := objectToActions(env, c.obj)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchActionsResult, _ = objectToActions(env, c.obj)
			}
		})
	}
}

func BenchmarkListToActions(b *testing.B) {
	f := newConvBenchFixture(b)
	env := f.env()
	for _, c := range convBenchListCases(f) {
		list, ok := c.obj.(rideList)
		require.True(b, ok)
		b.Run(c.name, func(b *testing.B) {
			_, err := listToActions(env, list)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchActionsResult, _ = listToActions(env, list)
			}
		})
	}
}

func BenchmarkScriptActionToObject(b *testing.B) {
	f := newConvBenchFixture(b)
	pk := f.native.accounts.senderPK
	id := crypto.MustDigestFromBase58("GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9")
	for _, c := range convBenchScriptActionCases(f) {
		b.Run(strings.TrimPrefix(c.name(), "*proto."), func(b *testing.B) {
			_, err := scriptActionToObject(nativeBenchScheme, c.action, pk, id, convBenchTimestamp)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchObjectResult, _ = scriptActionToObject(nativeBenchScheme, c.action, pk, id, convBenchTimestamp)
			}
		})
	}
}

func BenchmarkTransactionToObject(b *testing.B) {
	f := newConvBenchFixture(b)
	for _, c := range convBenchTxCases(b, f) {
		env := f.envs[c.version]
		b.Run(c.name(), func(b *testing.B) {
			_, err := transactionToObject(env, c.tx)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchObjectResult, _ = transactionToObject(env, c.tx)
			}
		})
	}
}

func BenchmarkObjectConverters(b *testing.B) {
	f := newConvBenchFixture(b)
	for _, c := range convBenchObjectCases(b, f) {
		b.Run(c.name, func(b *testing.B) {
			_, err := c.fn()
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				convBenchObjectResult, _ = c.fn()
			}
		})
	}
}

// requireUniqueNames fails if the benchmark case names are not unique.
func requireUniqueNames(t *testing.T, names []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		_, dup := seen[n]
		require.False(t, dup, "duplicate benchmark case %q", n)
		seen[n] = struct{}{}
	}
}

func TestConversionBenchmarkCases(t *testing.T) {
	f := newConvBenchFixture(t)
	t.Run("convertToAction", func(t *testing.T) {
		var names []string
		for _, c := range convBenchActionCases(f) {
			names = append(names, c.name())
			a, err := convertToAction(f.env(), c.obj)
			require.NoError(t, err, c.name())
			require.NotNil(t, a, c.name())
		}
		requireUniqueNames(t, names)
	})
	t.Run("objectToActions", func(t *testing.T) {
		expected := map[string]int{
			writeSetTypeName + "/small": 1, writeSetTypeName + "/large": convBenchMaxEntries,
			transferSetTypeName + "/small": 1, transferSetTypeName + "/large": convBenchMaxTransfers,
			scriptResultTypeName + "/small": 2,
			scriptResultTypeName + "/large": convBenchMaxEntries + convBenchMaxTransfers,
		}
		for _, c := range convBenchV3ResultCases(f) {
			actions, err := objectToActions(f.envs[rideast.LibV3], c.obj)
			require.NoError(t, err, c.name)
			require.Len(t, actions, expected[c.name], c.name)
		}
	})
	t.Run("listToActions", func(t *testing.T) {
		for _, c := range convBenchListCases(f) {
			list, ok := c.obj.(rideList)
			require.True(t, ok)
			actions, err := listToActions(f.env(), list)
			require.NoError(t, err, c.name)
			require.Len(t, actions, len(list), c.name)
			if len(list) == 0 {
				require.Nil(t, actions)
			}
		}
	})
	t.Run("scriptActionToObject", func(t *testing.T) {
		for _, c := range convBenchScriptActionCases(f) {
			r, err := scriptActionToObject(nativeBenchScheme, c.action, f.native.accounts.senderPK, crypto.Digest{},
				convBenchTimestamp)
			require.NoError(t, err, c.name())
			require.NotNil(t, r, c.name())
		}
	})
	t.Run("transactionToObject", func(t *testing.T) {
		var names []string
		for _, c := range convBenchTxCases(t, f) {
			names = append(names, c.name())
			r, err := transactionToObject(f.envs[c.version], c.tx)
			require.NoError(t, err, c.name())
			require.NotNil(t, r, c.name())
		}
		requireUniqueNames(t, names)
	})
	t.Run("objectConverters", func(t *testing.T) {
		var names []string
		for _, c := range convBenchObjectCases(t, f) {
			names = append(names, c.name)
			r, err := c.fn()
			require.NoError(t, err, c.name)
			require.NotNil(t, r, c.name)
		}
		requireUniqueNames(t, names)
	})
}

// switchCases parses converters.go and returns the cases of the first top level switch statement
// of the function. The type cases are returned as written (e.g. "*proto.Genesis"), the identifiers of
// the constants are replaced with their values from consts.
func switchCases(t *testing.T, function string, consts map[string]string) []string {
	t.Helper()
	const file = "converters.go"
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Name.Name != function {
			continue
		}
		for _, s := range fd.Body.List {
			if body := switchBody(s); body != nil {
				r := caseValues(t, body, consts)
				require.NotEmpty(t, r, "no cases in switch of function %q", function)
				return r
			}
		}
		t.Fatalf("no top level switch in function %q", function)
	}
	t.Fatalf("function %q not found in file %q", function, file)
	return nil
}

// switchBody returns the body of the switch or type switch statement, or nil for other statements.
func switchBody(s ast.Stmt) *ast.BlockStmt {
	switch st := s.(type) {
	case *ast.SwitchStmt:
		return st.Body
	case *ast.TypeSwitchStmt:
		return st.Body
	default:
		return nil
	}
}

func caseValues(t *testing.T, body *ast.BlockStmt, consts map[string]string) []string {
	t.Helper()
	var r []string
	for _, s := range body.List {
		cc, ok := s.(*ast.CaseClause)
		require.True(t, ok)
		for _, e := range cc.List {
			c := types.ExprString(e)
			if v, isConst := consts[c]; isConst {
				c = v
			}
			r = append(r, c)
		}
	}
	return r
}

// typeNameConstants returns the values of the string constants declared in the file.
func typeNameConstants(t *testing.T, file string) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	r := make(map[string]string)
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, s := range gd.Specs {
			vs, isValue := s.(*ast.ValueSpec)
			require.True(t, isValue)
			for i, n := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				if lit, isLit := vs.Values[i].(*ast.BasicLit); isLit && lit.Kind == token.STRING {
					v, uErr := strconv.Unquote(lit.Value)
					require.NoError(t, uErr)
					r[n.Name] = v
				}
			}
		}
	}
	return r
}

func requireCovered(t *testing.T, function string, cases []string, benchmarked map[string]struct{}) {
	t.Helper()
	for _, c := range cases {
		_, ok := benchmarked[c]
		require.True(t, ok, "case %q of %s has no benchmark case", c, function)
	}
	for c := range benchmarked {
		require.Contains(t, cases, c, "benchmarked %q is not a case of %s", c, function)
	}
}

func TestConversionBenchmarkCoverage(t *testing.T) {
	consts := typeNameConstants(t, "objects.gen.go")
	f := newConvBenchFixture(t)

	actions := make(map[string]struct{})
	for _, c := range convBenchActionCases(f) {
		actions[reflect.TypeOf(c.obj).Name()] = struct{}{}
	}
	requireCovered(t, "convertToAction", switchCases(t, "convertToAction", consts), actions)

	results := make(map[string]struct{})
	for _, c := range convBenchV3ResultCases(f) {
		results[reflect.TypeOf(c.obj).Name()] = struct{}{}
	}
	requireCovered(t, "objectToActions", switchCases(t, "objectToActions", consts), results)

	scriptActions := make(map[string]struct{})
	for _, c := range convBenchScriptActionCases(f) {
		scriptActions[c.name()] = struct{}{}
	}
	requireCovered(t, "scriptActionToObject", switchCases(t, "scriptActionToObject", consts), scriptActions)

	txs := make(map[string]struct{})
	kinds := make(map[string]struct{})
	for _, c := range convBenchTxCases(t, f) {
		txs[c.key()] = struct{}{}
		if eth, ok := c.tx.(*proto.EthereumTransaction); ok {
			kinds[fmt.Sprintf("%T", eth.TxKind)] = struct{}{}
		}
	}
	requireCovered(t, "transactionToObject", switchCases(t, "transactionToObject", consts), txs)
	requireCovered(t, "ethereumTransactionToObject",
		switchCases(t, "ethereumTransactionToObject", consts), kinds)
}

// TestListToActionsErrors checks that listToActions returns the unwrapped error of the failed conversion.
func TestListToActionsErrors(t *testing.T) {
	f := newConvBenchFixture(t)
	list := rideList{newRideIntegerEntry(nativeBenchKey, 1), rideInt(1)}
	_, err := listToActions(f.env(), list)
	require.EqualError(t, err, "unexpected type 'Int'")
	actions, err := listToActions(f.env(), rideList{})
	require.NoError(t, err)
	require.Nil(t, actions)
}
