package state

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// Equivalence of the sequential and the parallel execution of invokes.
//
// The harness applies the same blocks to two states, the sequential one and the parallel one, through
// txAppender.appendBlock, the way stateManager.addNewBlock does, and compares everything the application of
// a block leaves behind: the block snapshot, the snapshot state hash and the results of invokes.
// Consensus is not a subject of the tests, so the headers of blocks are not validated, but the blocks are
// well-formed enough to be read back by the reopened state.

const (
	equivalenceWorkers  = 4
	equivalenceAccounts = 6
	// equivalenceFunds is the initial balance of every account, it is enough for thousands of invokes.
	equivalenceFunds = 50_000 * FeeUnit
	// equivalenceHashes is the number of hashes calculated by function work of the script.
	equivalenceHashes = 40
)

// equivalenceScript returns the script of dApps. Function inc adds the given step, so the scripts with different
// steps are distinguishable by their results.
//   - put and inc write the data of the dApp, inc also reads it and returns the new value;
//   - read copies the entry of another account, balance stores the balance of an account, txHeight stores the
//     height of a transaction, none of them fails if there is nothing to read;
//   - call invokes inc of another dApp;
//   - pay transfers from the dApp to the caller, heavyPay does the same after spending more than
//     FailFreeInvokeComplexity;
//   - reject fails cheaply, heavy fails after spending more than FailFreeInvokeComplexity;
//   - work is the CPU consuming version of inc;
//   - assetIssue issues the asset of the dApp and stores its ID, assetReissue, assetBurn and assetPay change
//     the quantity and the balances of the asset of the given issuer, which is the dApp itself for the first two,
//     assetQuantity and assetBalanceOf store them;
//   - deposit adds the amount of the attached payment to the entry;
//   - lease leases from the dApp and stores the ID of the lease, leaseCancel cancels the stored lease if there is
//     one, leaseCancelAgain cancels it without forgetting it, effective stores the effective balance of an account;
//   - callFunction invokes the function without arguments of another dApp.
func equivalenceScript(tb testing.TB, step int) proto.Script {
	sb := new(strings.Builder)
	sb.WriteString("let b0 = toBytes(key) + toBytes(key)\n")
	const doublings = 9
	for i := 1; i <= doublings; i++ {
		fmt.Fprintf(sb, "let b%d = b%d + b%d\n", i, i-1, i-1)
	}
	fmt.Fprintf(sb, "let h0 = sha256(b%d)\n", doublings)
	for i := 1; i <= equivalenceHashes; i++ {
		fmt.Fprintf(sb, "let h%d = sha256(b%d + h%d)\n", i, doublings, i-1)
	}
	src := fmt.Sprintf(`
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func burn(b: ByteVector) =
			sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b) ||
			sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b)

		func assetOf(issuer: String) = getBinaryValue(addressFromStringValue(issuer), "asset")

		@Callable(i)
		func assetIssue() = {
			let a = Issue("Equivalence", "", 1000000, 0, true)
			[a, BinaryEntry("asset", calculateAssetId(a))]
		}

		@Callable(i)
		func assetReissue(amount: Int) = [Reissue(getBinaryValue(this, "asset"), amount, true)]

		@Callable(i)
		func assetBurn(issuer: String, amount: Int) = [Burn(assetOf(issuer), amount)]

		@Callable(i)
		func assetPay(issuer: String, amount: Int) = [ScriptTransfer(i.caller, amount, assetOf(issuer))]

		@Callable(i)
		func assetQuantity(issuer: String, key: String) = {
			let q = match assetInfo(assetOf(issuer)) {
				case a: Asset => a.quantity
				case _ => -1
			}
			[IntegerEntry(key, q)]
		}

		@Callable(i)
		func assetBalanceOf(of: String, issuer: String, key: String) =
			[IntegerEntry(key, assetBalance(addressFromStringValue(of), assetOf(issuer)))]

		@Callable(i)
		func deposit(key: String) = {
			let amount = if (size(i.payments) > 0) then i.payments[0].amount else 0
			[IntegerEntry(key, valueOrElse(getInteger(this, key), 0) + amount)]
		}

		@Callable(i)
		func lease(to: String, amount: Int) = {
			let l = Lease(addressFromStringValue(to), amount)
			[l, BinaryEntry("lease", calculateLeaseId(l))]
		}

		@Callable(i)
		func leaseCancel() = match getBinary(this, "lease") {
			case id: ByteVector => [LeaseCancel(id), DeleteEntry("lease")]
			case _ => []
		}

		@Callable(i)
		func leaseCancelAgain() = [LeaseCancel(getBinaryValue(this, "lease"))]

		@Callable(i)
		func effective(of: String, key: String) =
			[IntegerEntry(key, wavesBalance(addressFromStringValue(of)).effective)]

		@Callable(i)
		func callFunction(callee: String, function: String) = {
			strict r = invoke(addressFromStringValue(callee), function, [], [])
			[]
		}

		@Callable(i)
		func put(key: String, value: Int) = [IntegerEntry(key, value)]

		@Callable(i)
		func inc(key: String) = {
			let v = valueOrElse(getInteger(this, key), 0) + %d
			([IntegerEntry(key, v)], v)
		}

		@Callable(i)
		func read(from: String, key: String) =
			[IntegerEntry(key, valueOrElse(getInteger(addressFromStringValue(from), key), -1))]

		@Callable(i)
		func balance(of: String, key: String) =
			[IntegerEntry(key, wavesBalance(addressFromStringValue(of)).available)]

		@Callable(i)
		func txHeight(id: ByteVector, key: String) =
			[IntegerEntry(key, valueOrElse(transactionHeightById(id), -1))]

		@Callable(i)
		func call(callee: String, key: String) = {
			strict r = invoke(addressFromStringValue(callee), "inc", [key], [])
			match r {
				case v: Int => [IntegerEntry(key, v)]
				case _ => throw("unexpected result of inc")
			}
		}

		@Callable(i)
		func pay(amount: Int) = [ScriptTransfer(i.caller, amount, unit)]

		@Callable(i)
		func heavyPay(amount: Int) =
			if (burn(toBytes(amount))) then [] else [ScriptTransfer(i.caller, amount, unit)]

		@Callable(i)
		func reject() = throw("cheap failure")

		@Callable(i)
		func heavy(key: String) = if (burn(toBytes(key))) then [] else throw("heavy failure")

		@Callable(i)
		func work(key: String) = {
			%s
			[IntegerEntry(key, valueOrElse(getInteger(this, key), 0) + size(h%d))]
		}`, step, sb.String(), equivalenceHashes)
	script, errs := compiler.Compile(src, false, true)
	require.NoError(tb, errors.Join(errs...))
	return script
}

// equivalenceActors are the accounts of the tests, they are the same in all states.
type equivalenceActors struct {
	senders []*testWavesAddrData
	dApps   []*testWavesAddrData
}

func newEquivalenceActors(tb testing.TB, n int) *equivalenceActors {
	account := func(role string, i int) *testWavesAddrData {
		acc, err := newTestWavesAddrData(base58.Encode(fmt.Appendf(nil, "invoke equivalence %s %d", role, i)), nil)
		require.NoError(tb, err)
		return acc
	}
	actors := new(equivalenceActors)
	for i := range n {
		actors.senders = append(actors.senders, account("sender", i))
		actors.dApps = append(actors.dApps, account("dApp", i))
	}
	return actors
}

func (a *equivalenceActors) all() []*testWavesAddrData {
	return append(append([]*testWavesAddrData{}, a.senders...), a.dApps...)
}

// equivalenceState is a state on disk that blocks of transactions are applied to.
type equivalenceState struct {
	tb            testing.TB
	dir           string
	goroutinesNum int
	state         *stateManager
	// counters are the counters of the executors of the states closed by reopen.
	counters invokeExecutionCounters
}

