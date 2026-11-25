package state

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const accessTestScheme = proto.TestNetScheme

// has reports whether the set holds the key, it is needed by tests only.
func (s accessSet) has(k accessKey) bool {
	_, ok := s[k]
	return ok
}

func accessTestDigest(b byte) crypto.Digest {
	var d crypto.Digest
	for i := range d {
		d[i] = b
	}
	return d
}

func accessTestAccount(t *testing.T, seed string) (crypto.PublicKey, proto.WavesAddress) {
	t.Helper()
	_, pk, err := crypto.GenerateKeyPair([]byte(seed))
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(accessTestScheme, pk)
	require.NoError(t, err)
	return pk, addr
}

func TestAccessKeyConstructors(t *testing.T) {
	_, addr := accessTestAccount(t, "access-key-constructors")
	id := addr.ID()
	digest := accessTestDigest(0x11)

	var idPayload [accessKeyPayloadSize]byte
	copy(idPayload[:], id.Bytes())
	var aliasPayload [accessKeyPayloadSize]byte
	copy(aliasPayload[:], "alice")
	var assetPayload [accessKeyPayloadSize]byte
	copy(assetPayload[:], digest[:proto.AssetIDSize])

	for _, test := range []struct {
		name     string
		key      accessKey
		expected accessKey
	}{
		{"AccountData", accountDataAccessKey(id), accessKey{accessKindAccountData, idPayload}},
		{"AccountBalance", accountBalanceAccessKey(id), accessKey{accessKindAccountBalance, idPayload}},
		{"AccountScript", accountScriptAccessKey(id), accessKey{accessKindAccountScript, idPayload}},
		{"Asset", assetAccessKey(digest), accessKey{accessKindAsset, assetPayload}},
		{"Alias", aliasAccessKey("alice"), accessKey{accessKindAlias, aliasPayload}},
		{"Lease", leaseAccessKey(digest), accessKey{accessKindLease, digest}},
		{"Tx", txAccessKey(digest.Bytes()), accessKey{accessKindTx, digest}},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.key)
			assert.True(t, strings.HasPrefix(test.key.String(), test.name+"("), test.key.String())
		})
	}
}

func TestAssetAccessKeyIgnoresDigestTail(t *testing.T) {
	digest := accessTestDigest(0x11)
	other := digest
	other[crypto.DigestSize-1] = 0xff // Differs only in the tail that is not a part of the short asset ID.
	assert.Equal(t, assetAccessKey(digest), assetAccessKey(other))
	assert.Equal(t, assetAccessKey(digest), shortAssetAccessKey(proto.AssetIDFromDigest(digest)))
}

func TestAccessKeyDistinctness(t *testing.T) {
	_, addr1 := accessTestAccount(t, "access-key-1")
	_, addr2 := accessTestAccount(t, "access-key-2")
	digest := accessTestDigest(0x22)

	// Same payload, different kinds.
	assert.NotEqual(t, accountDataAccessKey(addr1.ID()), accountBalanceAccessKey(addr1.ID()))
	assert.NotEqual(t, accountDataAccessKey(addr1.ID()), accountScriptAccessKey(addr1.ID()))
	assert.NotEqual(t, assetAccessKey(digest), leaseAccessKey(digest))
	assert.NotEqual(t, assetAccessKey(digest), txAccessKey(digest.Bytes()))
	// Same kind, different payloads.
	assert.NotEqual(t, accountDataAccessKey(addr1.ID()), accountDataAccessKey(addr2.ID()))
	assert.NotEqual(t, aliasAccessKey("alice"), aliasAccessKey("alice1"))
	assert.NotEqual(t, assetAccessKey(digest), assetAccessKey(accessTestDigest(0x23)))
	// Same kind, same payload.
	assert.Equal(t, accountDataAccessKey(addr1.ID()), accountDataAccessKey(addr1.ID()))
	assert.Equal(t, aliasAccessKey("alice"), aliasAccessKey("alice"))
}

