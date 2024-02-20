package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	totalBlocksNumber = 200

	addr0 = "3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi"
	addr1 = "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7"
	addr2 = "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	addr3 = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"
)

type balancesTestObjects struct {
	stor     *testStorageObjects
	balances *balances
}

func createBalances(t *testing.T) *balancesTestObjects {
	stor := createStorageObjects(t, true)
	balances, err := newBalances(stor.db, stor.hs, stor.entities.assets, proto.MainNetScheme, true)
	require.NoError(t, err)
	return &balancesTestObjects{stor, balances}
}

func genAsset(fillWith byte) crypto.Digest {
	var asset crypto.Digest
	for i := range asset {
		asset[i] = fillWith
	}
	return asset
}

func newWavesValueFromProfile(p balanceProfile) wavesValue {
	val := wavesValue{profile: p}
	if p.leaseIn != 0 || p.leaseOut != 0 {
		val.leaseChange = true
	}
	if p.balance != 0 {
		val.balanceChange = true
	}
	return val
}

func TestCancelAllLeases(t *testing.T) {
	to := createBalances(t)

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	tests := []struct {
		addr    string
		profile balanceProfile
		blockID proto.BlockID
	}{
		{addr0, balanceProfile{100, 1, 1}, blockID0},
		{addr1, balanceProfile{2500, 2, 0}, blockID0},
		{addr2, balanceProfile{10, 0, 10}, blockID1},
		{addr3, balanceProfile{10, 5, 3}, blockID1},
	}
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		err = to.balances.setWavesBalance(addr.ID(), newWavesValueFromProfile(tc.profile), tc.blockID)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	zeroLeaseBalanceSnapshots, err := to.balances.generateZeroLeaseBalanceSnapshotsForAllLeases()
	assert.NoError(t, err, "generateZeroLeaseBalanceSnapshotsForAllLeases() failed")
	to.stor.flush(t)

	expected := make(map[proto.WavesAddress]proto.LeaseBalanceSnapshot, len(zeroLeaseBalanceSnapshots))
	for _, s := range zeroLeaseBalanceSnapshots {
		expected[s.Address] = s
	}
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := to.balances.wavesBalance(addr.ID())
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, tc.profile.balance, profile.balance)
		assert.Equal(t, tc.profile.leaseIn, profile.leaseIn)
		assert.Equal(t, tc.profile.leaseOut, profile.leaseOut)
		// check that lease balance snapshot is zero and is included in the list
		s, ok := expected[addr]
		assert.True(t, ok, "did not find lease balance snapshot")
		assert.Equal(t, addr, s.Address)
		assert.Equal(t, uint64(0), s.LeaseIn)
		assert.Equal(t, uint64(0), s.LeaseOut)
	}
}

func TestCancelLeaseOverflows(t *testing.T) {
	to := createBalances(t)

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	tests := []struct {
		addr    string
		profile balanceProfile
		blockID proto.BlockID
	}{
		{addr0, balanceProfile{100, 0, 1}, blockID0},
		{addr1, balanceProfile{2500, 2, 0}, blockID0},
		{addr2, balanceProfile{10, 1, 11}, blockID1},
		{addr3, balanceProfile{10, 5, 2000}, blockID1},
	}
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		err = to.balances.setWavesBalance(addr.ID(), newWavesValueFromProfile(tc.profile), tc.blockID)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	leaseBalanceSnapshots, overflows, err := to.balances.generateLeaseBalanceSnapthosForLeaseOverflows()
	assert.NoError(t, err, "generateLeaseBalanceSnapthosForLeaseOverflows() failed")
	to.stor.flush(t)

	expected := make(map[proto.WavesAddress]proto.LeaseBalanceSnapshot, len(leaseBalanceSnapshots))
	for _, lb := range leaseBalanceSnapshots {
		expected[lb.Address] = lb
	}
	overflowsCount := 0
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := to.balances.wavesBalance(addr.ID())
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.profile.balance)
		assert.Equal(t, profile.leaseIn, tc.profile.leaseIn)
		// profile.leaseOut should not be changed because we've just generated lease balance snapshot
		assert.Equal(t, tc.profile.leaseOut, profile.leaseOut)
		if uint64(tc.profile.leaseOut) > tc.profile.balance {
			assert.Contains(t, overflows, addr, "did not include overflowed address to the list")
			overflowsCount++
			snap, ok := expected[addr]
			assert.True(t, ok, "did not find lease balance snapshot")
			assert.Equal(t, addr, snap.Address)
			assert.Equal(t, uint64(0), snap.LeaseOut)
			assert.Equal(t, uint64(tc.profile.leaseIn), snap.LeaseIn)
		}
	}
	assert.Equal(t, len(overflows), overflowsCount)
}

