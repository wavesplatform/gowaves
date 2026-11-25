package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const schedulerTestScheme = proto.TestNetScheme

// schedulerTestAccount is a deterministic account for scheduler tests.
type schedulerTestAccount struct {
	pk   crypto.PublicKey
	addr proto.WavesAddress
	id   proto.AddressID
}

func newSchedulerTestAccount(t testing.TB, n int) schedulerTestAccount {
	_, pk, err := crypto.GenerateKeyPair([]byte{byte(n), byte(n >> 8), 0xAC})
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(schedulerTestScheme, pk)
	require.NoError(t, err)
	return schedulerTestAccount{pk: pk, addr: addr, id: addr.ID()}
}

func (a schedulerTestAccount) recipient() proto.Recipient {
	return proto.NewRecipientFromAddress(a.addr)
}

func schedulerTestAlias(alias string) proto.Alias {
	return *proto.NewAlias(schedulerTestScheme, alias)
}

func testInvokeTx(
	sender schedulerTestAccount, dApp proto.Recipient, function string, payments ...proto.ScriptPayment,
) *proto.InvokeScriptWithProofs {
	return &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		SenderPK:        sender.pk,
		ScriptRecipient: dApp,
		FunctionCall:    proto.NewFunctionCall(function, nil),
		Payments:        payments,
	}
}

func testTransferTx(sender schedulerTestAccount, recipient proto.Recipient) *proto.TransferWithProofs {
	return &proto.TransferWithProofs{
		Type:     proto.TransferTransaction,
		SenderPK: sender.pk, Recipient: recipient,
	}
}

// schedulerTestStats is a statistics resolver over a map.
type schedulerTestStats map[invokeStatsKey]*invokeStats

func (s schedulerTestStats) get(key invokeStatsKey) *invokeStats { return s[key] }

// record records an execution of the function by the sender with the given actual accesses.
func (s schedulerTestStats) record(
	dApp proto.AddressID, function string, sender proto.AddressID, reads, writes accessSet,
) {
	key := invokeStatsKey{dApp: dApp, function: function}
	if s[key] == nil {
		s[key] = newInvokeStats()
	}
	s[key].update(normalize(reads, sender, dApp), normalize(writes, sender, dApp), false)
}