func newEquivalenceState(
	tb testing.TB, goroutinesNum int, actors *equivalenceActors, script proto.Script,
) *equivalenceState {
	s := &equivalenceState{tb: tb, dir: tb.TempDir(), goroutinesNum: goroutinesNum}
	s.open()
	tb.Cleanup(func() { assert.NoError(tb, s.state.Close()) })

	st := s.state
	// Everything prepared for the tests belongs to the genesis block.
	blockID := st.TopBlock().BlockID()
	for _, f := range []settings.Feature{
		settings.NG, settings.SmartAccounts, settings.DataTransaction, settings.Ride4DApps,
		settings.SmartAccountTrading, settings.BlockV5, settings.RideV5,
	} {
		require.NoError(tb, st.stor.features.activateFeature(int16(f), &activatedFeaturesRecord{1}, blockID))
	}
	tree, err := serialization.Parse(script)
	require.NoError(tb, err)
	const estimatorVersion = 1 // The same as in the tests of the invoke applier.
	estimation, err := ride.EstimateTree(tree, estimatorVersion)
	require.NoError(tb, err)
	se := scriptEstimation{currentEstimatorVersion: estimatorVersion, estimation: estimation}
	for _, dApp := range actors.dApps {
		require.NoError(tb, storeScriptByAddress(st.stor, proto.MainNetScheme, dApp.pk, script, se, blockID))
	}
	for _, acc := range actors.all() {
		require.NoError(tb, st.stor.balances.setWavesBalance(acc.addr.ID(),
			wavesValue{profile: balanceProfile{Balance: equivalenceFunds}}, blockID))
	}
	require.NoError(tb, st.flush())
	st.reset()
	s.reopen() // The state is opened again to pick up the activated features.
	return s
}

func (s *equivalenceState) open() {
	params := DefaultTestingStateParams()
	params.InvokeExecutionGoroutinesNum = s.goroutinesNum
	params.StoreExtendedApiData = true
	st, err := newStateManager(s.tb.Context(), s.dir, true, params, settings.MustMainNetSettings(), false, nil)
	require.NoError(s.tb, err)
	require.Equal(s.tb, s.goroutinesNum > 0, st.appender.invokeExec != nil)
	s.state = st
}

// reopen closes the state and opens it again, only the persisted data survive it.
func (s *equivalenceState) reopen() {
	if exec := s.state.appender.invokeExec; exec != nil {
		s.counters = s.counters.add(exec.total)
	}
	require.NoError(s.tb, s.state.Close())
	s.open()
}

func (c invokeExecutionCounters) add(o invokeExecutionCounters) invokeExecutionCounters {
	return invokeExecutionCounters{
		speculated:   c.speculated + o.speculated,
		reused:       c.reused + o.reused,
		mispredicted: c.mispredicted + o.mispredicted,
		inline:       c.inline + o.inline,
	}
}

// total returns the counters of all blocks applied to the state.
func (s *equivalenceState) total() invokeExecutionCounters {
	if exec := s.state.appender.invokeExec; exec != nil {
		return s.counters.add(exec.total)
	}
	return s.counters
}

func (s *equivalenceState) height() proto.Height {
	h, err := s.state.Height()
	require.NoError(s.tb, err)
	return h
}

// equivalenceBlock is a block being applied to the state.
type equivalenceBlock struct {
	id     proto.BlockID
	params *appendBlockParams
}

// beginBlock starts the block of the given transactions on top of the last block of the state the way
// stateManager does before the application of a block.
func (s *equivalenceState) beginBlock(txs []proto.Transaction, timestamp uint64) *equivalenceBlock {
	st := s.state
	height := s.height()
	parent, err := st.HeaderByHeight(height)
	require.NoError(s.tb, err, "height %d", height)
	// The generation signature is the VRF proof, the VRF is checked when the information of the block is built.
	pos := consensus.NewFairPosCalculator(st.settings.DelayDelta, st.settings.MinBlockTime)
	refHitSource, err := st.HitSourceAtHeight(pos.HeightForHit(height))
	require.NoError(s.tb, err)
	genSig, err := crypto.SignVRF(testGlobal.minerInfo.sk, refHitSource)
	require.NoError(s.tb, err)
	ok, vrf, err := crypto.VerifyVRF(testGlobal.minerInfo.pk, refHitSource, genSig)
	require.NoError(s.tb, err)
	require.True(s.tb, ok)
	header := &proto.BlockHeader{
		Version:            proto.ProtobufBlockVersion,
		Timestamp:          timestamp, // It is unique, so a block that failed is never applied again.
		Parent:             parent.BlockID(),
		BaseTarget:         parent.BaseTarget,
		GenSignature:       genSig,
		TransactionCount:   len(txs),
		GeneratorPublicKey: testGlobal.minerInfo.pk,
	}
	require.NoError(s.tb, header.GenerateBlockID(proto.MainNetScheme))
	id := header.BlockID()
	stateHash, err := st.stor.stateHashes.newestSnapshotStateHash(height)
	require.NoError(s.tb, err)
	require.NoError(s.tb, st.stateDB.addBlock(id))
	require.NoError(s.tb, st.stor.hitSources.appendBlockHitSource(&proto.Block{BlockHeader: *header}, height+1, vrf))
	require.NoError(s.tb, st.rw.startBlock(id))
	require.NoError(s.tb, st.rw.writeBlockHeader(header))
	return &equivalenceBlock{id: id, params: &appendBlockParams{
		transactions:          txs,
		chans:                 launchVerifier(s.tb.Context(), runtime.NumCPU(), proto.MainNetScheme),
		block:                 header,
		parent:                parent,
		blockchainHeight:      height,
		lastSnapshotStateHash: stateHash,
	}}
}

// appendBlock applies the transactions of the block and waits for the verification of their signatures.
func (s *equivalenceState) appendBlock(b *equivalenceBlock) error {
	err := s.state.appender.appendBlock(b.params)
	if vErr := b.params.chans.closeAndWait(); err == nil {
		err = vErr
	}
	return err
}

// endBlock saves the applied block or drops everything left of the block that failed.
func (s *equivalenceState) endBlock(b *equivalenceBlock, err error) {
	st := s.state
	if err != nil {
		st.reset()
		require.NoError(s.tb, st.rw.syncWithDb())
		return
	}
	require.NoError(s.tb, st.rw.finishBlock(b.id))
	require.NoError(s.tb, st.flush())
	st.reset()
}

func (s *equivalenceState) applyBlock(txs []proto.Transaction, timestamp uint64) error {
	b := s.beginBlock(txs, timestamp)
	err := s.appendBlock(b)
	s.endBlock(b, err)
	return err
}

func (s *equivalenceState) integerEntry(acc *testWavesAddrData, key string) int64 {
	entry, err := s.state.RetrieveIntegerEntry(acc.rcp, key)
	require.NoError(s.tb, err)
	return entry.Value
}

func (s *equivalenceState) stats(dApp *testWavesAddrData, function string) *invokeStats {
	stats, ok, err := s.state.stor.invokeStats.stats(invokeStatsKey{dApp: dApp.addr.ID(), function: function})
	require.NoError(s.tb, err)
	require.True(s.tb, ok, "no statistics for function %q", function)
	return stats
}

// equivalenceTxBuilder builds the transactions, which are the same for all the states they are applied to.
type equivalenceTxBuilder struct {
	tb        testing.TB
	timestamp uint64
}

func newEquivalenceTxBuilder(tb testing.TB) *equivalenceTxBuilder {
	return &equivalenceTxBuilder{tb: tb, timestamp: defaultTimestamp}
}

func (b *equivalenceTxBuilder) next() uint64 {
	b.timestamp++ // Makes transactions unique.
	return b.timestamp
}

func equivalenceFee(t proto.TransactionType) uint64 { return FeeUnit * feeConstants[t] }

func (b *equivalenceTxBuilder) invokeRecipient(
	sender *testWavesAddrData, dApp proto.Recipient, function string, args ...proto.Argument,
) *proto.InvokeScriptWithProofs {
	return b.invokeWith(sender, dApp, nil, invokeFee, function, args...)
}

