package state

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// smartStateRead is a single call of a types.EnrichedSmartState method.
type smartStateRead struct {
	name   string // Name of the case.
	method string // Name of the called types.EnrichedSmartState method.
	call   func(s types.EnrichedSmartState) (any, error)
}

// smartStateReadResult is the outcome of a smartStateRead, the error is kept as text to be comparable.
type smartStateReadResult struct {
	res any
	err string
}

func (r smartStateRead) do(s types.EnrichedSmartState) smartStateReadResult {
	res, err := r.call(s)
	if err != nil {
		return smartStateReadResult{err: err.Error()}
	}
	return smartStateReadResult{res: res}
}

// concurrentReadsFixture describes the entities stored in the state populated by newConcurrentReadsState.
type concurrentReadsFixture struct {
	dApp, user, newcomer, unknown *testWavesAddrData
	dAppAlias, newAlias           proto.Alias
	asset, smartAsset             crypto.Digest
	lease, newLease               crypto.Digest
	committedTx, pendingTx        proto.Transaction
}

func concurrentReadsGenesisTx(fill byte, recipient proto.WavesAddress) proto.Transaction {
	var sig crypto.Signature
	for i := range sig {
		sig[i] = fill
	}
	return &proto.Genesis{
		Type:      proto.GenesisTransaction,
		Version:   1,
		ID:        &sig,
		Signature: &sig,
		Timestamp: defaultTimestamp,
		Recipient: recipient,
		Amount:    1,
	}
}

func concurrentReadsScriptEstimation(t *testing.T, script proto.Script) scriptEstimation {
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	est, err := ride.EstimateTree(tree, maxEstimatorVersion)
	require.NoError(t, err)
	return scriptEstimation{currentEstimatorVersion: maxEstimatorVersion, estimation: est}
}