func TestAccessKeyLongPayload(t *testing.T) {
	// Transaction IDs of legacy transactions are 64 bytes signatures, they have to be folded into the payload.
	sig1 := bytes.Repeat([]byte{0x01}, crypto.SignatureSize)
	sig2 := bytes.Repeat([]byte{0x02}, crypto.SignatureSize)
	// These two share the first 32 bytes.
	sig3 := append(bytes.Repeat([]byte{0x01}, crypto.DigestSize), bytes.Repeat([]byte{0x03}, crypto.DigestSize)...)

	k1, k2, k3 := txAccessKey(sig1), txAccessKey(sig2), txAccessKey(sig3)
	assert.Equal(t, k1, txAccessKey(sig1))
	assert.NotEqual(t, k1, k2)
	assert.NotEqual(t, k1, k3)

	expected, err := crypto.FastHash(sig1)
	require.NoError(t, err)
	assert.Equal(t, accessKey{kind: accessKindTx, payload: expected}, k1)

	longAlias := strings.Repeat("a", accessKeyPayloadSize+1)
	assert.Equal(t, aliasAccessKey(longAlias), aliasAccessKey(longAlias))
	assert.NotEqual(t, aliasAccessKey(longAlias), aliasAccessKey(longAlias+"a"))
}

func TestAccessKindString(t *testing.T) {
	assert.Equal(t, "Unknown(0)", accessKind(0).String())
	assert.Equal(t, "Unknown(200)", accessKind(200).String())
}

func TestAccessSetOperations(t *testing.T) {
	a := assetAccessKey(accessTestDigest(1))
	b := assetAccessKey(accessTestDigest(2))
	c := leaseAccessKey(accessTestDigest(1))
	d := aliasAccessKey("dave")

	for _, test := range []struct {
		name       string
		left       []accessKey
		right      []accessKey
		intersects bool
		union      []accessKey
	}{
		{"both empty", nil, nil, false, []accessKey{}},
		{"left empty", nil, []accessKey{a}, false, []accessKey{a}},
		{"right empty", []accessKey{a}, nil, false, []accessKey{a}},
		{"equal", []accessKey{a, b}, []accessKey{b, a}, true, []accessKey{a, b}},
		{"disjoint", []accessKey{a, b}, []accessKey{c, d}, false, []accessKey{a, b, d, c}},
		{"same payload different kind", []accessKey{a}, []accessKey{c}, false, []accessKey{a, c}},
		{"one common", []accessKey{a, b, c}, []accessKey{c, d}, true, []accessKey{a, b, d, c}},
		{"subset", []accessKey{a, b, c, d}, []accessKey{d}, true, []accessKey{a, b, d, c}},
		{"superset", []accessKey{d}, []accessKey{a, b, c, d}, true, []accessKey{a, b, d, c}},
		{"duplicates", []accessKey{a, a, a}, []accessKey{b, b}, false, []accessKey{a, b}},
	} {
		t.Run(test.name, func(t *testing.T) {
			left, right := newAccessSet(test.left...), newAccessSet(test.right...)
			rightLen := right.len()

			assert.Equal(t, test.intersects, left.intersects(right))
			assert.Equal(t, test.intersects, right.intersects(left))

			left.union(right)
			assert.Equal(t, test.union, left.sorted())
			assert.Equal(t, len(test.union), left.len())
			assert.Equal(t, rightLen, right.len(), "union must not change the argument")
			for _, k := range test.union {
				assert.True(t, left.has(k))
			}
			if rightLen > 0 {
				assert.True(t, left.intersects(right))
			}

			left.reset()
			assert.Zero(t, left.len())
			assert.Empty(t, left.sorted())
			assert.False(t, left.intersects(right))
			left.add(a) // The set is usable after reset.
			assert.Equal(t, []accessKey{a}, left.sorted())
		})
	}
}

func TestAccessSetZeroValue(t *testing.T) {
	var s accessSet
	a := assetAccessKey(accessTestDigest(1))
	assert.Zero(t, s.len())
	assert.False(t, s.has(a))
	assert.False(t, s.intersects(newAccessSet(a)))
	assert.False(t, newAccessSet(a).intersects(s))
	assert.Empty(t, s.sorted())
	s.reset()
	other := newAccessSet(a)
	other.union(s)
	assert.Equal(t, 1, other.len())
}

