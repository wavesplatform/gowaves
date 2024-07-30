package state

import (
	"strconv"
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
	balances, err := newBalances(stor.db, stor.hs, stor.entities.assets, stor.settings, true)
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
	leaseBalanceSnapshots, overflows, err := to.balances.generateLeaseBalanceSnapshotsForLeaseOverflows()
	assert.NoError(t, err, "generateLeaseBalanceSnapshotsForLeaseOverflows() failed")
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

func generateBlocksWithIncreasingBalance(
	t *testing.T,
	to *balancesTestObjects,
	blocksCount int,
	addressIDs ...proto.AddressID,
) {
	for i := 1; i <= blocksCount; i++ {
		blockID := genBlockId(byte(i))
		to.stor.addBlock(t, blockID)
		p := balanceProfile{uint64(i), 0, 0}
		for _, addr := range addressIDs {
			err := to.balances.setWavesBalance(addr, newWavesValueFromProfile(p), blockID)
			require.NoError(t, err, "setWavesBalance() failed")
		}
	}
}

func TestMinBalanceInRange(t *testing.T) {
	to := createBalances(t)

	addr, err := proto.NewAddressFromString(addr0)
	require.NoError(t, err, "NewAddressFromString() failed")
	generateBlocksWithIncreasingBalance(t, to, totalBlocksNumber, addr.ID())
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

func TestBalancesChangesByStoredChallenge(t *testing.T) {
	to := createBalances(t)

	challenged, err := proto.NewAddressFromString(addr0)
	require.NoError(t, err, "NewAddressFromString() failed")
	challengedID := challenged.ID()
	challenger, err := proto.NewAddressFromString(addr1)
	require.NoError(t, err, "NewAddressFromString() failed")
	challengerID := challenger.ID()
	generateBlocksWithIncreasingBalance(t, to, totalBlocksNumber, challengedID, challengerID)

	const (
		challengeHeight1 = totalBlocksNumber / 4
		challengeHeight2 = totalBlocksNumber / 2
	)
	err = to.balances.storeChallenge(challengerID, challengedID, challengeHeight1, genBlockId(challengeHeight1))
	require.NoError(t, err)
	err = to.balances.storeChallenge(challengerID, challengedID, challengeHeight2, genBlockId(challengeHeight2))
	require.NoError(t, err)

	to.stor.flush(t)

	const (
		firstBlock                = 1
		lastBlockBeforeChallenge1 = challengeHeight1 - 1
		firstBlockAfterChallenge1 = challengeHeight1 + 1
		lastBlockBeforeChallenge2 = challengeHeight2 - 1
		firstBlockAfterChallenge2 = challengeHeight2 + 1
	)
	t.Run("minEffectiveBalanceInRange", func(t *testing.T) {
		tests := []struct {
			startHeight     proto.Height
			endHeight       proto.Height
			expectedBalance uint64
			addr            proto.AddressID
		}{
			{firstBlock, totalBlocksNumber, 0, challengedID},
			// first challenge tests
			{challengeHeight1, challengeHeight1, 0, challengedID},
			{firstBlock, lastBlockBeforeChallenge1, 1, challengedID},
			{firstBlock, challengeHeight1, 0, challengedID},
			{firstBlock, firstBlockAfterChallenge1, 0, challengedID},
			{lastBlockBeforeChallenge1, firstBlockAfterChallenge1, 0, challengedID},
			{challengeHeight1, lastBlockBeforeChallenge2, 0, challengedID},
			{firstBlockAfterChallenge1, lastBlockBeforeChallenge2, firstBlockAfterChallenge1, challengedID},
			{lastBlockBeforeChallenge1, totalBlocksNumber, 0, challengedID}, // challenges 1 and 2 included
			// second challenge tests
			{challengeHeight2, challengeHeight2, 0, challengedID},
			{firstBlockAfterChallenge1, totalBlocksNumber, 0, challengedID}, // challenge 2 included
			{firstBlockAfterChallenge1, challengeHeight2, 0, challengedID},
			{lastBlockBeforeChallenge2, firstBlockAfterChallenge2, 0, challengedID},
			{challengeHeight2, totalBlocksNumber, 0, challengedID},
			{firstBlockAfterChallenge2, totalBlocksNumber, firstBlockAfterChallenge2, challengedID},
			// challenger tests
			{firstBlock, totalBlocksNumber, 1, challengerID},
			{challengeHeight1, totalBlocksNumber, challengeHeight1, challengerID},
			{challengeHeight2, totalBlocksNumber, challengeHeight2, challengerID},
			{firstBlock, challengeHeight1, 1, challengerID},
			{firstBlockAfterChallenge1, challengeHeight2, firstBlockAfterChallenge1, challengerID},
			{challengeHeight1, challengeHeight1, challengeHeight1, challengerID}, // should be without bonus
			{challengeHeight2, challengeHeight2, challengeHeight2, challengerID}, // should be without bonus
		}
		for i, tc := range tests {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				minBalance, mbErr := to.balances.minEffectiveBalanceInRange(tc.addr, tc.startHeight, tc.endHeight)
				require.NoError(t, mbErr)
				assert.Equal(t, tc.expectedBalance, minBalance)
			})
		}
	})

	t.Run("generatingBalance", func(t *testing.T) {
		const generationBalanceDepthDiff = 49
		start, end := to.stor.settings.RangeForGeneratingBalanceByHeight(100)
		require.Equal(t, uint64(generationBalanceDepthDiff), end-start) // sanity check for the next test cases

		tests := []struct {
			height   proto.Height
			addr     proto.AddressID
			expected uint64
		}{
			// FOR CHALLENGER
			{firstBlock, challengerID, 1},
			// because lastBlockBeforeChallenge1 == generationBalanceDepthDiff, so we use the lowes value in the range
			{lastBlockBeforeChallenge1, challengerID, 1},
			// challengerGenBalance + challengedGenBalance = 1 + 1
			{challengeHeight1, challengerID, 2 * (challengeHeight1 - generationBalanceDepthDiff)},
			{firstBlockAfterChallenge1, challengerID, firstBlockAfterChallenge1 - generationBalanceDepthDiff},
			{lastBlockBeforeChallenge2, challengerID, lastBlockBeforeChallenge2 - generationBalanceDepthDiff},
			// challengerGenBalance + challengedGenBalance = 51 + 51
			{challengeHeight2, challengerID, 2 * (challengeHeight2 - generationBalanceDepthDiff)},
			{firstBlockAfterChallenge2, challengerID, firstBlockAfterChallenge2 - generationBalanceDepthDiff},
			{totalBlocksNumber, challengerID, totalBlocksNumber - generationBalanceDepthDiff},
			// FOR CHALLENGED
			{firstBlock, challengedID, 1},
			{lastBlockBeforeChallenge1, challengedID, 1},
			{challengeHeight1, challengedID, 0},
			{firstBlockAfterChallenge1, challengedID, 0},
			{lastBlockBeforeChallenge1 + generationBalanceDepthDiff, challengedID, 0},
			{lastBlockBeforeChallenge2, challengerID, lastBlockBeforeChallenge2 - generationBalanceDepthDiff},
			{challengeHeight2, challengedID, 0},
			{firstBlockAfterChallenge2, challengedID, 0},
			{lastBlockBeforeChallenge2 + generationBalanceDepthDiff, challengedID, 0},
			{totalBlocksNumber, challengedID, totalBlocksNumber - generationBalanceDepthDiff},
		}
		for i, tc := range tests {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				generatingBalance, gbErr := to.balances.generatingBalance(tc.addr, tc.height)
				require.NoError(t, gbErr)
				assert.Equal(t, tc.expected, generatingBalance)
			})
		}
	})
}

func TestBalancesStoreSelfChallenge(t *testing.T) {
	to := createBalances(t)

	addr, err := proto.NewAddressFromString(addr0)
	require.NoError(t, err, "NewAddressFromString() failed")
	addrID := addr.ID()
	generateBlocksWithIncreasingBalance(t, to, totalBlocksNumber, addrID)

	const challengeHeight = totalBlocksNumber / 2
	err = to.balances.storeChallenge(addrID, addrID, challengeHeight, genBlockId(challengeHeight))
	require.EqualError(t, err, "challenger and challenged addresses are the same")

	to.stor.flush(t)

	err = to.balances.storeChallenge(addrID, addrID, challengeHeight, genBlockId(challengeHeight))
	require.EqualError(t, err, "challenger and challenged addresses are the same")
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