// invokeWith builds the invoke with the given payments and fee.
func (b *equivalenceTxBuilder) invokeWith(
	sender *testWavesAddrData, dApp proto.Recipient, payments proto.ScriptPayments, fee uint64,
	function string, args ...proto.Argument,
) *proto.InvokeScriptWithProofs {
	tx := proto.NewUnsignedInvokeScriptWithProofs(2, sender.pk, dApp, proto.NewFunctionCall(function, args),
		payments, feeAsset, fee, b.next())
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

// deposit builds the invoke of function deposit with the payment of the given amount of the asset.
func (b *equivalenceTxBuilder) deposit(
	sender, dApp *testWavesAddrData, asset proto.OptionalAsset, amount uint64, key string,
) *proto.InvokeScriptWithProofs {
	payments := proto.ScriptPayments{{Amount: amount, Asset: asset}}
	return b.invokeWith(sender, dApp.rcp, payments, invokeFee, "deposit", stringArg(key))
}

func (b *equivalenceTxBuilder) lease(from, to *testWavesAddrData, amount uint64) *proto.LeaseWithProofs {
	tx := proto.NewUnsignedLeaseWithProofs(2, from.pk, to.rcp, amount, equivalenceFee(proto.LeaseTransaction), b.next())
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, from.sk))
	return tx
}

func (b *equivalenceTxBuilder) leaseCancel(from *testWavesAddrData, lease crypto.Digest) *proto.LeaseCancelWithProofs {
	tx := proto.NewUnsignedLeaseCancelWithProofs(2, from.pk, lease, equivalenceFee(proto.LeaseCancelTransaction),
		b.next())
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, from.sk))
	return tx
}

func (b *equivalenceTxBuilder) invoke(
	sender, dApp *testWavesAddrData, function string, args ...proto.Argument,
) *proto.InvokeScriptWithProofs {
	return b.invokeRecipient(sender, dApp.rcp, function, args...)
}

func (b *equivalenceTxBuilder) transfer(from, to *testWavesAddrData, amount uint64) *proto.TransferWithProofs {
	tx := proto.NewUnsignedTransferWithProofs(2, from.pk, feeAsset, feeAsset, b.next(), amount,
		equivalenceFee(proto.TransferTransaction), to.rcp, nil)
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, from.sk))
	return tx
}

func (b *equivalenceTxBuilder) data(sender *testWavesAddrData, key string, value int64) *proto.DataWithProofs {
	tx := proto.NewUnsignedDataWithProofs(1, sender.pk, equivalenceFee(proto.DataTransaction), b.next())
	require.NoError(b.tb, tx.AppendEntry(&proto.IntegerDataEntry{Key: key, Value: value}))
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

func (b *equivalenceTxBuilder) setScript(acc *testWavesAddrData, script proto.Script) *proto.SetScriptWithProofs {
	tx := proto.NewUnsignedSetScriptWithProofs(1, acc.pk, script, equivalenceFee(proto.SetScriptTransaction), b.next())
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, acc.sk))
	return tx
}

func (b *equivalenceTxBuilder) createAlias(acc *testWavesAddrData, alias string) *proto.CreateAliasWithProofs {
	tx := proto.NewUnsignedCreateAliasWithProofs(2, acc.pk, *proto.NewAlias(proto.MainNetScheme, alias),
		equivalenceFee(proto.CreateAliasTransaction), b.next())
	require.NoError(b.tb, tx.Sign(proto.MainNetScheme, acc.sk))
	return tx
}

func txDigest(tb testing.TB, tx proto.Transaction) crypto.Digest {
	id, err := tx.GetID(proto.MainNetScheme)
	require.NoError(tb, err)
	d, err := crypto.NewDigestFromBytes(id)
	require.NoError(tb, err)
	return d
}

func addrArg(acc *testWavesAddrData) proto.Argument { return stringArg(acc.addr.String()) }

func binaryArg(v []byte) proto.Argument { return &proto.BinaryArgument{Value: v} }

func intArg(v int64) proto.Argument { return proto.NewIntegerArgument(v) }

// equivalencePair is the pair of states the same blocks are applied to.
type equivalencePair struct {
	t          *testing.T
	actors     *equivalenceActors
	builder    *equivalenceTxBuilder
	sequential *equivalenceState
	parallel   *equivalenceState
	// statuses are the statuses of the transactions of the applied blocks.
	statuses map[crypto.Digest]proto.TransactionStatus
	// applied and failed are the numbers of applied blocks and of blocks that failed.
	applied, failed int
}

func newEquivalencePair(t *testing.T) *equivalencePair {
	actors := newEquivalenceActors(t, equivalenceAccounts)
	script := equivalenceScript(t, 1)
	return &equivalencePair{
		t:          t,
		actors:     actors,
		builder:    newEquivalenceTxBuilder(t),
		sequential: newEquivalenceState(t, 0, actors, script),
		parallel:   newEquivalenceState(t, equivalenceWorkers, actors, script),
		statuses:   make(map[crypto.Digest]proto.TransactionStatus),
	}
}

func (p *equivalencePair) reopen() {
	p.sequential.reopen()
	p.parallel.reopen()
}

// applyBlock applies the block to both states and checks that the results are the same.
func (p *equivalencePair) applyBlock(txs []proto.Transaction) {
	t := p.t
	timestamp := p.builder.next()
	seqErr := p.sequential.applyBlock(txs, timestamp)
	parErr := p.parallel.applyBlock(txs, timestamp)
	if seqErr != nil {
		require.Error(t, parErr, "the block must fail the way it does sequentially: %v", seqErr)
		require.Equal(t, seqErr.Error(), parErr.Error())
		require.Equal(t, p.sequential.height(), p.parallel.height())
		p.failed++
		return
	}
	require.NoError(t, parErr)
	p.applied++
	height := p.sequential.height()
	require.Equal(t, height, p.parallel.height())

	seqHash, err := p.sequential.state.SnapshotStateHashAtHeight(height)
	require.NoError(t, err)
	parHash, err := p.parallel.state.SnapshotStateHashAtHeight(height)
	require.NoError(t, err)
	require.Equal(t, seqHash, parHash, "snapshot state hash at height %d", height)

	seqSnapshot, err := p.sequential.state.SnapshotsAtHeight(height)
	require.NoError(t, err)
	parSnapshot, err := p.parallel.state.SnapshotsAtHeight(height)
	require.NoError(t, err)
	require.Len(t, seqSnapshot.TxSnapshots, len(txs))
	require.Len(t, parSnapshot.TxSnapshots, len(txs))
	for i, tx := range txs {
		// The order of balance snapshots of a transaction is not defined, the state hash does not depend on it.
		require.ElementsMatch(t, seqSnapshot.TxSnapshots[i], parSnapshot.TxSnapshots[i],
			"height %d, transaction %d", height, i)
		id := txDigest(t, tx)
		for _, s := range seqSnapshot.TxSnapshots[i] {
			if status, ok := s.(*proto.TransactionStatusSnapshot); ok {
				p.statuses[id] = status.Status
			}
		}
		if _, ok := tx.(*proto.InvokeScriptWithProofs); !ok {
			continue
		}
		seqRes, seqResErr := p.sequential.state.InvokeResultByID(id)
		parRes, parResErr := p.parallel.state.InvokeResultByID(id)
		require.NoError(t, seqResErr, "height %d, transaction %d", height, i)
		require.NoError(t, parResErr, "height %d, transaction %d", height, i)
		require.Equal(t, seqRes, parRes, "invoke result at height %d, transaction %d", height, i)
	}
}

func (p *equivalencePair) applyBlocks(blocks [][]proto.Transaction) {
	for _, txs := range blocks {
		p.applyBlock(txs)
	}
}

func (p *equivalencePair) status(tx proto.Transaction) proto.TransactionStatus {
	status, ok := p.statuses[txDigest(p.t, tx)]
	require.True(p.t, ok, "transaction was not applied")
	return status
}

// entry returns the value of the entry, which is the same in both states.
func (p *equivalencePair) entry(acc *testWavesAddrData, key string) int64 {
	v := p.sequential.integerEntry(acc, key)
	require.Equal(p.t, v, p.parallel.integerEntry(acc, key))
	return v
}