func TestCancelInvalidLeaseIns(t *testing.T) {
	to := createBalances(t)

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	tests := []struct {
		addr         string
		profile      balanceProfile
		blockID      proto.BlockID
		validLeaseIn int64
	}{
		{addr0, balanceProfile{100, 0, 0}, blockID0, 1},
		{addr1, balanceProfile{2500, 2, 0}, blockID0, 3},
		{addr2, balanceProfile{10, 1, 0}, blockID1, 1},
		{addr3, balanceProfile{10, 5, 0}, blockID1, 0},
	}
	leaseIns := make(map[proto.WavesAddress]int64)
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		err = to.balances.setWavesBalance(addr.ID(), newWavesValueFromProfile(tc.profile), tc.blockID)
		assert.NoError(t, err, "setWavesBalance() failed")
		leaseIns[addr] = tc.validLeaseIn
	}
	leaseBalanceSnapshots, err := to.balances.generateCorrectingLeaseBalanceSnapshotsForInvalidLeaseIns(leaseIns)
	assert.NoError(t, err, "generateCorrectingLeaseBalanceSnapshotsForInvalidLeaseIns() failed")
	to.stor.flush(t)

	expected := make(map[proto.WavesAddress]proto.LeaseBalanceSnapshot, len(leaseBalanceSnapshots))
	for _, lb := range leaseBalanceSnapshots {
		expected[lb.Address] = lb
	}
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := to.balances.wavesBalance(addr.ID())
		assert.NoError(t, err, "wavesBalance() failed")

		assert.Equal(t, tc.profile.balance, profile.balance)
		assert.Equal(t, tc.profile.leaseIn, profile.leaseIn) // should not be changed
		assert.Equal(t, tc.profile.leaseOut, profile.leaseOut)

		if tc.validLeaseIn == tc.profile.leaseIn {
			assert.NotContains(t, expected, addr, "should not include address to the list")
		} else {
			snap, ok := expected[addr]
			assert.True(t, ok, "did not find lease balance snapshot")
			assert.Equal(t, addr, snap.Address)
			assert.Equal(t, uint64(tc.validLeaseIn), snap.LeaseIn)
			assert.Equal(t, uint64(tc.profile.leaseOut), snap.LeaseOut)
		}
	}
}

func TestMinBalanceInRange(t *testing.T) {
	to := createBalances(t)

	addr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	for i := 1; i <= totalBlocksNumber; i++ {
		blockID := genBlockId(byte(i))
		to.stor.addBlock(t, blockID)
		p := balanceProfile{uint64(i), 0, 0}
		if err := to.balances.setWavesBalance(addr.ID(), newWavesValueFromProfile(p), blockID); err != nil {
			t.Fatalf("Faied to set waves balance: %v\n", err)
		}
	}
	to.stor.flush(t)
	minBalance, err := to.balances.minEffectiveBalanceInRange(addr.ID(), 1, totalBlocksNumber)
	if err != nil {
		t.Fatalf("minEffectiveBalanceInRange(): %v\n", err)
	}
	if minBalance != 1 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 1, minBalance)
	}
	minBalance, err = to.balances.minEffectiveBalanceInRange(addr.ID(), 99, 150)
	if err != nil {
		t.Fatalf("minEffectiveBalanceInRange(): %v\n", err)
	}
	if minBalance != 99 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 99, minBalance)
	}
}

