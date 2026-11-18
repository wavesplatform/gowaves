package ride //nolint:testpackage // White-box tests of the unexported wrapped state internals.

import (
	"cmp"
	"fmt"
	"slices"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const wrappedStateTestBalance = 100_000_000_000

// wrappedStateTestState is the evaluator benchmark state with configurable balances of the sender.
type wrappedStateTestState struct {
	*evaluatorBenchState

	sender        proto.AddressID
	senderProfile types.WavesBalanceProfile
	senderAsset   uint64
}

func (s *wrappedStateTestState) WavesBalanceProfile(id proto.AddressID) (*types.WavesBalanceProfile, error) {
	if id == s.sender {
		p := s.senderProfile
		return &p, nil
	}
	return &types.WavesBalanceProfile{Balance: wrappedStateTestBalance, Generating: wrappedStateTestBalance}, nil
}

func (s *wrappedStateTestState) NewestAssetBalanceByAddressID(id proto.AddressID, _ crypto.Digest) (uint64, error) {
	if id == s.sender {
		return s.senderAsset, nil
	}
	return wrappedStateTestBalance, nil
}

func newWrappedStateTestState(sender proto.WavesAddress, balance, leaseOut int64, asset uint64) *wrappedStateTestState {
	return &wrappedStateTestState{
		evaluatorBenchState: newEvaluatorBenchState(),
		sender:              sender.ID(),
		senderProfile: types.WavesBalanceProfile{
			Balance: uint64(balance), LeaseOut: leaseOut, Generating: uint64(balance),
		},
		senderAsset: asset,
	}
}

func newWrappedStateTestEnvironment(t *testing.T, a evaluatorBenchAccounts, rideV6 bool) *EvaluationEnvironment {
	env, err := NewEnvironment(evaluatorBenchScheme, newEvaluatorBenchState(), 10, 20, true, rideV6, true, false, true)
	require.NoError(t, err)
	env.SetThisFromAddress(a.dApp)
	env.ChooseTakeString(true)
	env.ChooseMaxDataEntriesSize(true)
	env.SetLimit(evaluatorBenchLimit)
	return env
}

func sortedChangedAccounts(ca changedAccounts) []changedAccountKey {
	keys := make([]changedAccountKey, 0, len(ca))
	for k := range ca {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(x, y changedAccountKey) int {
		if c := cmp.Compare(string(x.account.Bytes()), string(y.account.Bytes())); c != 0 {
			return c
		}
		return cmp.Compare(x.asset.String(), y.asset.String())
	})
	return keys
}

func sortedProfiles(p []changedWavesBalancesProfile) []changedWavesBalancesProfile {
	return slices.SortedFunc(slices.Values(p), func(x, y changedWavesBalancesProfile) int {
		return cmp.Compare(string(x.addrID.Bytes()), string(y.addrID.Bytes()))
	})
}

func TestNewEnvironmentWithWrappedStatePayments(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	const (
		wavesAmount = 1_000_000
		assetAmount = 2_000_000
	)
	payments := proto.ScriptPayments{
		{Amount: wavesAmount, Asset: proto.NewOptionalAssetWaves()},
		{Amount: assetAmount, Asset: *proto.NewOptionalAssetFromDigest(a.asset)},
	}
	for _, rv := range evaluatorRideV6Variants {
		t.Run(rv.name, func(t *testing.T) {
			env := newWrappedStateTestEnvironment(t, a, rv.rideV6)
			origSt := env.st
			st := newWrappedStateTestState(a.sender, wrappedStateTestBalance, 0, wrappedStateTestBalance)
			res, err := NewEnvironmentWithWrappedState(env, st, payments, a.sender, true, ast.LibV6, true)
			require.NoError(t, err)

			// The original environment is left untouched.
			assert.Same(t, origSt, env.st)
			assert.False(t, env.isProtobufTransaction)
			// The new environment is a copy with the wrapped state and the protobuf flag.
			assert.True(t, res.isProtobufTransaction)
			exp := *env
			exp.st = res.st
			exp.isProtobufTransaction = true
			exp.check, exp.takeStr = nil, nil
			got := *res
			got.check, got.takeStr = nil, nil
			assert.Equal(t, exp, got)
			assert.Equal(t, rv.rideV6, res.rideV6Activated())
			assert.Equal(t, proto.Height(10), res.internalPaymentsValidationHeight())
			assert.Equal(t, proto.Height(20), res.paymentsFixAfterHeight())

			ws, ok := res.st.(*WrappedState)
			require.True(t, ok)
			assert.Equal(t, a.dApp, ws.callee())
			assert.Equal(t, evaluatorBenchScheme, ws.scheme)
			assert.Equal(t, proto.Height(evaluatorBenchHeight), ws.height)
			assert.Equal(t, ast.LibV6, ws.rootScriptLibVersion)
			assert.True(t, ws.isLightNodeActivated)
			assert.Equal(t, proto.NewScriptActionsCountValidator(), ws.rootActionsCountValidator)
			assert.Empty(t, ws.act)
			assert.Zero(t, ws.invCount())
			assert.Zero(t, ws.dataEntriesSize)

			sender, dApp := a.sender.ID(), a.dApp.ID()
			assert.Equal(t, map[proto.AddressID]diffBalance{
				sender: {balance: wrappedStateTestBalance - wavesAmount, stateGenerating: wrappedStateTestBalance},
				dApp:   {balance: wrappedStateTestBalance + wavesAmount, stateGenerating: wrappedStateTestBalance},
			}, ws.diff.wavesBalances)
			assert.Equal(t, map[assetBalanceKey]assetBalance{
				{id: sender, asset: a.asset}: wrappedStateTestBalance - assetAmount,
				{id: dApp, asset: a.asset}:   wrappedStateTestBalance + assetAmount,
			}, ws.diff.assetBalances)
			expChanged := changedAccounts{}
			expChanged.addWavesBalanceChange(sender)
			expChanged.addWavesBalanceChange(dApp)
			expChanged.addAssetBalanceChange(sender, a.asset)
			expChanged.addAssetBalanceChange(dApp, a.asset)
			assert.Equal(t, sortedChangedAccounts(expChanged), sortedChangedAccounts(ws.diff.changedAccounts))
			assert.Empty(t, ws.diff.data)
			assert.Empty(t, ws.diff.sponsorships)
			assert.Empty(t, ws.diff.newAssetsInfo)
			assert.Empty(t, ws.diff.oldAssetsInfo)
			assert.Empty(t, ws.diff.leases)
			assert.Same(t, st, ws.diff.state)

			// The balances seen through the wrapped state.
			rcp := proto.NewRecipientFromAddress(a.sender)
			b, err := ws.NewestWavesBalance(rcp)
			require.NoError(t, err)
			assert.Equal(t, uint64(wrappedStateTestBalance-wavesAmount), b)
			ab, err := ws.NewestAssetBalance(rcp, a.asset)
			require.NoError(t, err)
			assert.Equal(t, uint64(wrappedStateTestBalance-assetAmount), ab)

			// The root layer of balance changes is empty, the original balances come from the original state.
			last, ok := ws.lbc.lastChanges()
			require.True(t, ok)
			assert.Empty(t, last.changedWavesBalances)
			assert.Empty(t, last.changedLeaseBalances)
			require.Len(t, ws.lbc.changes, 1)
			ws.lbc.addWavesBalanceChanges(ws.diff.changedAccounts.changedWavesBalancesAccounts())
			changed, old, err := ws.lbc.loadWavesBalanceChanges(ws.diff)
			require.NoError(t, err)
			expChangedBalances := []changedWavesBalancesProfile{
				{addrID: sender, diff: ws.diff.wavesBalances[sender]},
				{addrID: dApp, diff: ws.diff.wavesBalances[dApp]},
			}
			assert.Equal(t, sortedProfiles(expChangedBalances), sortedProfiles(changed))
			expOld := []changedWavesBalancesProfile{
				{addrID: sender, diff: diffBalance{balance: wrappedStateTestBalance, stateGenerating: wrappedStateTestBalance}},
				{addrID: dApp, diff: diffBalance{balance: wrappedStateTestBalance, stateGenerating: wrappedStateTestBalance}},
			}
			assert.Equal(t, sortedProfiles(expOld), sortedProfiles(old))
		})
	}
}

func TestLayeredBalanceChangesLayers(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	st := newWrappedStateTestState(a.sender, 5_000, 1_000, 0)
	lbc := newLayeredWavesBalanceChanges(st)
	sender, dApp := a.sender.ID(), a.dApp.ID()

	lbc.addLeaseBalanceChanges(slices.Values([]proto.AddressID{sender}))
	lbc.initNewLayeredWavesBalanceChanges()
	lbc.addWavesBalanceChanges(slices.Values([]proto.AddressID{dApp}))
	lbc.addLeaseBalanceChanges(slices.Values([]proto.AddressID{dApp}))
	require.Len(t, lbc.changes, 2)
	last, ok := lbc.lastChanges()
	require.True(t, ok)
	assert.Equal(t, map[proto.AddressID]struct{}{dApp: {}}, last.changedWavesBalances)
	assert.Equal(t, map[proto.AddressID]struct{}{dApp: {}}, last.changedLeaseBalances)

	lbc.squashLastChanges()
	require.Len(t, lbc.changes, 1)
	last, ok = lbc.lastChanges()
	require.True(t, ok)
	assert.Equal(t, map[proto.AddressID]struct{}{dApp: {}}, last.changedWavesBalances)
	assert.Equal(t, map[proto.AddressID]struct{}{sender: {}, dApp: {}}, last.changedLeaseBalances)
	lbc.squashLastChanges() // Nothing to squash on the root layer.
	require.Len(t, lbc.changes, 1)

	diff := newDiffState(st)
	_, err := diff.loadWavesBalance(dApp)
	require.NoError(t, err)
	require.NoError(t, diff.addWavesBalance(dApp, 7))
	changed, old, err := lbc.loadWavesBalanceChanges(diff)
	require.NoError(t, err)
	assert.Equal(t, []changedWavesBalancesProfile{{addrID: dApp, diff: diffBalance{
		balance: wrappedStateTestBalance + 7, stateGenerating: wrappedStateTestBalance,
	}}}, changed)
	assert.Equal(t, []changedWavesBalancesProfile{{addrID: dApp, diff: diffBalance{
		balance: wrappedStateTestBalance, stateGenerating: wrappedStateTestBalance,
	}}}, old)

	// The original balances are cached once and don't see the later changes of the diff.
	lbc.addWavesBalanceChanges(slices.Values([]proto.AddressID{sender}))
	_, err = diff.loadWavesBalance(sender)
	require.NoError(t, err)
	require.NoError(t, diff.addWavesBalance(sender, -3))
	changed, old, err = lbc.loadWavesBalanceChanges(diff)
	require.NoError(t, err)
	assert.Equal(t, sortedProfiles([]changedWavesBalancesProfile{
		{addrID: sender, diff: diffBalance{balance: 4_997, leaseOut: 1_000, stateGenerating: 5_000}},
		{addrID: dApp, diff: diffBalance{balance: wrappedStateTestBalance + 7, stateGenerating: wrappedStateTestBalance}},
	}), sortedProfiles(changed))
	assert.Equal(t, sortedProfiles([]changedWavesBalancesProfile{
		{addrID: sender, diff: diffBalance{balance: 5_000, leaseOut: 1_000, stateGenerating: 5_000}},
		{addrID: dApp, diff: diffBalance{balance: wrappedStateTestBalance, stateGenerating: wrappedStateTestBalance}},
	}), sortedProfiles(old))

	// Changes without a loaded diff are reported.
	unknown := proto.AddressID{1}
	lbc.addWavesBalanceChanges(slices.Values([]proto.AddressID{unknown}))
	_, _, err = lbc.loadWavesBalanceChanges(diff)
	require.EqualError(t, err, "failed to find waves balance diff by addrID '"+base58.Encode(unknown.Bytes())+"'")

	var empty layeredBalanceChanges
	_, _, err = empty.loadWavesBalanceChanges(diff)
	require.EqualError(t, err, "no layers available")
}

func TestNewEnvironmentWithWrappedStateNotEnoughMoney(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	waves := proto.NewOptionalAssetWaves()
	asset := *proto.NewOptionalAssetFromDigest(a.asset)
	// Regular balance is 1_500_000, available balance is 500_000 because of the lease.
	const (
		balance  = 1_500_000
		leaseOut = 1_000_000
	)
	for _, tc := range []struct {
		payments proto.ScriptPayments
		check    bool
		errV5    string // RideV6 is not activated, the regular balance is checked.
		errV6    string // RideV6 is activated, the available balance is checked.
	}{
		{
			payments: proto.ScriptPayments{{Amount: 400_000, Asset: waves}},
			check:    true,
		},
		{
			payments: proto.ScriptPayments{{Amount: 1_000_000, Asset: waves}},
			check:    true,
			errV6:    "not enough money for tx attached payment #1 of asset 'WAVES' with amount 1000000",
		},
		{
			payments: proto.ScriptPayments{{Amount: 1_000_000, Asset: waves}},
			check:    false,
		},
		{
			payments: proto.ScriptPayments{{Amount: 1_000_000, Asset: waves}, {Amount: 1_000_000, Asset: waves}},
			check:    true,
			errV5:    "not enough money for tx attached payment #2 of asset 'WAVES' with amount 1000000",
			errV6:    "not enough money for tx attached payment #1 of asset 'WAVES' with amount 1000000",
		},
		{
			payments: proto.ScriptPayments{{Amount: 10, Asset: waves}, {Amount: 11, Asset: asset}},
			check:    true,
			errV5:    "not enough money for tx attached payment #2 of asset '" + a.asset.String() + "' with amount 11",
			errV6:    "not enough money for tx attached payment #2 of asset '" + a.asset.String() + "' with amount 11",
		},
		{
			payments: proto.ScriptPayments{{Amount: 10, Asset: asset}, {Amount: 1, Asset: asset}},
			check:    true, // Every payment is checked against the balance after the previous payments.
			errV5:    "not enough money for tx attached payment #2 of asset '" + a.asset.String() + "' with amount 1",
			errV6:    "not enough money for tx attached payment #2 of asset '" + a.asset.String() + "' with amount 1",
		},
		{
			payments: proto.ScriptPayments{{Amount: 4, Asset: asset}, {Amount: 6, Asset: asset}},
			check:    true,
		},
	} {
		for _, rv := range evaluatorRideV6Variants {
			t.Run(fmt.Sprintf("%v/%t/%s", tc.payments, tc.check, rv.name), func(t *testing.T) {
				env := newWrappedStateTestEnvironment(t, a, rv.rideV6)
				st := newWrappedStateTestState(a.sender, balance, leaseOut, 10)
				res, err := NewEnvironmentWithWrappedState(env, st, tc.payments, a.sender, false, ast.LibV5, tc.check)
				expErr := tc.errV5
				if rv.rideV6 {
					expErr = tc.errV6
				}
				if expErr != "" {
					require.EqualError(t, err, expErr)
					assert.Nil(t, res)
					return
				}
				require.NoError(t, err)
				ws, ok := res.st.(*WrappedState)
				require.True(t, ok)
				checkWrappedStatePaymentBalances(t, a, ws, tc.payments, balance, leaseOut, 10)
			})
		}
	}
}

func checkWrappedStatePaymentBalances(
	t *testing.T, a evaluatorBenchAccounts, ws *WrappedState, payments proto.ScriptPayments,
	balance, leaseOut, asset int64,
) {
	var wavesSum, assetSum int64
	for _, p := range payments {
		if p.Asset.Present {
			assetSum += int64(p.Amount)
		} else {
			wavesSum += int64(p.Amount)
		}
	}
	sender, dApp := a.sender.ID(), a.dApp.ID()
	if wavesSum > 0 {
		assert.Equal(t, map[proto.AddressID]diffBalance{
			sender: {balance: balance - wavesSum, leaseOut: leaseOut, stateGenerating: balance},
			dApp:   {balance: wrappedStateTestBalance + wavesSum, stateGenerating: wrappedStateTestBalance},
		}, ws.diff.wavesBalances)
	} else {
		assert.Empty(t, ws.diff.wavesBalances)
	}
	if assetSum > 0 {
		assert.Equal(t, map[assetBalanceKey]assetBalance{
			{id: sender, asset: a.asset}: assetBalance(asset - assetSum),
			{id: dApp, asset: a.asset}:   assetBalance(wrappedStateTestBalance + assetSum),
		}, ws.diff.assetBalances)
	} else {
		assert.Empty(t, ws.diff.assetBalances)
	}
}

func TestCombineActions(t *testing.T) {
	a1 := &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: "a", Value: 1}}
	a2 := &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: "b", Value: 2}}
	a3 := &proto.DataEntryScriptAction{Entry: &proto.IntegerDataEntry{Key: "c", Value: 3}}
	// The actions of the wrapped state are either nil or not empty, they are only appended to a nil slice.
	for _, tc := range []struct {
		ws, call []proto.ScriptAction
	}{
		{nil, nil},
		{nil, []proto.ScriptAction{}},
		{nil, []proto.ScriptAction{a1}},
		{[]proto.ScriptAction{a1}, nil},
		{[]proto.ScriptAction{a1}, []proto.ScriptAction{a2, a3}},
		{[]proto.ScriptAction{a1, a2}, []proto.ScriptAction{a3}},
		{append(make([]proto.ScriptAction, 0, 10), a1), []proto.ScriptAction{a2}}, // Spare capacity of ws actions.
	} {
		exp := append(slices.Clip(tc.ws), tc.call...) // Reference behavior.
		got := combineActions(tc.ws, tc.call)
		assert.Equal(t, exp, got)
		assert.Equal(t, exp == nil, got == nil)
		if cap(tc.ws) > len(tc.ws) {
			// The spare capacity of the wrapped state actions is never written.
			assert.Nil(t, tc.ws[:cap(tc.ws)][len(tc.ws)])
		}
	}
}