// issueAssets applies the block in which every dApp issues its asset and returns the IDs of the assets by
// the indexes of the dApps.
func (p *equivalencePair) issueAssets() []crypto.Digest {
	const issueFee = 1000 * FeeUnit // The fee for the asset issued by an invoke.
	a := p.actors
	txs := make([]proto.Transaction, 0, len(a.dApps))
	for i, dApp := range a.dApps {
		txs = append(txs, p.builder.invokeWith(a.senders[i], dApp.rcp, nil, invokeFee+issueFee, "assetIssue"))
	}
	p.applyBlock(txs)
	assets := make([]crypto.Digest, 0, len(a.dApps))
	for _, dApp := range a.dApps {
		seqEntry, err := p.sequential.state.RetrieveBinaryEntry(dApp.rcp, "asset")
		require.NoError(p.t, err)
		parEntry, err := p.parallel.state.RetrieveBinaryEntry(dApp.rcp, "asset")
		require.NoError(p.t, err)
		require.Equal(p.t, seqEntry, parEntry)
		id, err := crypto.NewDigestFromBytes(seqEntry.Value)
		require.NoError(p.t, err)
		assets = append(assets, id)
	}
	return assets
}

// readsOf returns the reads of the function of the dApp called by the sender according to the statistics.
func (p *equivalencePair) readsOf(sender, dApp *testWavesAddrData, function string) accessSet {
	reads, _ := instantiate(p.parallel.stats(dApp, function), sender.addr.ID(), dApp.addr.ID())
	return reads
}

// equivalenceScenario returns the blocks of the given pass of the scenario.
type equivalenceScenario func(p *equivalencePair, pass int) [][]proto.Transaction

const equivalencePasses = 3

// run applies the scenario three times: without statistics, with the statistics collected by the first pass
// and with the statistics loaded from the storage of the reopened state.
func (p *equivalencePair) run(scenario equivalenceScenario) {
	for pass := range equivalencePasses {
		if pass == equivalencePasses-1 {
			p.reopen()
		}
		p.applyBlocks(scenario(p, pass))
	}
	c := p.parallel.total()
	assert.Positive(p.t, c.speculated, "nothing was evaluated in parallel")
	assert.Positive(p.t, c.reused, "no speculative result was used")
	// The results left of a block that failed are dropped, so it is not an equality.
	assert.GreaterOrEqual(p.t, c.speculated, c.reused+c.mispredicted)
}

// background returns the invokes that do not depend on anything else in the scenarios, they make batches
// worth launching. The accounts with the last indexes are reserved for them.
func (p *equivalencePair) background() []proto.Transaction {
	a, last := p.actors, equivalenceAccounts-1
	return []proto.Transaction{
		p.builder.invoke(a.senders[last], a.dApps[last], "inc", stringArg("background")),
		p.builder.invoke(a.senders[last-1], a.dApps[last-1], "inc", stringArg("background")),
	}
}

func TestInvokeEquivalenceManyDApps(t *testing.T) {
	p := newEquivalencePair(t)
	const blocksNum = 2
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, blocksNum)
		for i := range blocks {
			for j, sender := range p.actors.senders { // Every sender calls its own dApp.
				blocks[i] = append(blocks[i], p.builder.invoke(sender, p.actors.dApps[j], "inc", stringArg("counter")))
			}
		}
		return blocks
	})
	const invokes = equivalencePasses * blocksNum * equivalenceAccounts
	for _, dApp := range p.actors.dApps {
		assert.Equal(t, int64(equivalencePasses*blocksNum), p.entry(dApp, "counter"))
		// Statistics survive the reopening of the state.
		stats := p.parallel.stats(dApp, "inc")
		assert.Equal(t, uint64(equivalencePasses*blocksNum), stats.executions)
		assert.Zero(t, stats.mispredictions)
	}
	assert.Equal(t, invokeExecutionCounters{speculated: invokes, reused: invokes}, p.parallel.total())
	assert.Equal(t, equivalencePasses*blocksNum, p.applied)
}

func TestInvokeEquivalenceOneDApp(t *testing.T) {
	p := newEquivalencePair(t)
	p.run(func(p *equivalencePair, pass int) [][]proto.Transaction {
		var txs []proto.Transaction
		dApp := p.actors.dApps[0]
		for i, sender := range p.actors.senders[:4] {
			txs = append(txs, p.builder.invoke(sender, dApp, "inc", stringArg("counter")))
			txs = append(txs, p.builder.invoke(sender, dApp, "put", stringArg("last"), intArg(int64(pass*10+i))))
		}
		return [][]proto.Transaction{append(txs, p.background()...)}
	})
	assert.Equal(t, int64(equivalencePasses*4), p.entry(p.actors.dApps[0], "counter"))
	assert.Equal(t, int64(23), p.entry(p.actors.dApps[0], "last"))
	c := p.parallel.total()
	assert.Positive(t, c.mispredicted, "without statistics the invokes of one dApp are speculated together")
	assert.Positive(t, c.reused)
}

func TestInvokeEquivalenceOneSender(t *testing.T) {
	p := newEquivalencePair(t)
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		var txs []proto.Transaction
		for _, dApp := range p.actors.dApps {
			txs = append(txs, p.builder.invoke(p.actors.senders[0], dApp, "inc", stringArg("counter")))
		}
		// The balance of the sender is changed by every invoke and is read by this one.
		txs = append(txs, p.builder.invoke(p.actors.senders[1], p.actors.dApps[1], "balance",
			addrArg(p.actors.senders[0]), stringArg("sender balance")))
		return [][]proto.Transaction{txs}
	})
	for _, dApp := range p.actors.dApps {
		assert.Equal(t, int64(equivalencePasses), p.entry(dApp, "counter"))
	}
	spent := equivalencePasses * equivalenceAccounts * invokeFee
	assert.Equal(t, int64(equivalenceFunds-spent), p.entry(p.actors.dApps[1], "sender balance"))
}

func TestInvokeEquivalenceNestedInvokesOfSharedCallee(t *testing.T) {
	p := newEquivalencePair(t)
	const callers = 4
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		a := p.actors
		callee := a.dApps[callers]
		var txs []proto.Transaction
		for i := range callers {
			txs = append(txs, p.builder.invoke(a.senders[i], a.dApps[i], "call", addrArg(callee), stringArg("shared")))
		}
		txs = append(txs, p.builder.invoke(a.senders[callers], callee, "inc", stringArg("shared")))
		return [][]proto.Transaction{append(txs, p.background()...)}
	})
	a := p.actors
	assert.Equal(t, int64(equivalencePasses*(callers+1)), p.entry(a.dApps[callers], "shared"))
	// Every caller has stored the value returned by the callee.
	for i := range callers {
		assert.Equal(t, int64((equivalencePasses-1)*(callers+1)+i+1), p.entry(a.dApps[i], "shared"))
	}
	assert.Positive(t, p.parallel.total().mispredicted, "the callee is not known without statistics")
	reads, writes := instantiate(p.parallel.stats(a.dApps[0], "call"), a.senders[0].addr.ID(), a.dApps[0].addr.ID())
	callee := accountDataAccessKey(a.dApps[callers].addr.ID())
	assert.True(t, reads.has(callee) && writes.has(callee), "statistics must have the data of the callee")
}

func TestInvokeEquivalenceTransferToDAppBeforeInvoke(t *testing.T) {
	p := newEquivalencePair(t)
	const amount = 12345
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		a := p.actors
		txs := p.background()
		txs = append(txs,
			p.builder.transfer(a.senders[1], a.dApps[0], amount),
			// The dependency of the invoke on the transfer to its dApp is predicted from the transactions.
			p.builder.invoke(a.senders[0], a.dApps[0], "balance", addrArg(a.dApps[0]), stringArg("balance")),
			// The dependency of the invoke of another dApp is not known until statistics are collected.
			p.builder.invoke(a.senders[2], a.dApps[2], "balance", addrArg(a.dApps[0]), stringArg("balance")),
		)
		return [][]proto.Transaction{txs}
	})
	expected := int64(equivalenceFunds) + equivalencePasses*amount
	assert.Equal(t, expected, p.entry(p.actors.dApps[0], "balance"))
	assert.Equal(t, expected, p.entry(p.actors.dApps[2], "balance"))
	// The result of the invoke of another dApp is invalidated by the transfer, so the invoke is evaluated again
	// when it is reached, statistics get the read either way.
	a := p.actors
	reads, _ := instantiate(p.parallel.stats(a.dApps[2], "balance"), a.senders[2].addr.ID(), a.dApps[2].addr.ID())
	assert.True(t, reads.has(accountBalanceAccessKey(a.dApps[0].addr.ID())))
}