// newConcurrentReadsState creates a state with one committed block and one block in progress.
// Both blocks hold accounts with data, balances, scripts, assets, aliases, leases and a transaction, so the reads
// go through DB, the local history storage and the in-memory structures of the block being applied.
func newConcurrentReadsState(t *testing.T) (*stateManager, *concurrentReadsFixture) {
	state, to := createMockStateManager(t, settings.MustMainNetSettings())
	stor := to.entities
	unknown, err := newTestWavesAddrData("9TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk9bc", nil)
	require.NoError(t, err)
	fx := &concurrentReadsFixture{
		dApp:       testGlobal.recipientInfo,
		user:       testGlobal.senderInfo,
		newcomer:   testGlobal.matcherInfo,
		unknown:    unknown,
		dAppAlias:  *proto.NewAlias(to.settings.AddressSchemeCharacter, "committed-alias"),
		newAlias:   *proto.NewAlias(to.settings.AddressSchemeCharacter, "pending-alias"),
		asset:      testGlobal.asset0.assetID,
		smartAsset: testGlobal.asset1.assetID,
		lease:      crypto.MustFastHash([]byte("committed lease")),
		newLease:   crypto.MustFastHash([]byte("pending lease")),
	}
	fx.committedTx = concurrentReadsGenesisTx(0xC0, fx.dApp.addr)
	fx.pendingTx = concurrentReadsGenesisTx(0xC1, fx.user.addr)

	startBlock := func(id, parent proto.BlockID, tx proto.Transaction) {
		header := &proto.BlockHeader{
			Version:            proto.NgBlockVersion,
			Timestamp:          defaultTimestamp,
			BlockSignature:     id.Signature(),
			Parent:             parent,
			GeneratorPublicKey: testGlobal.minerInfo.pk,
			ID:                 id,
		}
		require.NoError(t, to.stateDB.addBlock(id))
		require.NoError(t, to.rw.startBlock(id))
		require.NoError(t, to.rw.writeBlockHeader(header))
		_, wErr := to.rw.writeTransaction(tx, proto.TransactionSucceeded)
		require.NoError(t, wErr)
	}

	// The committed block.
	startBlock(blockID0, genBlockId(0xAA), fx.committedTx)
	for _, f := range []settings.Feature{settings.SmartAccounts, settings.Ride4DApps, settings.FeeSponsorship} {
		require.NoError(t, stor.features.activateFeature(int16(f), &activatedFeaturesRecord{1}, blockID0))
	}
	to.setWavesBalance(t, fx.dApp.addr, balanceProfile{Balance: 1000, LeaseIn: 10}, blockID0)
	to.setWavesBalance(t, fx.user.addr, balanceProfile{Balance: 2000, LeaseOut: 10}, blockID0)
	to.setWavesBalance(t, testGlobal.issuerInfo.addr, balanceProfile{Balance: 3000}, blockID0)
	require.NoError(t, stor.assets.issueAsset(
		proto.AssetIDFromDigest(fx.asset), defaultAssetInfo(proto.DigestTail(fx.asset), true), blockID0))
	require.NoError(t, stor.assets.issueAsset(
		proto.AssetIDFromDigest(fx.smartAsset), defaultAssetInfo(proto.DigestTail(fx.smartAsset), false), blockID0))
	require.NoError(t, stor.sponsoredAssets.sponsorAsset(fx.asset, 5, blockID0))
	require.NoError(t, stor.scriptsStorage.setAssetScript(fx.smartAsset, testGlobal.scriptBytes, blockID0))
	require.NoError(t, stor.scriptsComplexity.saveComplexitiesForAsset(
		fx.smartAsset, concurrentReadsScriptEstimation(t, testGlobal.scriptBytes), blockID0))
	require.NoError(t, stor.balances.setAssetBalance(
		fx.dApp.addr.ID(), proto.AssetIDFromDigest(fx.asset), 100, blockID0))
	to.setScript(t, fx.dApp.pk, testGlobal.scriptBytes, blockID0)
	to.createAlias(t, fx.dApp.addr, fx.dAppAlias.Alias, blockID0)
	require.NoError(t, stor.leases.addLeasing(fx.lease, createLease(t, fx.user.pk, fx.lease), blockID0))
	for _, e := range []proto.DataEntry{
		&proto.IntegerDataEntry{Key: "int", Value: 1},
		&proto.BooleanDataEntry{Key: "bool", Value: true},
		&proto.StringDataEntry{Key: "string", Value: "value"},
		&proto.BinaryDataEntry{Key: "binary", Value: []byte{1, 2, 3}},
	} {
		require.NoError(t, stor.accountsDataStor.appendEntry(fx.dApp.addr, e, blockID0))
	}
	require.NoError(t, to.rw.finishBlock(blockID0))
	to.flush(t)

	// The block in progress, its changes stay in memory.
	startBlock(blockID1, blockID0, fx.pendingTx)
	to.setWavesBalance(t, fx.dApp.addr, balanceProfile{Balance: 1500, LeaseIn: 10}, blockID1)
	to.setWavesBalance(t, fx.newcomer.addr, balanceProfile{Balance: 500}, blockID1)
	require.NoError(t, stor.balances.setAssetBalance(
		fx.user.addr.ID(), proto.AssetIDFromDigest(fx.asset), 7, blockID1))
	to.setScript(t, fx.newcomer.pk, testGlobal.scriptBytes, blockID1)
	to.createAlias(t, fx.newcomer.addr, fx.newAlias.Alias, blockID1)
	require.NoError(t, stor.leases.addLeasing(fx.newLease, createLease(t, fx.dApp.pk, fx.newLease), blockID1))
	require.NoError(t, stor.accountsDataStor.appendEntry(
		fx.dApp.addr, &proto.IntegerDataEntry{Key: "int", Value: 2}, blockID1))
	require.NoError(t, stor.accountsDataStor.appendEntry(
		fx.newcomer.addr, &proto.StringDataEntry{Key: "string", Value: "new"}, blockID1))
	return state, fx
}