func TestAccessSetSortedIsDeterministic(t *testing.T) {
	_, addr1 := accessTestAccount(t, "sorted-1")
	_, addr2 := accessTestAccount(t, "sorted-2")
	keys := []accessKey{
		txAccessKey(accessTestDigest(9).Bytes()),
		leaseAccessKey(accessTestDigest(3)),
		aliasAccessKey("bob"),
		aliasAccessKey("alice"),
		assetAccessKey(accessTestDigest(7)),
		assetAccessKey(accessTestDigest(5)),
		accountScriptAccessKey(addr1.ID()),
		accountBalanceAccessKey(addr2.ID()),
		accountBalanceAccessKey(addr1.ID()),
		accountDataAccessKey(addr1.ID()),
	}
	sorted := newAccessSet(keys...).sorted()
	require.Len(t, sorted, len(keys))
	for i := 1; i < len(sorted); i++ {
		assert.Negative(t, sorted[i-1].compare(sorted[i]), "%s must precede %s", sorted[i-1], sorted[i])
		assert.Positive(t, sorted[i].compare(sorted[i-1]))
		assert.LessOrEqual(t, sorted[i-1].kind, sorted[i].kind)
	}
	assert.Zero(t, sorted[0].compare(sorted[0]))
	for range 10 { // Map iteration order is random, the result must not depend on it.
		assert.Equal(t, sorted, newAccessSet(keys...).sorted())
	}
}

func TestWriteSetFromSnapshot(t *testing.T) {
	senderPK, sender := accessTestAccount(t, "write-set-sender")
	_, recipient := accessTestAccount(t, "write-set-recipient")
	assetID := accessTestDigest(0xA5)
	leaseID := accessTestDigest(0x1E)
	txID := accessTestDigest(0x7C).Bytes()

	for _, test := range []struct {
		name     string
		snapshot txSnapshot
		expected []accessKey
	}{
		{"empty", txSnapshot{}, nil},
		{
			"waves balance",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.WavesBalanceSnapshot{Address: sender, Balance: 100},
			}},
			[]accessKey{accountBalanceAccessKey(sender.ID())},
		},
		{
			"asset balance",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AssetBalanceSnapshot{Address: sender, AssetID: assetID, Balance: 100},
			}},
			[]accessKey{accountBalanceAccessKey(sender.ID())},
		},
		{
			"lease balance",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.LeaseBalanceSnapshot{Address: recipient, LeaseIn: 10},
			}},
			[]accessKey{accountBalanceAccessKey(recipient.ID())},
		},
		{
			"data entries",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.DataEntriesSnapshot{Address: sender, DataEntries: proto.DataEntries{
					&proto.IntegerDataEntry{Key: "k1", Value: 1},
					&proto.DeleteDataEntry{Key: "k2"},
				}},
			}},
			[]accessKey{accountDataAccessKey(sender.ID())},
		},
		{
			"account script",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AccountScriptSnapshot{SenderPublicKey: senderPK, Script: proto.Script{1, 2, 3}},
			}},
			[]accessKey{accountScriptAccessKey(sender.ID())},
		},
		{
			"alias",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AliasSnapshot{Address: sender, Alias: "alice"},
			}},
			[]accessKey{aliasAccessKey("alice")},
		},
		{
			"new asset",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.NewAssetSnapshot{AssetID: assetID, IssuerPublicKey: senderPK, Decimals: 8},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"asset volume",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AssetVolumeSnapshot{AssetID: assetID, TotalQuantity: *big.NewInt(1000), IsReissuable: true},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"asset description",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AssetDescriptionSnapshot{AssetID: assetID, AssetName: "name", AssetDescription: "description"},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"asset script",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.AssetScriptSnapshot{AssetID: assetID, Script: proto.Script{1, 2, 3}},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"sponsorship",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.SponsorshipSnapshot{AssetID: assetID, MinSponsoredFee: 5},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"new lease",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.NewLeaseSnapshot{LeaseID: leaseID, Amount: 10, SenderPK: senderPK, RecipientAddr: recipient},
			}},
			[]accessKey{leaseAccessKey(leaseID)},
		},
		{
			"cancelled lease",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.CancelledLeaseSnapshot{LeaseID: leaseID},
			}},
			[]accessKey{leaseAccessKey(leaseID)},
		},
		{
			"filled volume and fee",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.FilledVolumeFeeSnapshot{OrderID: accessTestDigest(0x0D), FilledVolume: 1, FilledFee: 2},
			}},
			nil,
		},
		{
			"generation commitment",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.GenerationCommitmentSnapshot{SenderPublicKey: senderPK},
			}},
			[]accessKey{accountBalanceAccessKey(sender.ID())},
		},
		{
			"transaction status succeeded",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
			}},
			[]accessKey{txAccessKey(txID)},
		},
		{
			"transaction status failed",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.TransactionStatusSnapshot{Status: proto.TransactionFailed},
			}},
			[]accessKey{txAccessKey(txID)},
		},
		{
			"transaction status elided",
			txSnapshot{regular: []proto.AtomicSnapshot{
				&proto.TransactionStatusSnapshot{Status: proto.TransactionElided},
			}},
			[]accessKey{txAccessKey(txID)},
		},
		{
			"internal dApp complexity",
			txSnapshot{internal: []internalSnapshot{
				&InternalDAppComplexitySnapshot{ScriptAddress: sender},
			}},
			[]accessKey{accountScriptAccessKey(sender.ID())},
		},
		{
			"internal dApp update complexity",
			txSnapshot{internal: []internalSnapshot{
				&InternalDAppUpdateComplexitySnapshot{ScriptAddress: sender},
			}},
			[]accessKey{accountScriptAccessKey(sender.ID())},
		},
		{
			"internal asset script complexity",
			txSnapshot{internal: []internalSnapshot{
				&InternalAssetScriptComplexitySnapshot{AssetID: assetID},
			}},
			[]accessKey{assetAccessKey(assetID)},
		},
		{
			"internal new lease info",
			txSnapshot{internal: []internalSnapshot{
				&InternalNewLeaseInfoSnapshot{LeaseID: leaseID, OriginHeight: 10},
			}},
			[]accessKey{leaseAccessKey(leaseID)},
		},
		{
			"internal cancelled lease info",
			txSnapshot{internal: []internalSnapshot{
				&InternalCancelledLeaseInfoSnapshot{LeaseID: leaseID, CancelHeight: 11},
			}},
			[]accessKey{leaseAccessKey(leaseID)},
		},
		{
			"internal script result",
			txSnapshot{internal: []internalSnapshot{
				&InternalScriptResultSnapshot{ScriptResult: &proto.ScriptResult{}},
			}},
			[]accessKey{txAccessKey(txID)},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			set, err := writeSetFromSnapshot(txID, test.snapshot, accessTestScheme)
			require.NoError(t, err)
			assert.Equal(t, newAccessSet(test.expected...).sorted(), set.sorted())
		})
	}
}