func TestInvokeEquivalenceDataTransactionBeforeInvoke(t *testing.T) {
	p := newEquivalencePair(t)
	p.run(func(p *equivalencePair, pass int) [][]proto.Transaction {
		a := p.actors
		txs := p.background()
		txs = append(txs,
			p.builder.data(a.senders[1], "k", int64(pass+100)),
			p.builder.invoke(a.senders[0], a.dApps[0], "read", addrArg(a.senders[1]), stringArg("k")),
		)
		return [][]proto.Transaction{txs}
	})
	assert.Equal(t, int64(equivalencePasses-1+100), p.entry(p.actors.dApps[0], "k"))
	assert.Positive(t, p.parallel.total().mispredicted)
}

func TestInvokeEquivalenceSetScriptBeforeInvoke(t *testing.T) {
	p := newEquivalencePair(t)
	steps := [equivalencePasses]int{10, 1, 100}
	p.run(func(p *equivalencePair, pass int) [][]proto.Transaction {
		a := p.actors
		txs := p.background()
		txs = append(txs,
			p.builder.setScript(a.dApps[0], equivalenceScript(t, steps[pass])),
			p.builder.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter")),
		)
		return [][]proto.Transaction{txs}
	})
	// Every invoke is evaluated by the script set just before it.
	assert.Equal(t, int64(111), p.entry(p.actors.dApps[0], "counter"))
	// The dependency of an invoke on the script of its dApp is predicted from the transactions.
	assert.Zero(t, p.parallel.total().mispredicted)
}

func TestInvokeEquivalenceCreateAliasBeforeInvoke(t *testing.T) {
	p := newEquivalencePair(t)
	alias := func(pass int) string { return fmt.Sprintf("equivalence-%d", pass) }
	p.run(func(p *equivalencePair, pass int) [][]proto.Transaction {
		a := p.actors
		rcp := func(pass int) proto.Recipient {
			return proto.NewRecipientFromAlias(*proto.NewAlias(proto.MainNetScheme, alias(pass)))
		}
		txs := p.background()
		txs = append(txs,
			p.builder.createAlias(a.dApps[0], alias(pass)),
			p.builder.invokeRecipient(a.senders[0], rcp(pass), "inc", stringArg("counter")),
		)
		if pass > 0 { // The alias created by the previous pass.
			txs = append(txs, p.builder.invokeRecipient(a.senders[1], rcp(pass-1), "inc", stringArg("counter")))
		}
		return [][]proto.Transaction{txs}
	})
	assert.Equal(t, int64(2*equivalencePasses-1), p.entry(p.actors.dApps[0], "counter"))
}

func TestInvokeEquivalenceTransactionOfSameBlockByID(t *testing.T) {
	p := newEquivalencePair(t)
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		a := p.actors
		transfer := p.builder.transfer(a.senders[1], a.senders[2], 1)
		id := txDigest(t, transfer)
		txs := p.background()
		txs = append(txs,
			// The transfer is not in the blockchain yet.
			p.builder.invoke(a.senders[0], a.dApps[0], "txHeight", binaryArg(id.Bytes()), stringArg("before")),
			transfer,
			p.builder.invoke(a.senders[3], a.dApps[1], "txHeight", binaryArg(id.Bytes()), stringArg("after")),
		)
		return [][]proto.Transaction{txs}
	})
	assert.Equal(t, int64(-1), p.entry(p.actors.dApps[0], "before"))
	assert.Equal(t, int64(p.sequential.height()), p.entry(p.actors.dApps[1], "after"))
	assert.Positive(t, p.parallel.total().mispredicted)
}

func TestInvokeEquivalenceScriptFailure(t *testing.T) {
	p := newEquivalencePair(t)
	var failed, rejected []proto.Transaction
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		a := p.actors
		heavy := p.builder.invoke(a.senders[1], a.dApps[1], "heavy", stringArg("k"))
		reject := p.builder.invoke(a.senders[1], a.dApps[1], "reject")
		failed = append(failed, heavy)
		rejected = append(rejected, reject)
		return [][]proto.Transaction{
			{ // The failed invoke is saved to the block.
				p.builder.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter")),
				heavy,
				p.builder.invoke(a.senders[2], a.dApps[1], "inc", stringArg("counter")),
				p.builder.invoke(a.senders[3], a.dApps[3], "inc", stringArg("counter")),
			},
			{ // The rejected invoke makes the block invalid.
				p.builder.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter")),
				reject,
				p.builder.invoke(a.senders[2], a.dApps[2], "inc", stringArg("counter")),
			},
			// The block after the invalid one.
			append(p.background(), p.builder.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter"))),
		}
	})
	assert.Equal(t, equivalencePasses, p.failed)
	assert.Equal(t, 2*equivalencePasses, p.applied)
	for _, tx := range failed {
		assert.Equal(t, proto.TransactionFailed, p.status(tx))
	}
	for _, tx := range rejected {
		assert.NotContains(t, p.statuses, txDigest(t, tx))
	}
	// Nothing of the invalid blocks is applied.
	assert.Equal(t, int64(2*equivalencePasses), p.entry(p.actors.dApps[0], "counter"))
	assert.Equal(t, int64(equivalencePasses), p.entry(p.actors.dApps[1], "counter"))
	_, err := p.parallel.state.RetrieveIntegerEntry(p.actors.dApps[2].rcp, "counter")
	assert.Error(t, err)
}

func TestInvokeEquivalenceInsufficientBalance(t *testing.T) {
	p := newEquivalencePair(t)
	drain := 100 * invokeFee
	amount := int64(equivalenceFunds - drain/2) // The dApp is able to pay it only before it is drained.
	var failed, succeeded []proto.Transaction
	p.run(func(p *equivalencePair, pass int) [][]proto.Transaction {
		a := p.actors
		// Both invokes fail on the validation of their actions after the evaluation of the script. At the heights
		// of the tests such invokes are saved as failed whatever the complexity is.
		heavyPay := p.builder.invoke(a.senders[0], a.dApps[0], "heavyPay", intArg(amount))
		pay := p.builder.invoke(a.senders[1], a.dApps[1], "pay", intArg(amount))
		failed = append(failed, heavyPay, pay)
		txs := append(p.background(),
			p.builder.transfer(a.dApps[0], a.senders[2], drain),
			heavyPay,
			p.builder.transfer(a.dApps[1], a.senders[2], drain),
			pay,
		)
		if pass == 0 { // The same invoke without the transfer before it.
			tx := p.builder.invoke(a.senders[3], a.dApps[3], "heavyPay", intArg(amount))
			succeeded = append(succeeded, tx)
			txs = append(txs, tx)
		}
		return [][]proto.Transaction{txs}
	})
	assert.Zero(t, p.failed)
	for _, tx := range failed {
		assert.Equal(t, proto.TransactionFailed, p.status(tx))
	}
	for _, tx := range succeeded {
		assert.Equal(t, proto.TransactionSucceeded, p.status(tx))
	}
}