func addTailInfoToAssetsState(a *assets, fullAssetID crypto.Digest) {
	// see to.balances.setAssetBalance function details for more info
	shortAssetID := proto.AssetIDFromDigest(fullAssetID)
	// add digest tail info for correct state hash calculation
	wrappedAssetInfo := wrappedUncertainInfo{
		assetInfo:     assetInfo{assetConstInfo: assetConstInfo{Tail: proto.DigestTail(fullAssetID)}},
		wasJustIssued: false,
	}
	a.uncertainAssetInfo[shortAssetID] = wrappedAssetInfo
}

func TestBalances(t *testing.T) {
	to := createBalances(t)

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	wavesTests := []struct {
		addr    string
		profile balanceProfile
		blockID proto.BlockID
	}{
		{addr0, balanceProfile{100, 0, 0}, blockID0},
		{addr1, balanceProfile{2500, 0, 0}, blockID0},
		{addr2, balanceProfile{10, 5, 0}, blockID1},
		{addr3, balanceProfile{10, 5, 3}, blockID1},
	}
	for _, tc := range wavesTests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		if err := to.balances.setWavesBalance(addr.ID(), newWavesValueFromProfile(tc.profile), tc.blockID); err != nil {
			t.Fatalf("Faied to set waves balance:%v\n", err)
		}
		to.stor.flush(t)
		profile, err := to.balances.wavesBalance(addr.ID())
		if err != nil {
			t.Fatalf("Failed to retrieve waves balance: %v\n", err)
		}
		if profile != tc.profile {
			t.Errorf("Waves balance profiles are not equal: %v and %v\n", profile, tc.profile)
		}
	}

	assetTests := []struct {
		addr    string
		assetID crypto.Digest
		balance uint64
		blockID proto.BlockID
	}{
		{addr0, genAsset(1), 100, blockID0},
		{addr0, genAsset(1), 2500, blockID0},
		{addr0, genAsset(1), 10, blockID1},
	}
	for _, tc := range assetTests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		addTailInfoToAssetsState(to.stor.entities.assets, tc.assetID)
		if err := to.balances.setAssetBalance(addr.ID(), proto.AssetIDFromDigest(tc.assetID), tc.balance, tc.blockID); err != nil {
			t.Fatalf("Faied to set asset balance: %v\n", err)
		}
		to.stor.flush(t)
		balance, err := to.balances.assetBalance(addr.ID(), proto.AssetIDFromDigest(tc.assetID))
		if err != nil {
			t.Fatalf("Failed to retrieve asset balance: %v\n", err)
		}
		if balance != tc.balance {
			t.Errorf("Asset balances are not equal: %d and %d\n", balance, tc.balance)
		}
	}
}

func TestNftList(t *testing.T) {
	to := createBalances(t)

	// see to.balances.setAssetBalance function details for more info
	assetIDBytes := testGlobal.asset1.assetID
	addTailInfoToAssetsState(to.stor.entities.assets, assetIDBytes)

	to.stor.addBlock(t, blockID0)

	addr := testGlobal.senderInfo.addr
	assetID := testGlobal.asset1.asset.ID
	err := to.balances.setAssetBalance(addr.ID(), proto.AssetIDFromDigest(assetIDBytes), 123, blockID0)
	assert.NoError(t, err)
	asset := defaultNFT(proto.DigestTail(assetID))
	err = to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(assetID), asset, blockID0)
	assert.NoError(t, err)
	to.stor.flush(t)

	var (
		height = to.stor.rw.recentHeight()
		feats  = to.stor.entities.features
	)
	nfts, err := to.balances.nftList(addr.ID(), 1, nil, height, feats)
	assert.NoError(t, err)
	assert.Equal(t, []crypto.Digest{assetID}, nfts)

	to.stor.activateFeature(t, int16(settings.ReducedNFTFee))

	nfts, err = to.balances.nftList(addr.ID(), 1, nil, height, feats)
	assert.NoError(t, err)
	assert.Equal(t, []crypto.Digest{assetID}, nfts)

	to.stor.activateFeature(t, int16(settings.BlockV5))

	nfts, err = to.balances.nftList(addr.ID(), 1, nil, height, feats)
	assert.NoError(t, err)
	assert.Equal(t, []crypto.Digest(nil), nfts)

}