func TestWriteSetFromInvokeSnapshot(t *testing.T) {
	_, sender := accessTestAccount(t, "invoke-sender")
	dAppPK, dApp := accessTestAccount(t, "invoke-dapp")
	_, callee := accessTestAccount(t, "invoke-callee")
	_, miner := accessTestAccount(t, "invoke-miner")
	issuedAsset := accessTestDigest(0x01)
	paymentAsset := accessTestDigest(0x02)
	sponsoredAsset := accessTestDigest(0x03)
	newLease := accessTestDigest(0x04)
	oldLease := accessTestDigest(0x05)
	txID := accessTestDigest(0x06).Bytes()

	snapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{Address: sender, Balance: 1},
			&proto.WavesBalanceSnapshot{Address: dApp, Balance: 2},
			&proto.WavesBalanceSnapshot{Address: miner, Balance: 3},
			&proto.AssetBalanceSnapshot{Address: sender, AssetID: paymentAsset, Balance: 4},
			&proto.AssetBalanceSnapshot{Address: dApp, AssetID: paymentAsset, Balance: 5},
			&proto.AssetBalanceSnapshot{Address: dApp, AssetID: issuedAsset, Balance: 6},
			&proto.NewAssetSnapshot{AssetID: issuedAsset, IssuerPublicKey: dAppPK},
			&proto.AssetVolumeSnapshot{AssetID: issuedAsset, TotalQuantity: *big.NewInt(6)},
			&proto.AssetDescriptionSnapshot{AssetID: issuedAsset, AssetName: "issued"},
			&proto.SponsorshipSnapshot{AssetID: sponsoredAsset, MinSponsoredFee: 1},
			&proto.DataEntriesSnapshot{Address: dApp, DataEntries: proto.DataEntries{
				&proto.StringDataEntry{Key: "k", Value: "v"},
			}},
			&proto.DataEntriesSnapshot{Address: callee, DataEntries: proto.DataEntries{
				&proto.BooleanDataEntry{Key: "called", Value: true},
			}},
			&proto.NewLeaseSnapshot{LeaseID: newLease, Amount: 7, SenderPK: dAppPK, RecipientAddr: callee},
			&proto.CancelledLeaseSnapshot{LeaseID: oldLease},
			&proto.LeaseBalanceSnapshot{Address: dApp, LeaseOut: 7},
			&proto.LeaseBalanceSnapshot{Address: callee, LeaseIn: 7},
			&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
		},
		internal: []internalSnapshot{
			&InternalNewLeaseInfoSnapshot{LeaseID: newLease, OriginHeight: 1},
			&InternalCancelledLeaseInfoSnapshot{LeaseID: oldLease, CancelHeight: 1},
			&InternalDAppUpdateComplexitySnapshot{ScriptAddress: dApp},
			&InternalScriptResultSnapshot{ScriptResult: &proto.ScriptResult{}},
		},
	}
	expected := newAccessSet(
		accountBalanceAccessKey(sender.ID()),
		accountBalanceAccessKey(dApp.ID()),
		accountBalanceAccessKey(miner.ID()),
		accountBalanceAccessKey(callee.ID()),
		accountDataAccessKey(dApp.ID()),
		accountDataAccessKey(callee.ID()),
		accountScriptAccessKey(dApp.ID()),
		assetAccessKey(issuedAsset),
		assetAccessKey(sponsoredAsset),
		leaseAccessKey(newLease),
		leaseAccessKey(oldLease),
		txAccessKey(txID),
	)

	set, err := writeSetFromSnapshot(txID, snapshot, accessTestScheme)
	require.NoError(t, err)
	assert.Equal(t, expected.sorted(), set.sorted())
	// The payment asset itself is not modified, only balances in it.
	assert.False(t, set.has(assetAccessKey(paymentAsset)))
	// The sender only pays, neither its data nor its script change.
	assert.False(t, set.has(accountDataAccessKey(sender.ID())))
	assert.False(t, set.has(accountScriptAccessKey(sender.ID())))
}