func concurrentSmartStateReads(t *testing.T, fx *concurrentReadsFixture) []smartStateRead {
	var (
		dAppAlias = proto.NewRecipientFromAlias(fx.dAppAlias)
		newAlias  = proto.NewRecipientFromAlias(fx.newAlias)
		noAlias   = *proto.NewAlias(fx.dAppAlias.Scheme, "unknown-alias")
		noDigest  = crypto.MustFastHash([]byte("unknown"))
	)
	committedTxID, err := fx.committedTx.GetID(fx.dAppAlias.Scheme)
	require.NoError(t, err)
	pendingTxID, err := fx.pendingTx.GetID(fx.dAppAlias.Scheme)
	require.NoError(t, err)

	type accountCase struct {
		name string
		rcp  proto.Recipient
	}
	accounts := []accountCase{
		{"dApp", fx.dApp.rcp}, {"dApp by alias", dAppAlias}, {"user", fx.user.rcp},
		{"newcomer", fx.newcomer.rcp}, {"newcomer by alias", newAlias},
		{"unknown", fx.unknown.rcp}, {"unknown alias", proto.NewRecipientFromAlias(noAlias)},
	}
	var reads []smartStateRead
	add := func(method, name string, call func(s types.EnrichedSmartState) (any, error)) {
		reads = append(reads, smartStateRead{name: method + "/" + name, method: method, call: call})
	}
	for _, a := range accounts {
		add("NewestScriptByAccount", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestScriptByAccount(a.rcp)
		})
		add("NewestScriptBytesByAccount", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestScriptBytesByAccount(a.rcp)
		})
		add("NewestRecipientToAddress", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestRecipientToAddress(a.rcp)
		})
		add("IsStateUntouched", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.IsStateUntouched(a.rcp)
		})
		add("NewestWavesBalance", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestWavesBalance(a.rcp)
		})
		add("NewestFullWavesBalance", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestFullWavesBalance(a.rcp)
		})
		add("RetrieveEntries", a.name, func(s types.EnrichedSmartState) (any, error) {
			return s.RetrieveEntries(a.rcp)
		})
		for _, asset := range []crypto.Digest{fx.asset, fx.smartAsset, noDigest} {
			add("NewestAssetBalance", a.name+"/"+asset.String(), func(s types.EnrichedSmartState) (any, error) {
				return s.NewestAssetBalance(a.rcp, asset)
			})
		}
		for _, key := range []string{"int", "bool", "string", "binary", "unknown"} {
			add("RetrieveNewestIntegerEntry", a.name+"/"+key, func(s types.EnrichedSmartState) (any, error) {
				return s.RetrieveNewestIntegerEntry(a.rcp, key)
			})
			add("RetrieveNewestBooleanEntry", a.name+"/"+key, func(s types.EnrichedSmartState) (any, error) {
				return s.RetrieveNewestBooleanEntry(a.rcp, key)
			})
			add("RetrieveNewestStringEntry", a.name+"/"+key, func(s types.EnrichedSmartState) (any, error) {
				return s.RetrieveNewestStringEntry(a.rcp, key)
			})
			add("RetrieveNewestBinaryEntry", a.name+"/"+key, func(s types.EnrichedSmartState) (any, error) {
				return s.RetrieveNewestBinaryEntry(a.rcp, key)
			})
		}
	}
	for _, a := range []*testWavesAddrData{fx.dApp, fx.user, fx.newcomer, fx.unknown, testGlobal.issuerInfo} {
		name := a.addr.String()
		add("NewestScriptPKByAddr", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestScriptPKByAddr(a.addr)
		})
		add("WavesBalanceProfile", name, func(s types.EnrichedSmartState) (any, error) {
			return s.WavesBalanceProfile(a.addr.ID())
		})
		add("NewestAssetBalanceByAddressID", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestAssetBalanceByAddressID(a.addr.ID(), fx.asset)
		})
	}
	for _, al := range []proto.Alias{fx.dAppAlias, fx.newAlias, noAlias} {
		add("NewestAddrByAlias", al.Alias, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestAddrByAlias(al)
		})
	}
	for _, id := range []crypto.Digest{fx.lease, fx.newLease, noDigest} {
		add("NewestLeasingInfo", id.String(), func(s types.EnrichedSmartState) (any, error) {
			return s.NewestLeasingInfo(id)
		})
	}
	for _, asset := range []crypto.Digest{fx.asset, fx.smartAsset, noDigest} {
		name := asset.String()
		add("NewestAssetIsSponsored", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestAssetIsSponsored(asset)
		})
		add("NewestAssetConstInfo", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestAssetConstInfo(proto.AssetIDFromDigest(asset))
		})
		add("NewestAssetInfo", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestAssetInfo(asset)
		})
		add("NewestFullAssetInfo", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestFullAssetInfo(asset)
		})
		add("NewestScriptByAsset", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestScriptByAsset(asset)
		})
	}
	for name, id := range map[string][]byte{
		"committed": committedTxID, "pending": pendingTxID, "unknown": noDigest.Bytes(),
	} {
		add("NewestTransactionByID", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestTransactionByID(id)
		})
		add("NewestTransactionHeightByID", name, func(s types.EnrichedSmartState) (any, error) {
			return s.NewestTransactionHeightByID(id)
		})
	}
	for _, h := range []proto.Height{1, 2, 3} {
		add("NewestBlockInfoByHeight", string(rune('0'+h)), func(s types.EnrichedSmartState) (any, error) {
			return s.NewestBlockInfoByHeight(h)
		})
	}
	add("AddingBlockHeight", "", func(s types.EnrichedSmartState) (any, error) { return s.AddingBlockHeight() })
	add("EstimatorVersion", "", func(s types.EnrichedSmartState) (any, error) { return s.EstimatorVersion() })
	add("IsNotFound", "", func(s types.EnrichedSmartState) (any, error) {
		_, lErr := s.NewestLeasingInfo(noDigest)
		return s.IsNotFound(lErr), nil
	})
	return reads
}