func TestPredictTxAccess(t *testing.T) {
	sender := newSchedulerTestAccount(t, 1)
	other := newSchedulerTestAccount(t, 2)
	third := newSchedulerTestAccount(t, 3)
	dApp := newSchedulerTestAccount(t, 4)
	extra := newSchedulerTestAccount(t, 5)
	asset, txID := testDigest(7), testDigest(8)
	knownAlias, unknownAlias := schedulerTestAlias("known"), schedulerTestAlias("unknown")

	stats := make(schedulerTestStats)
	stats.record(dApp.id, "call", other.id,
		newAccessSet(accountDataAccessKey(dApp.id), accountBalanceAccessKey(other.id), accountDataAccessKey(extra.id)),
		newAccessSet(accountDataAccessKey(dApp.id), accountBalanceAccessKey(other.id), assetAccessKey(asset)),
	)
	stats[invokeStatsKey{dApp: dApp.id, function: "wide"}] = &invokeStats{wide: true}
	stats.record(dApp.id, "default", other.id, newAccessSet(), newAccessSet(leaseAccessKey(txID)))

	p := &accessPredictor{
		scheme: schedulerTestScheme,
		stats:  stats.get,
		resolveAlias: func(alias proto.Alias) (proto.AddressID, bool) {
			if alias == knownAlias {
				return dApp.id, true
			}
			return proto.AddressID{}, false
		},
	}
	senderBalance := accountBalanceAccessKey(sender.id)
	invokeStaticReads := func(extraKeys ...accessKey) accessSet {
		res := newAccessSet(senderBalance, accountScriptAccessKey(dApp.id), accountDataAccessKey(dApp.id),
			accountBalanceAccessKey(dApp.id))
		res.union(newAccessSet(extraKeys...))
		return res
	}
	order := func(a schedulerTestAccount) *proto.OrderV1 {
		return &proto.OrderV1{SenderPK: a.pk}
	}
	for _, tc := range []struct {
		name         string
		tx           proto.Transaction
		reads        accessSet
		writes       accessSet
		barrier      bool
		speculatable bool
	}{
		{
			name:         "invoke without statistics",
			tx:           testInvokeTx(sender, dApp.recipient(), "unknown"),
			reads:        invokeStaticReads(),
			writes:       newAccessSet(senderBalance),
			speculatable: true,
		},
		{
			name: "invoke without statistics with payments",
			tx: testInvokeTx(sender, dApp.recipient(), "unknown",
				proto.ScriptPayment{Amount: 1, Asset: proto.NewOptionalAssetWaves()},
				proto.ScriptPayment{Amount: 1, Asset: *proto.NewOptionalAssetFromDigest(asset)}),
			reads:        invokeStaticReads(assetAccessKey(asset)),
			writes:       newAccessSet(senderBalance, accountBalanceAccessKey(dApp.id)),
			speculatable: true,
		},
		{
			name:         "invoke with statistics",
			tx:           testInvokeTx(sender, dApp.recipient(), "call"),
			reads:        invokeStaticReads(accountDataAccessKey(extra.id)),
			writes:       newAccessSet(senderBalance, accountDataAccessKey(dApp.id), assetAccessKey(asset)),
			speculatable: true,
		},
		{
			name:         "invoke of default function with statistics",
			tx:           testInvokeTx(sender, dApp.recipient(), ""),
			reads:        invokeStaticReads(),
			writes:       newAccessSet(senderBalance, leaseAccessKey(txID)),
			speculatable: true,
		},
		{
			name:         "invoke with wide statistics",
			tx:           testInvokeTx(sender, dApp.recipient(), "wide"),
			reads:        invokeStaticReads(),
			writes:       newAccessSet(senderBalance),
			barrier:      true,
			speculatable: true,
		},
		{
			name:         "invoke by known alias",
			tx:           testInvokeTx(sender, proto.NewRecipientFromAlias(knownAlias), "call"),
			reads:        invokeStaticReads(aliasAccessKey("known"), accountDataAccessKey(extra.id)),
			writes:       newAccessSet(senderBalance, accountDataAccessKey(dApp.id), assetAccessKey(asset)),
			speculatable: true,
		},
		{
			name:         "invoke by unknown alias",
			tx:           testInvokeTx(sender, proto.NewRecipientFromAlias(unknownAlias), "call"),
			reads:        newAccessSet(senderBalance, aliasAccessKey("unknown")),
			writes:       newAccessSet(senderBalance),
			speculatable: true,
		},
		{
			name: "invoke of itself",
			tx:   testInvokeTx(dApp, dApp.recipient(), "call"),
			reads: newAccessSet(accountScriptAccessKey(dApp.id), accountDataAccessKey(dApp.id),
				accountBalanceAccessKey(dApp.id), accountDataAccessKey(extra.id)),
			writes: newAccessSet(accountBalanceAccessKey(dApp.id), accountDataAccessKey(dApp.id),
				assetAccessKey(asset)),
			speculatable: true,
		},
		{
			name:   "payment",
			tx:     &proto.Payment{SenderPK: sender.pk, Recipient: other.addr},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id)),
		},
		{
			name: "transfer with signature",
			tx: &proto.TransferWithSig{
				SenderPK: sender.pk, Recipient: other.recipient(),
			},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id)),
		},
		{
			name:   "transfer with proofs",
			tx:     testTransferTx(sender, other.recipient()),
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id)),
		},
		{
			name:   "transfer to known alias",
			tx:     testTransferTx(sender, proto.NewRecipientFromAlias(knownAlias)),
			reads:  newAccessSet(aliasAccessKey("known")),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(dApp.id)),
		},
		{
			name:   "transfer to unknown alias",
			tx:     testTransferTx(sender, proto.NewRecipientFromAlias(unknownAlias)),
			reads:  newAccessSet(aliasAccessKey("unknown")),
			writes: newAccessSet(senderBalance),
		},
		{
			name: "mass transfer",
			tx: &proto.MassTransferWithProofs{SenderPK: sender.pk, Transfers: []proto.MassTransferEntry{
				{Recipient: other.recipient(), Amount: 1},
				{Recipient: proto.NewRecipientFromAlias(knownAlias), Amount: 2},
				{Recipient: proto.NewRecipientFromAlias(unknownAlias), Amount: 3},
				{Recipient: third.recipient(), Amount: 4},
			}},
			reads: newAccessSet(aliasAccessKey("known"), aliasAccessKey("unknown")),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id),
				accountBalanceAccessKey(dApp.id), accountBalanceAccessKey(third.id)),
		},
		{
			name: "lease with signature",
			tx: &proto.LeaseWithSig{
				ID: &txID, SenderPK: sender.pk, Recipient: other.recipient(),
			},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id), leaseAccessKey(txID)),
		},
		{
			name: "lease with proofs to alias without ID",
			tx: &proto.LeaseWithProofs{
				SenderPK: sender.pk, Recipient: proto.NewRecipientFromAlias(knownAlias),
			},
			reads:  newAccessSet(aliasAccessKey("known")),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(dApp.id)),
		},
		{
			name:   "data",
			tx:     &proto.DataWithProofs{SenderPK: sender.pk},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountDataAccessKey(sender.id)),
		},
		{
			name:   "set script",
			tx:     &proto.SetScriptWithProofs{SenderPK: sender.pk},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, accountScriptAccessKey(sender.id)),
		},
		{
			name:   "issue with signature",
			tx:     &proto.IssueWithSig{ID: &txID, SenderPK: sender.pk},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(txID)),
		},
		{
			name:   "issue with proofs without ID",
			tx:     &proto.IssueWithProofs{SenderPK: sender.pk},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance),
		},
		{
			name:   "reissue with signature",
			tx:     &proto.ReissueWithSig{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "reissue with proofs",
			tx:     &proto.ReissueWithProofs{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "burn with signature",
			tx:     &proto.BurnWithSig{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "burn with proofs",
			tx:     &proto.BurnWithProofs{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "sponsorship",
			tx:     &proto.SponsorshipWithProofs{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "set asset script",
			tx:     &proto.SetAssetScriptWithProofs{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name:   "update asset info",
			tx:     &proto.UpdateAssetInfoWithProofs{SenderPK: sender.pk, AssetID: asset},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, assetAccessKey(asset)),
		},
		{
			name: "create alias with signature",
			tx: &proto.CreateAliasWithSig{
				SenderPK: sender.pk, Alias: unknownAlias,
			},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, aliasAccessKey("unknown")),
		},
		{
			name: "create alias with proofs",
			tx: &proto.CreateAliasWithProofs{
				SenderPK: sender.pk, Alias: unknownAlias,
			},
			reads:  newAccessSet(),
			writes: newAccessSet(senderBalance, aliasAccessKey("unknown")),
		},
		{
			name:  "exchange with signature",
			tx:    &proto.ExchangeWithSig{SenderPK: sender.pk, Order1: order(other), Order2: order(third)},
			reads: newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id),
				accountBalanceAccessKey(third.id)),
		},
		{
			name:  "exchange with proofs",
			tx:    &proto.ExchangeWithProofs{SenderPK: sender.pk, Order1: order(other), Order2: order(third)},
			reads: newAccessSet(),
			writes: newAccessSet(senderBalance, accountBalanceAccessKey(other.id),
				accountBalanceAccessKey(third.id)),
		},
		{
			name:    "exchange without order",
			tx:      &proto.ExchangeWithProofs{SenderPK: sender.pk, Order1: order(other)},
			reads:   newAccessSet(),
			writes:  newAccessSet(),
			barrier: true,
		},
		{
			name: "genesis", tx: &proto.Genesis{Recipient: other.addr},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
		{
			name: "lease cancel with signature",
			tx: &proto.LeaseCancelWithSig{
				SenderPK: sender.pk, LeaseID: txID,
			},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
		{
			name: "lease cancel with proofs",
			tx: &proto.LeaseCancelWithProofs{
				SenderPK: sender.pk, LeaseID: txID,
			},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
		{
			name: "commit to generation", tx: &proto.CommitToGenerationWithProofs{SenderPK: sender.pk},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
		{
			name: "ethereum", tx: &proto.EthereumTransaction{},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
		{
			name: "invoke expression", tx: &proto.InvokeExpressionTransactionWithProofs{SenderPK: sender.pk},
			reads: newAccessSet(), writes: newAccessSet(), barrier: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := p.predictTxAccess(tc.tx)
			assert.Equal(t, tc.reads.sorted(), actual.reads.sorted(), "reads")
			assert.Equal(t, tc.writes.sorted(), actual.writes.sorted(), "writes")
			assert.Equal(t, tc.barrier, actual.barrier, "barrier")
			assert.Equal(t, tc.speculatable, actual.speculatable, "speculatable")
		})
	}
}

func TestPredictTxAccessWithoutResolvers(t *testing.T) {
	sender, dApp := newSchedulerTestAccount(t, 1), newSchedulerTestAccount(t, 2)
	p := &accessPredictor{scheme: schedulerTestScheme}

	actual := p.predictTxAccess(testInvokeTx(sender, dApp.recipient(), "call"))
	assert.True(t, actual.speculatable)
	assert.False(t, actual.barrier)
	assert.Equal(t, newAccessSet(accountBalanceAccessKey(sender.id)), actual.writes)

	actual = p.predictTxAccess(testTransferTx(sender, proto.NewRecipientFromAlias(schedulerTestAlias("alias"))))
	assert.Equal(t, newAccessSet(aliasAccessKey("alias")), actual.reads)
	assert.Equal(t, newAccessSet(accountBalanceAccessKey(sender.id)), actual.writes)
}

func TestBuildDependencyGraph(t *testing.T) {
	a, b, c := aliasAccessKey("a"), aliasAccessKey("b"), aliasAccessKey("c")
	access := func(reads, writes []accessKey) predictedAccess {
		return predictedAccess{reads: newAccessSet(reads...), writes: newAccessSet(writes...)}
	}
	for _, tc := range []struct {
		name     string
		window   []predictedAccess
		expected [][]int
	}{
		{name: "empty", window: nil, expected: [][]int{}},
		{
			name: "independent",
			window: []predictedAccess{
				access([]accessKey{a}, []accessKey{a}), access([]accessKey{b}, []accessKey{b}),
				access([]accessKey{a, b}, []accessKey{c}),
			},
			expected: [][]int{nil, nil, {0, 1}},
		},
		{
			name: "write after read is not a dependency",
			window: []predictedAccess{
				access([]accessKey{a}, nil), access(nil, []accessKey{a}), access([]accessKey{a}, nil),
			},
			expected: [][]int{nil, nil, {1}},
		},
		{
			name: "write after write is a dependency",
			window: []predictedAccess{
				access(nil, []accessKey{a}), access(nil, []accessKey{a}), access(nil, []accessKey{a, b}),
			},
			expected: [][]int{nil, {0}, {0, 1}},
		},
		{
			name: "barriers",
			window: []predictedAccess{
				access(nil, []accessKey{a}), access(nil, []accessKey{b}), barrierAccess(),
				access([]accessKey{a}, []accessKey{c}), barrierAccess(), access([]accessKey{c}, nil),
			},
			expected: [][]int{nil, nil, {0, 1}, {0, 2}, {0, 1, 2, 3}, {2, 3, 4}},
		},
		{
			name: "barrier with keys at the head",
			window: []predictedAccess{
				{reads: newAccessSet(a), writes: newAccessSet(), barrier: true},
				access([]accessKey{b}, nil),
			},
			expected: [][]int{nil, {0}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := buildDependencyGraph(tc.window)
			assert.Equal(t, tc.expected, g.incoming)
		})
	}
}

func TestSelectBatch(t *testing.T) {
	const accountsCount = 8
	senders := make([]schedulerTestAccount, accountsCount)
	dApps := make([]schedulerTestAccount, accountsCount)
	for i := range accountsCount {
		senders[i] = newSchedulerTestAccount(t, 100+i)
		dApps[i] = newSchedulerTestAccount(t, 200+i)
	}
	alias := schedulerTestAlias("dapp7")

	stats := make(schedulerTestStats)
	// Function "chain" of dApp i writes data of dApp i+1.
	for i := range accountsCount - 1 {
		stats.record(dApps[i].id, "chain", senders[0].id,
			newAccessSet(accountDataAccessKey(dApps[i].id)),
			newAccessSet(accountDataAccessKey(dApps[i+1].id)),
		)
	}
	// Function "write" of every dApp writes its own data, function "read" only reads it.
	for i := range accountsCount {
		own := accountDataAccessKey(dApps[i].id)
		stats.record(dApps[i].id, "write", senders[0].id, newAccessSet(own), newAccessSet(own))
		stats.record(dApps[i].id, "read", senders[0].id, newAccessSet(own), newAccessSet())
	}
	stats[invokeStatsKey{dApp: dApps[0].id, function: "wide"}] = &invokeStats{wide: true}

	p := &accessPredictor{
		scheme: schedulerTestScheme,
		stats:  stats.get,
		resolveAlias: func(a proto.Alias) (proto.AddressID, bool) {
			return proto.AddressID{}, false // The alias is created inside the block.
		},
	}
	invoke := func(sender, dApp int, function string) proto.Transaction {
		return testInvokeTx(senders[sender], dApps[dApp].recipient(), function)
	}
	for _, tc := range []struct {
		name         string
		txs          []proto.Transaction
		alreadyValid []bool
		graph        [][]int
		batch        []int
	}{
		{name: "empty window", txs: nil, graph: [][]int{}, batch: nil},
		{
			name: "fully independent invokes",
			txs: []proto.Transaction{
				invoke(0, 0, "write"), invoke(1, 1, "write"), invoke(2, 2, "write"), invoke(3, 3, "write"),
			},
			graph: [][]int{nil, nil, nil, nil},
			batch: []int{0, 1, 2, 3},
		},
		{
			name: "chain of dependent invokes",
			txs: []proto.Transaction{
				invoke(0, 0, "chain"), invoke(1, 1, "chain"), invoke(2, 2, "chain"), invoke(3, 3, "read"),
			},
			graph: [][]int{nil, {0}, {1}, {2}},
			batch: []int{0},
		},
		{
			name: "same dApp twice",
			txs: []proto.Transaction{
				invoke(0, 0, "write"), invoke(1, 0, "write"), invoke(2, 1, "write"), invoke(3, 0, "read"),
			},
			graph: [][]int{nil, {0}, nil, {0, 1}},
			batch: []int{0, 2},
		},
		{
			name:  "readers of the same dApp are independent",
			txs:   []proto.Transaction{invoke(0, 0, "read"), invoke(1, 0, "read"), invoke(2, 0, "read")},
			graph: [][]int{nil, nil, nil},
			batch: []int{0, 1, 2},
		},
		{
			name:  "same sender twice",
			txs:   []proto.Transaction{invoke(0, 0, "read"), invoke(0, 1, "read"), invoke(1, 2, "read")},
			graph: [][]int{nil, {0}, nil},
			batch: []int{0, 2},
		},
		{
			name: "transfer to a dApp before its invoke",
			txs: []proto.Transaction{
				testTransferTx(senders[0], dApps[0].recipient()), invoke(1, 0, "read"), invoke(2, 1, "read"),
			},
			graph: [][]int{nil, {0}, nil},
			batch: []int{2},
		},
		{
			name: "transfer to the sender of an invoke",
			txs: []proto.Transaction{
				testTransferTx(senders[0], senders[1].recipient()), invoke(1, 0, "read"),
			},
			graph: [][]int{nil, {0}},
			batch: nil,
		},
		{
			name: "data transaction of a dApp before its invoke",
			txs: []proto.Transaction{
				&proto.DataWithProofs{SenderPK: dApps[0].pk}, invoke(1, 0, "read"), invoke(2, 1, "read"),
			},
			graph: [][]int{nil, {0}, nil},
			batch: []int{2},
		},
		{
			name: "set script of a dApp before its invoke",
			txs: []proto.Transaction{
				invoke(0, 1, "read"), &proto.SetScriptWithProofs{SenderPK: dApps[0].pk}, invoke(1, 0, "read"),
			},
			graph: [][]int{nil, nil, {1}},
			batch: []int{0},
		},
		{
			name: "create alias before the invoke by the alias",
			txs: []proto.Transaction{
				&proto.CreateAliasWithProofs{SenderPK: dApps[7].pk, Alias: alias},
				testInvokeTx(senders[0], proto.NewRecipientFromAlias(alias), "read"),
				invoke(1, 1, "read"),
			},
			graph: [][]int{nil, {0}, nil},
			batch: []int{2},
		},
		{
			name: "barrier in the middle",
			txs: []proto.Transaction{
				invoke(0, 0, "read"), invoke(1, 1, "read"),
				&proto.LeaseCancelWithProofs{SenderPK: senders[7].pk},
				invoke(2, 2, "read"), invoke(3, 3, "read"),
			},
			graph: [][]int{nil, nil, {0, 1}, {2}, {2}},
			batch: []int{0, 1},
		},
		{
			name:  "barrier at the head",
			txs:   []proto.Transaction{&proto.EthereumTransaction{}, invoke(0, 0, "read"), invoke(1, 1, "read")},
			graph: [][]int{nil, {0}, {0}},
			batch: nil,
		},
		{
			name:  "wide invoke at the head is speculated alone",
			txs:   []proto.Transaction{invoke(0, 0, "wide"), invoke(1, 1, "read"), invoke(2, 2, "read")},
			graph: [][]int{nil, {0}, {0}},
			batch: []int{0},
		},
		{
			name:  "wide invoke is not speculated with predecessors",
			txs:   []proto.Transaction{invoke(1, 1, "read"), invoke(0, 0, "wide"), invoke(2, 2, "read")},
			graph: [][]int{nil, {0}, {1}},
			batch: []int{0},
		},
		{
			name: "unknown statistics are speculated optimistically",
			txs: []proto.Transaction{
				invoke(0, 0, "unknown"), invoke(1, 0, "unknown"), invoke(2, 0, "another"), invoke(3, 1, "unknown"),
			},
			graph: [][]int{nil, nil, nil, nil},
			batch: []int{0, 1, 2, 3},
		},
		{
			name: "invokes with valid results are not selected but still counted as writers",
			txs: []proto.Transaction{
				invoke(0, 0, "write"), invoke(1, 0, "read"), invoke(2, 1, "write"), invoke(3, 2, "write"),
			},
			alreadyValid: []bool{true, false, true},
			graph:        [][]int{nil, {0}, nil, nil},
			batch:        []int{3},
		},
		{
			name:         "head with a valid result",
			txs:          []proto.Transaction{invoke(0, 0, "write"), invoke(1, 1, "write")},
			alreadyValid: []bool{true, true},
			graph:        [][]int{nil, nil},
			batch:        nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			window := make([]predictedAccess, len(tc.txs))
			for i, tx := range tc.txs {
				window[i] = p.predictTxAccess(tx)
			}
			assert.Equal(t, tc.graph, buildDependencyGraph(window).incoming)
			assert.Equal(t, tc.batch, selectBatch(window, tc.alreadyValid))
		})
	}
}

func TestSelectBatchLookahead(t *testing.T) {
	p := &accessPredictor{scheme: schedulerTestScheme}
	const extra = 10
	window := make([]predictedAccess, 0, invokeSpeculationLookahead+extra)
	for i := range invokeSpeculationLookahead + extra {
		sender, dApp := newSchedulerTestAccount(t, 2*i), newSchedulerTestAccount(t, 2*i+1)
		window = append(window, p.predictTxAccess(testInvokeTx(sender, dApp.recipient(), "call")))
	}
	batch := selectBatch(window, nil)
	require.Len(t, batch, invokeSpeculationLookahead)
	for i, idx := range batch {
		assert.Equal(t, i, idx)
	}
	// The window slides: transactions beyond the bound are selected once they get into it.
	batch = selectBatch(window[invokeSpeculationLookahead:], nil)
	assert.Len(t, batch, extra)

	// A barrier beyond the lookahead bound does not matter, a barrier inside cuts the batch.
	window[invokeSpeculationLookahead] = barrierAccess()
	assert.Len(t, selectBatch(window, nil), invokeSpeculationLookahead)
	window[3] = barrierAccess()
	assert.Equal(t, []int{0, 1, 2}, selectBatch(window, nil))
}