// TestInvokeEquivalenceAssets checks the invokes that depend on each other through an asset: its quantity and
// the balances of accounts in it.
func TestInvokeEquivalenceAssets(t *testing.T) {
	const (
		reissued  = 100
		paid      = 10
		deposited = 4
		burnt     = 1
	)
	p := newEquivalencePair(t)
	a := p.actors
	issuer := a.dApps[0]
	asset := p.issueAssets()[0]
	speculated := p.parallel.total().speculated
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		b := p.builder
		// In every block the first invoke changes what is read by the second one, which calls another dApp and
		// is sent by another sender. Nothing in the invokes themselves tells that they depend on each other.
		return [][]proto.Transaction{
			append(p.background(),
				b.invoke(a.senders[0], issuer, "assetReissue", intArg(reissued)),
				b.invoke(a.senders[1], a.dApps[1], "assetQuantity", addrArg(issuer), stringArg("reissued")),
			),
			append(p.background(),
				b.invoke(a.senders[2], issuer, "assetPay", addrArg(issuer), intArg(paid)),
				b.invoke(a.senders[1], a.dApps[2], "assetBalanceOf", addrArg(a.senders[2]), addrArg(issuer),
					stringArg("paid")),
			),
			append(p.background(),
				b.deposit(a.senders[2], a.dApps[3], *proto.NewOptionalAssetFromDigest(asset), deposited, "deposited"),
				b.invoke(a.senders[3], a.dApps[1], "assetBalanceOf", addrArg(a.dApps[3]), addrArg(issuer),
					stringArg("deposited")),
			),
			append(p.background(),
				b.invoke(a.senders[0], issuer, "assetBurn", addrArg(issuer), intArg(burnt)),
				b.invoke(a.senders[3], a.dApps[2], "assetQuantity", addrArg(issuer), stringArg("burnt")),
			),
		}
	})
	const (
		passes = equivalencePasses
		issued = 1000000
	)
	assert.Equal(t, int64(issued+passes*reissued-(passes-1)*burnt), p.entry(a.dApps[1], "reissued"))
	assert.Equal(t, int64(passes*paid-(passes-1)*deposited), p.entry(a.dApps[2], "paid"))
	assert.Equal(t, int64(passes*deposited), p.entry(a.dApps[1], "deposited"))
	assert.Equal(t, int64(issued+passes*(reissued-burnt)), p.entry(a.dApps[2], "burnt"))
	assert.Equal(t, int64(passes*deposited), p.entry(a.dApps[3], "deposited"))
	assert.Zero(t, p.failed)

	// Without statistics the invokes of a block are evaluated together, so the result of every reader is stale.
	c := p.parallel.total()
	assert.GreaterOrEqual(t, c.mispredicted, 4)
	assert.Greater(t, c.speculated, speculated)
	assert.True(t, p.readsOf(a.senders[1], a.dApps[1], "assetQuantity").has(assetAccessKey(asset)))
	reads := p.readsOf(a.senders[1], a.dApps[2], "assetBalanceOf")
	assert.True(t, reads.has(accountBalanceAccessKey(a.senders[2].addr.ID())))
}

// TestInvokeEquivalencePayment checks the invoke of the dApp that receives the payment of an earlier invoke.
func TestInvokeEquivalencePayment(t *testing.T) {
	const amount = 777
	p := newEquivalencePair(t)
	a := p.actors
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		return [][]proto.Transaction{append(p.background(),
			p.builder.deposit(a.senders[0], a.dApps[0], proto.NewOptionalAssetWaves(), amount, "deposited"),
			p.builder.invoke(a.senders[1], a.dApps[0], "balance", addrArg(a.dApps[0]), stringArg("balance")),
		)}
	})
	assert.Equal(t, int64(equivalencePasses*amount), p.entry(a.dApps[0], "deposited"))
	assert.Equal(t, int64(equivalenceFunds+equivalencePasses*amount), p.entry(a.dApps[0], "balance"))
	// The dependency of an invoke on the payment to its dApp is predicted from the transactions.
	assert.Zero(t, p.parallel.total().mispredicted)
}

// TestInvokeEquivalenceLeases checks the invokes that depend on each other through a lease: the balances
// of its sender and its recipient, and the state of the lease itself.
func TestInvokeEquivalenceLeases(t *testing.T) {
	const amount = 1000 * FeeUnit
	p := newEquivalencePair(t)
	a := p.actors
	lessor, recipient := a.dApps[0], a.senders[3] // The recipient sends nothing, so it pays no fees.
	for pass := range equivalencePasses {
		if pass == equivalencePasses-1 {
			p.reopen()
		}
		b := p.builder
		p.applyBlock(append(p.background(),
			b.invoke(a.senders[0], lessor, "lease", addrArg(recipient), intArg(amount)),
			b.invoke(a.senders[1], a.dApps[1], "effective", addrArg(recipient), stringArg("leased")),
		))
		require.Equal(t, int64(equivalenceFunds+amount), p.entry(a.dApps[1], "leased"))

		// Both invokes cancel the same lease by nested invokes of the lessor. The second one is not valid, because
		// the lease is cancelled already, while its evaluation over the state before the first one succeeds.
		failed := p.failed
		p.applyBlock(append(p.background(),
			b.invoke(a.senders[0], a.dApps[1], "callFunction", addrArg(lessor), stringArg("leaseCancelAgain")),
			b.invoke(a.senders[1], a.dApps[2], "callFunction", addrArg(lessor), stringArg("leaseCancelAgain")),
		))
		require.Equal(t, failed+1, p.failed, "the second cancellation of the lease must make the block invalid")

		// The lease is created and cancelled in the same block.
		p.applyBlock(append(p.background(),
			b.invoke(a.senders[2], lessor, "leaseCancel"),
			b.invoke(a.senders[1], a.dApps[2], "effective", addrArg(recipient), stringArg("cancelled")),
			b.invoke(a.senders[0], lessor, "lease", addrArg(recipient), intArg(amount)),
			b.invoke(a.senders[2], lessor, "leaseCancel"),
			b.invoke(a.senders[1], a.dApps[3], "effective", addrArg(recipient), stringArg("cancelled")),
		))
		require.Equal(t, int64(equivalenceFunds), p.entry(a.dApps[2], "cancelled"))
		require.Equal(t, int64(equivalenceFunds), p.entry(a.dApps[3], "cancelled"))
	}
	c := p.parallel.total()
	assert.Positive(t, c.reused)
	assert.Positive(t, c.mispredicted)
	reads := p.readsOf(a.senders[1], a.dApps[1], "effective")
	assert.True(t, reads.has(accountBalanceAccessKey(recipient.addr.ID())))
}

// TestInvokeEquivalenceBlockGeneratorBalance checks the invokes that read the balance of the generator of
// the block. It is changed by the fee of every transaction, but this write is deliberately kept out of
// statistics, so the dependency is never predicted and only the validation of results protects such invokes.
func TestInvokeEquivalenceBlockGeneratorBalance(t *testing.T) {
	p := newEquivalencePair(t)
	a := p.actors
	miner := testGlobal.minerInfo
	// The generator needs funds to send an invoke.
	p.applyBlock([]proto.Transaction{p.builder.transfer(a.senders[0], miner, 100*invokeFee)})
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		b := p.builder
		return [][]proto.Transaction{
			append(p.background(),
				b.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter")),
				b.invoke(a.senders[1], a.dApps[1], "balance", addrArg(miner), stringArg("generator")),
			),
			// The generator is the sender of the invoke.
			append(p.background(),
				b.invoke(miner, a.dApps[2], "inc", stringArg("counter")),
				b.invoke(a.senders[1], a.dApps[1], "balance", addrArg(miner), stringArg("generator")),
			),
		}
	})
	generatorBalance, err := p.sequential.state.WavesBalance(miner.rcp)
	require.NoError(t, err)
	// The last invoke has read the balance before its own fee got to the generator.
	assert.Less(t, uint64(p.entry(a.dApps[1], "generator")), generatorBalance)
	assert.Positive(t, p.entry(a.dApps[1], "generator"))

	generatorKey := accountBalanceAccessKey(miner.addr.ID())
	// In the first block of every pass the result of the reader is invalidated by the fees of the transactions
	// before it, with and without statistics. In the second block the reader is known to depend on the invoke
	// sent by the generator, so it is not evaluated speculatively.
	assert.GreaterOrEqual(t, p.parallel.total().mispredicted, equivalencePasses)
	stats := p.parallel.stats(a.dApps[1], "balance")
	assert.Equal(t, uint64(2*equivalencePasses), stats.executions)
	assert.GreaterOrEqual(t, stats.mispredictions, uint64(equivalencePasses))
	assert.True(t, p.readsOf(a.senders[1], a.dApps[1], "balance").has(generatorKey))
	// The balance of the generator is not among the writes of an invoke unless the generator takes part in it.
	_, writes := instantiate(p.parallel.stats(a.dApps[0], "inc"), a.senders[0].addr.ID(), a.dApps[0].addr.ID())
	assert.False(t, writes.has(generatorKey))
	assert.True(t, writes.has(accountBalanceAccessKey(a.senders[0].addr.ID())))
	_, writes = instantiate(p.parallel.stats(a.dApps[2], "inc"), miner.addr.ID(), a.dApps[2].addr.ID())
	assert.True(t, writes.has(generatorKey))
}