// TestConcurrentSmartStateReads calls all the methods of types.EnrichedSmartState of the real state from many
// goroutines with no writer. Every result must be the same as the result of the sequential call.
// The test is meaningful under the race detector.
func TestConcurrentSmartStateReads(t *testing.T) {
	const (
		goroutines = 16
		rounds     = 5
	)
	state, fx := newConcurrentReadsState(t)
	reads := concurrentSmartStateReads(t, fx)

	t.Run("all methods are called", func(t *testing.T) { assertAllSmartStateMethodsCalled(t, reads) })

	// Sequential pass gives the reference results and checks that the fixture is meaningful.
	expected := make([]smartStateReadResult, len(reads))
	succeeded := make(map[string]bool)
	for i, r := range reads {
		expected[i] = r.do(state)
		if expected[i].err == "" {
			succeeded[r.method] = true
		}
	}
	for _, r := range reads {
		assert.True(t, succeeded[r.method], "no successful call of %s, the fixture does not cover it", r.method)
	}
	height, err := state.AddingBlockHeight()
	require.NoError(t, err)
	require.Equal(t, proto.Height(2), height, "the second block must be in progress")

	// Drop the caches filled by the sequential pass, so the concurrent readers race on misses also.
	require.NoError(t, state.stor.scriptsStorage.clearCache())
	state.stor.features.clearCache()

	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := range goroutines {
		wg.Go(func() {
			<-start
			for round := range rounds {
				// Goroutines walk the calls from different positions to mix the methods.
				if !checkSmartStateReads(t, state, reads, expected, g*len(reads)/goroutines+round) {
					return
				}
			}
		})
	}
	close(start)
	wg.Wait()
}

// assertAllSmartStateMethodsCalled checks that the reads cover exactly the methods of types.EnrichedSmartState.
func assertAllSmartStateMethodsCalled(t *testing.T, reads []smartStateRead) {
	covered := make(map[string]struct{})
	for _, r := range reads {
		covered[r.method] = struct{}{}
	}
	iface := reflect.TypeFor[types.EnrichedSmartState]()
	for method := range iface.Methods() {
		name := method.Name
		assert.Contains(t, covered, name, "method %s is not called", name)
		delete(covered, name)
	}
	assert.Empty(t, covered, "calls of methods that are not in the interface")
}

// checkSmartStateReads performs all the reads starting from the given position and compares the results with
// the expected ones. It returns false on the first difference.
func checkSmartStateReads(
	t *testing.T, state types.EnrichedSmartState, reads []smartStateRead, expected []smartStateReadResult, from int,
) bool {
	for k := range reads {
		i := (k + from) % len(reads)
		if actual := reads[i].do(state); !assert.Equal(t, expected[i], actual, reads[i].name) {
			return false
		}
	}
	return true
}

// TestConcurrentScriptsCache checks the script AST cache under concurrent lookups, insertions and evictions.
func TestConcurrentScriptsCache(t *testing.T) {
	const (
		goroutines = 8
		keys       = 64
		size       = 16 // Less than the number of keys to force evictions.
	)
	cache, err := newLru(size, size*scriptSize)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range 2000 {
				key := []byte{byte((i*7 + g) % keys)}
				switch i % 5 {
				case 0:
					cache.set(key, *testGlobal.scriptAst, scriptSize)
				case 1:
					cache.deleteIfExists(key)
				default:
					if tree, ok := cache.get(key); ok {
						assert.Equal(t, testGlobal.scriptAst.Digest, tree.Digest)
					}
				}
			}
		})
	}
	wg.Wait()
	cache.mu.Lock()
	assert.LessOrEqual(t, cache.size, uint64(size))
	assert.Len(t, cache.m, int(cache.size))
	cache.mu.Unlock()
	cache.clear()
	_, ok := cache.get([]byte{0})
	assert.False(t, ok)
}