func TestWriteSetFromSnapshotWithoutTxID(t *testing.T) {
	_, sender := accessTestAccount(t, "no-tx-id")

	t.Run("fix snapshot without transaction-bound parts", func(t *testing.T) {
		set, err := writeSetFromSnapshot(nil, txSnapshot{regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{Address: sender, Balance: 1},
		}}, accessTestScheme)
		require.NoError(t, err)
		assert.Equal(t, []accessKey{accountBalanceAccessKey(sender.ID())}, set.sorted())
	})
	t.Run("transaction status", func(t *testing.T) {
		_, err := writeSetFromSnapshot(nil, txSnapshot{regular: []proto.AtomicSnapshot{
			&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
		}}, accessTestScheme)
		assert.ErrorContains(t, err, "transaction ID is not set")
	})
	t.Run("script result", func(t *testing.T) {
		_, err := writeSetFromSnapshot([]byte{}, txSnapshot{internal: []internalSnapshot{
			&InternalScriptResultSnapshot{ScriptResult: &proto.ScriptResult{}},
		}}, accessTestScheme)
		assert.ErrorContains(t, err, "transaction ID is not set")
	})
}

func TestWriteSetFromSnapshotUsesScheme(t *testing.T) {
	pk, testNetAddr := accessTestAccount(t, "scheme")
	mainNetAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	require.NoError(t, err)
	snapshot := txSnapshot{regular: []proto.AtomicSnapshot{
		&proto.AccountScriptSnapshot{SenderPublicKey: pk, Script: proto.Script{1}},
		&proto.GenerationCommitmentSnapshot{SenderPublicKey: pk},
	}}
	for _, test := range []struct {
		scheme proto.Scheme
		addr   proto.WavesAddress
	}{
		{proto.TestNetScheme, testNetAddr},
		{proto.MainNetScheme, mainNetAddr},
	} {
		set, wsErr := writeSetFromSnapshot(nil, snapshot, test.scheme)
		require.NoError(t, wsErr)
		assert.Equal(t,
			newAccessSet(accountScriptAccessKey(test.addr.ID()), accountBalanceAccessKey(test.addr.ID())).sorted(),
			set.sorted(),
		)
	}
}