// TestInvokeEquivalenceBlockGeneratorBalanceReaders checks that the invokes with a dependency the prediction
// does not see are not evaluated speculatively over and over again. Every invoke of the blocks reads the balance
// of the generator, so the result of every invoke but the first one is invalidated by the fee of the first
// transaction of the block.
func TestInvokeEquivalenceBlockGeneratorBalanceReaders(t *testing.T) {
	p := newEquivalencePair(t)
	a := p.actors
	miner := testGlobal.minerInfo
	const blocksNum = 2 // Without statistics and with them.
	for range blocksNum {
		txs := make([]proto.Transaction, 0, equivalenceAccounts)
		for i := range equivalenceAccounts {
			txs = append(txs, p.builder.invoke(a.senders[i], a.dApps[i], "balance", addrArg(miner), stringArg("generator")))
		}
		p.applyBlock(txs)
	}
	require.Equal(t, blocksNum, p.applied)
	// Every invoke is evaluated speculatively only once, by the batch launched before the first transaction of
	// the block. The invalidated ones are then evaluated in order.
	n := equivalenceAccounts
	assert.Equal(t, invokeExecutionCounters{
		speculated: blocksNum * n, reused: blocksNum, mispredicted: blocksNum * (n - 1), inline: blocksNum * (n - 1),
	}, p.parallel.total())
	assert.Equal(t, uint64(blocksNum), p.parallel.stats(a.dApps[n-1], "balance").mispredictions)
}

// equivalenceRandom generates random transactions over the accounts of the tests.
type equivalenceRandom struct {
	rnd     *rand.Rand
	actors  *equivalenceActors
	builder *equivalenceTxBuilder
	// ids are the IDs of the generated transactions.
	ids []crypto.Digest
}

func (r *equivalenceRandom) pick(accounts []*testWavesAddrData) *testWavesAddrData {
	return accounts[r.rnd.IntN(len(accounts))]
}

func (r *equivalenceRandom) key() string {
	const keysNum = 3
	return fmt.Sprintf("k%d", r.rnd.IntN(keysNum))
}

// txID returns the ID of a transaction generated earlier, it may be applied already or not.
func (r *equivalenceRandom) txID() []byte {
	if len(r.ids) == 0 {
		return crypto.Digest{}.Bytes()
	}
	return r.ids[r.rnd.IntN(len(r.ids))].Bytes()
}

func (r *equivalenceRandom) tx() proto.Transaction {
	const (
		kinds     = 20
		maxAmount = 1000
	)
	a, b := r.actors, r.builder
	accounts := a.all()
	sender, dApp := r.pick(a.senders), r.pick(a.dApps)
	switch r.rnd.IntN(kinds) {
	case 0, 1:
		return b.transfer(r.pick(accounts), r.pick(accounts), 1+r.rnd.Uint64N(maxAmount))
	case 2, 3:
		return b.data(sender, r.key(), r.rnd.Int64N(maxAmount))
	case 4, 5:
		return b.invoke(sender, dApp, "inc", stringArg(r.key()))
	case 6:
		return b.invoke(sender, dApp, "put", stringArg(r.key()), intArg(r.rnd.Int64N(maxAmount)))
	case 7:
		return b.invoke(sender, dApp, "read", addrArg(r.pick(accounts)), stringArg(r.key()))
	case 8:
		return b.invoke(sender, dApp, "balance", addrArg(r.pick(accounts)), stringArg(r.key()))
	case 9:
		callee := r.pick(a.dApps)
		for callee == dApp {
			callee = r.pick(a.dApps)
		}
		return b.invoke(sender, dApp, "call", addrArg(callee), stringArg(r.key()))
	case 10:
		return b.invoke(sender, dApp, "txHeight", binaryArg(r.txID()), stringArg(r.key()))
	case 11:
		return b.invoke(sender, dApp, "heavy", stringArg(r.key()))
	// The transactions below need the assets issued by the dApps.
	case 12:
		return b.invoke(sender, dApp, "assetPay", addrArg(dApp), intArg(1+r.rnd.Int64N(maxAmount)))
	case 13:
		return b.invoke(sender, dApp, "assetBalanceOf", addrArg(r.pick(accounts)), addrArg(r.pick(a.dApps)),
			stringArg(r.key()))
	case 14:
		return b.invoke(sender, dApp, "assetReissue", intArg(1+r.rnd.Int64N(maxAmount)))
	case 15:
		return b.invoke(sender, dApp, "assetQuantity", addrArg(r.pick(a.dApps)), stringArg(r.key()))
	case 16:
		return b.deposit(sender, dApp, proto.NewOptionalAssetWaves(), 1+r.rnd.Uint64N(maxAmount), r.key())
	case 17: // The recipient of a lease is never a dApp, so a dApp never leases to itself.
		return b.invoke(sender, dApp, "lease", addrArg(r.pick(a.senders)), intArg(1+r.rnd.Int64N(maxAmount)))
	case 18:
		return b.invoke(sender, dApp, "leaseCancel")
	default:
		return b.invoke(sender, dApp, "effective", addrArg(r.pick(accounts)), stringArg(r.key()))
	}
}

// block returns the block of random transactions in random order.
func (r *equivalenceRandom) block(tb testing.TB, size int) []proto.Transaction {
	txs := make([]proto.Transaction, 0, size)
	for range size {
		tx := r.tx()
		r.ids = append(r.ids, txDigest(tb, tx))
		txs = append(txs, tx)
	}
	r.rnd.Shuffle(len(txs), func(i, j int) { txs[i], txs[j] = txs[j], txs[i] })
	return txs
}

// TestInvokeEquivalenceRandom applies random mixes of transfers, data transactions and invokes over a small
// set of accounts, so the transactions of a block depend on each other in all possible ways: through data,
// balances, assets, leases and transactions.
func TestInvokeEquivalenceRandom(t *testing.T) {
	const (
		blocksNum  = 6
		blockSize  = 24
		reopenedAt = blocksNum / 2
	)
	for _, seed := range []uint64{1, 2, 3, 42, 20261125} {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			p := newEquivalencePair(t)
			p.issueAssets()
			r := &equivalenceRandom{rnd: rand.New(rand.NewPCG(seed, seed)), actors: p.actors, builder: p.builder}
			for i := range blocksNum {
				if i == reopenedAt {
					p.reopen()
				}
				p.applyBlock(r.block(t, blockSize))
			}
			require.Equal(t, blocksNum+1, p.applied, "all random blocks are valid")
			c := p.parallel.total()
			assert.Positive(t, c.reused)
			assert.Positive(t, c.mispredicted)
			assert.GreaterOrEqual(t, c.speculated, c.reused+c.mispredicted)
		})
	}
}

// TestInvokeEquivalenceLargeBlocks applies the blocks that are larger than the lookahead window and contain
// transactions which access can not be predicted, so batches are selected up to them.
func TestInvokeEquivalenceLargeBlocks(t *testing.T) {
	const (
		blocksNum = 3
		blockSize = 150
		seed      = 150
	)
	require.Greater(t, blockSize, 2*invokeSpeculationLookahead)
	p := newEquivalencePair(t)
	p.issueAssets()
	a := p.actors
	// Function inc of the first dApp is known to touch too many keys, so its invokes are not predictable.
	keys := newAccessSet()
	for i := range invokeStatsKeysHardCap + 1 {
		keys.add(newAccessKey(accessKindAccountData, fmt.Appendf(nil, "account %d", i)))
	}
	wide := invokeStatsKey{dApp: a.dApps[0].addr.ID(), function: "inc"}
	require.NoError(t, p.parallel.state.stor.invokeStats.update(wide, a.senders[0].addr.ID(), keys, keys, false))
	require.True(t, p.parallel.stats(a.dApps[0], "inc").wide)

	r := &equivalenceRandom{rnd: rand.New(rand.NewPCG(seed, seed)), actors: a, builder: p.builder}
	for range blocksNum {
		txs := r.block(t, blockSize)
		// The cancellation of a lease is not predictable as well: the recipient is known only to the state.
		lease := p.builder.lease(a.senders[0], a.senders[1], 1+r.rnd.Uint64N(1000))
		cancel := p.builder.leaseCancel(a.senders[0], txDigest(t, lease))
		txs = slices.Insert(txs, blockSize/3, proto.Transaction(lease))
		txs = slices.Insert(txs, 2*blockSize/3, proto.Transaction(cancel))
		p.applyBlock(txs)
	}
	require.Equal(t, blocksNum+1, p.applied, "all blocks are valid")
	c := p.parallel.total()
	assert.Positive(t, c.reused)
	assert.Positive(t, c.mispredicted)
	assert.GreaterOrEqual(t, c.speculated, c.reused+c.mispredicted)
	stats := p.parallel.stats(a.dApps[0], "inc")
	assert.True(t, stats.wide)
	assert.Greater(t, stats.executions, uint64(1), "the function must be called by the blocks")
}

// TestInvokeEquivalenceRollback checks the blocks applied after a rollback. Statistics are not rolled back, so
// they describe the executions that are not in the state anymore.
func TestInvokeEquivalenceRollback(t *testing.T) {
	p := newEquivalencePair(t)
	a := p.actors
	// The second invoke copies the entry written by the first one to the data of the source dApp, the third
	// invoke always copies the entry of the first dApp.
	block := func(source *testWavesAddrData, value int64) []proto.Transaction {
		return append(p.background(),
			p.builder.invoke(a.senders[0], source, "put", stringArg("k"), intArg(value)),
			p.builder.invoke(a.senders[1], a.dApps[2], "read", addrArg(source), stringArg("k")),
			p.builder.invoke(a.senders[2], a.dApps[3], "read", addrArg(a.dApps[0]), stringArg("k")),
		)
	}
	const (
		kept       = 7
		rolledBack = 8
		reapplied  = 9
	)
	p.applyBlock(block(a.dApps[0], kept))
	height := p.sequential.height()
	p.applyBlock(block(a.dApps[0], rolledBack))
	require.Equal(t, int64(rolledBack), p.entry(a.dApps[2], "k"))
	executions := p.parallel.stats(a.dApps[2], "read").executions
	require.Equal(t, uint64(2), executions)

	for _, s := range []*equivalenceState{p.sequential, p.parallel} {
		require.NoError(t, s.state.RollbackToHeight(height, false))
		require.Equal(t, height, s.height())
	}
	require.Equal(t, int64(kept), p.entry(a.dApps[2], "k"))
	assert.Equal(t, executions, p.parallel.stats(a.dApps[2], "read").executions, "statistics are not rolled back")

	// The block applied instead of the one rolled back makes the invoke read another dApp, which statistics
	// know nothing about.
	p.applyBlock(block(a.dApps[1], reapplied))
	require.Equal(t, height+1, p.sequential.height())
	assert.Equal(t, int64(reapplied), p.entry(a.dApps[2], "k"))
	assert.Equal(t, int64(kept), p.entry(a.dApps[3], "k"))
	assert.Equal(t, executions+1, p.parallel.stats(a.dApps[2], "read").executions)
	reads := p.readsOf(a.senders[1], a.dApps[2], "read")
	assert.True(t, reads.has(accountDataAccessKey(a.dApps[0].addr.ID())))
	assert.True(t, reads.has(accountDataAccessKey(a.dApps[1].addr.ID())))
	p.reopen()
	assert.Equal(t, int64(reapplied), p.entry(a.dApps[2], "k"))
	assert.Equal(t, 3, p.applied)
	assert.Zero(t, p.failed)
}

// TestInvokeEquivalenceScriptOfOlderVersion checks the invokes of the scripts written before RideV5: they are
// evaluated without the wrapped state.
func TestInvokeEquivalenceScriptOfOlderVersion(t *testing.T) {
	const src = `
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func inc(key: String) = [IntegerEntry(key, valueOrElse(getInteger(this, key), 0) + 1)]

		@Callable(i)
		func read(from: String, key: String) =
			[IntegerEntry(key, valueOrElse(getInteger(addressFromStringValue(from), key), -1))]`
	script, errs := compiler.Compile(src, false, true)
	require.NoError(t, errors.Join(errs...))
	p := newEquivalencePair(t)
	a := p.actors
	p.applyBlock([]proto.Transaction{
		p.builder.setScript(a.dApps[0], script),
		p.builder.setScript(a.dApps[1], script),
	})
	p.run(func(p *equivalencePair, _ int) [][]proto.Transaction {
		return [][]proto.Transaction{append(p.background(),
			p.builder.invoke(a.senders[0], a.dApps[0], "inc", stringArg("counter")),
			p.builder.invoke(a.senders[2], a.dApps[0], "inc", stringArg("counter")),
			p.builder.invoke(a.senders[1], a.dApps[1], "read", addrArg(a.dApps[0]), stringArg("counter")),
		)}
	})
	assert.Equal(t, int64(2*equivalencePasses), p.entry(a.dApps[0], "counter"))
	assert.Equal(t, int64(2*equivalencePasses), p.entry(a.dApps[1], "counter"))
	assert.Positive(t, p.parallel.total().mispredicted)
	assert.True(t, p.readsOf(a.senders[1], a.dApps[1], "read").has(accountDataAccessKey(a.dApps[0].addr.ID())))
}

// BenchmarkAppendBlockInvokes measures the application of a block of CPU consuming invokes. In the independent
// workload every sender calls its own dApp, in the conflicting one all senders call the same function of
// the same dApp with the same key, so every invoke depends on the previous one.
func BenchmarkAppendBlockInvokes(b *testing.B) {
	const invokes = 32
	actors := newEquivalenceActors(b, invokes)
	script := equivalenceScript(b, 1)
	for _, workload := range []struct {
		name string
		dApp func(i int) *testWavesAddrData
	}{
		{name: "independent", dApp: func(i int) *testWavesAddrData { return actors.dApps[i] }},
		{name: "conflicting", dApp: func(int) *testWavesAddrData { return actors.dApps[0] }},
	} {
		for _, mode := range []struct {
			name          string
			goroutinesNum int
		}{
			{name: "sequential", goroutinesNum: 0},
			{name: "parallel", goroutinesNum: runtime.GOMAXPROCS(0)},
		} {
			b.Run(workload.name+"/"+mode.name, func(b *testing.B) {
				s := newEquivalenceState(b, mode.goroutinesNum, actors, script)
				builder := newEquivalenceTxBuilder(b)
				block := func() []proto.Transaction {
					txs := make([]proto.Transaction, 0, invokes)
					for i, sender := range actors.senders {
						txs = append(txs, builder.invoke(sender, workload.dApp(i), "work", stringArg("counter")))
					}
					return txs
				}
				// The first block collects statistics.
				require.NoError(b, s.applyBlock(block(), builder.next()))
				require.Positive(b, s.integerEntry(workload.dApp(0), "counter"), "the invokes must succeed")
				b.ResetTimer()
				for range b.N {
					b.StopTimer()
					blk := s.beginBlock(block(), builder.next())
					b.StartTimer()
					err := s.appendBlock(blk)
					b.StopTimer()
					s.endBlock(blk, err)
					require.NoError(b, err)
					b.StartTimer()
				}
				b.StopTimer()
				if exec := s.state.appender.invokeExec; exec != nil {
					c := exec.total
					b.ReportMetric(float64(c.reused)/float64(c.reused+c.inline), "reused/invoke")
				}
			})
		}
	}
}
