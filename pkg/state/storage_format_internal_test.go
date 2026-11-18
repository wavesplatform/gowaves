package state

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

// updateStorageFormat adds the samples of new storage records to the golden file of the current StateVersion.
// Run `go test ./pkg/state -run TestStorageFormat -update-storage-format` after adding a new kind of record or
// after the increment of StateVersion. The change of the format of existing records is not accepted, the version
// must be incremented and the migration must be added, see migration.go.
var updateStorageFormat = flag.Bool("update-storage-format", false, //nolint:gochecknoglobals // test flag.
	"add new storage format samples to the golden file of the current state version")

type storageSample struct {
	name string
	data func() ([]byte, error)
}

// entitySamplePrefix is the prefix of names of samples stored in the golden file as text, others are stored in hex.
const entitySamplePrefix = "entity/"

func noErr(b []byte) func() ([]byte, error) { return func() ([]byte, error) { return b, nil } }

func sampleAddressID(b byte) proto.AddressID {
	var id proto.AddressID
	for i := range id {
		id[i] = b + byte(i)
	}
	return id
}

func sampleAssetID(b byte) proto.AssetID {
	var id proto.AssetID
	for i := range id {
		id[i] = b + byte(i)
	}
	return id
}

func sampleBlockID() proto.BlockID { return proto.NewBlockIDFromDigest(testDigest(0x42)) }

func sampleFieldsHashes() proto.FieldsHashesV1 {
	return proto.FieldsHashesV1{
		WavesBalanceHash: testDigest(1), AssetBalanceHash: testDigest(2), DataEntryHash: testDigest(3),
		AccountScriptHash: testDigest(4), AssetScriptHash: testDigest(5), LeaseBalanceHash: testDigest(6),
		LeaseStatusHash: testDigest(7), SponsorshipHash: testDigest(8), AliasesHash: testDigest(9),
	}
}

func storageKeySamples() []storageSample {
	addr := sampleAddressID(1)
	asset := sampleAssetID(2)
	blockID := sampleBlockID()
	digest := testDigest(3)
	feature := func(k interface{ bytes() ([]byte, error) }) func() ([]byte, error) { return k.bytes }
	return []storageSample{
		{"key/stateInfo", noErr(stateInfoKeyBytes)},
		{"key/rollbackMinHeight", noErr(rollbackMinHeightKeyBytes)},
		{"key/lastBlockNum", noErr(lastBlockNumKeyBytes)},
		{"key/dbHeight", noErr(dbHeightKeyBytes)},
		{"key/txsByAddressesFileSize", noErr(fileSizeKeyBytes)},
		{"key/lastAccountsStorAddrNum", noErr([]byte{lastAccountsStorAddrNumKeyPrefix})},
		{"key/rewardChanges", noErr([]byte{rewardChangesKeyPrefix})},
		{"key/rwProtobufInfo", noErr([]byte{rwProtobufInfoKeyPrefix})},
		{"key/finalization", noErr([]byte{finalizationKeyPrefix})},
		{"key/migrationProgress", noErr([]byte{migrationProgressKeyPrefix})},
		{"key/wavesBalance", noErr((&wavesBalanceKey{address: addr}).bytes())},
		{"key/assetBalance", noErr((&assetBalanceKey{address: addr, asset: asset}).bytes())},
		{"key/blockIdToNum", noErr((&blockIdToNumKey{blockID: blockID}).bytes())},
		{"key/blockNumToId", noErr((&blockNumToIdKey{blockNum: 0x01020304}).bytes())},
		{"key/validBlockNum", noErr((&validBlockNumKey{blockNum: 0x01020304}).bytes())},
		{"key/blockOffset", noErr((&blockOffsetKey{blockID: blockID}).bytes())},
		{"key/txInfo", noErr((&txInfoKey{txID: digest.Bytes()}).bytes())},
		{"key/score", noErr((&scoreKey{height: 0x0102030405}).bytes())},
		{"key/assetConst", noErr((&assetConstKey{assetID: asset}).bytes())},
		{"key/assetHist", noErr((&assetHistKey{assetID: asset}).bytes())},
		{"key/lease", noErr((&leaseKey{leaseID: digest}).bytes())},
		{"key/alias", noErr((&aliasKey{alias: "alias"}).bytes())},
		{"key/addressToAliases", noErr((&addressToAliasesKey{addressID: addr}).bytes())},
		{"key/disabledAlias", noErr((&disabledAliasKey{alias: "alias"}).bytes())},
		{"key/activatedFeatures", feature(&activatedFeaturesKey{featureID: 25})},
		{"key/approvedFeatures", feature(&approvedFeaturesKey{featureID: 25})},
		{"key/votesFeatures", feature(&votesFeaturesKey{featureID: 25})},
		{"key/ordersVolume", noErr((&ordersVolumeKey{orderID: digest.Bytes()}).bytes())},
		{"key/blocksInfo", noErr((&blocksInfoKey{blockID: blockID}).bytes())},
		{"key/accountStorAddrToNum", noErr((&accountStorAddrToNumKey{addressID: addr}).bytes())},
		{"key/accountsDataStor", noErr((&accountsDataStorKey{addrNum: 0x0102, entryKey: "key"}).bytes())},
		{"key/sponsorship", noErr((&sponsorshipKey{assetID: asset}).bytes())},
		{"key/accountScript", noErr((&accountScriptKey{addr: addr}).bytes())},
		{"key/assetScript", noErr((&assetScriptKey{assetID: asset}).bytes())},
		{"key/scriptBasicInfo/account",
			noErr((&scriptBasicInfoKey{scriptKey: &accountScriptKey{addr: addr}}).bytes())},
		{"key/scriptBasicInfo/asset", noErr((&scriptBasicInfoKey{scriptKey: &assetScriptKey{assetID: asset}}).bytes())},
		{"key/accountScriptComplexity", noErr((&accountScriptComplexityKey{addressID: addr}).bytes())},
		{"key/assetScriptComplexity", noErr((&assetScriptComplexityKey{asset: asset}).bytes())},
		{"key/batchedStor",
			noErr((&batchedStorKey{prefix: 1, internalKey: []byte{2, 3}, batchNum: 0x01020304}).bytes())},
		{"key/lastBatch", noErr((&lastBatchKey{prefix: 1, internalKey: []byte{2, 3}}).bytes())},
		{"key/invokeResult", noErr((&invokeResultKey{invokeID: digest}).bytes())},
		{"key/legacyStateHash", noErr((&legacyStateHashKey{height: 0x0102030405}).bytes())},
		{"key/snapshotStateHash", noErr((&snapshotStateHashKey{height: 0x0102030405}).bytes())},
		{"key/hitSource", noErr((&hitSourceKey{height: 0x0102030405}).bytes())},
		{"key/snapshots", noErr((&snapshotsKey{height: 0x0102030405}).bytes())},
		{"key/patch", noErr((&patchKey{blockID: blockID}).bytes())},
		{"key/rewardVotes", noErr((&rewardVotesKey{height: 0x0102030405}).bytes())},
		{"key/challengedAddress", noErr((&challengedAddressKey{address: addr}).bytes())},
		{"key/commitment", noErr((&commitmentKey{periodStart: 0x01020304}).bytes())},
		{"key/generators", noErr((&generatorsKey{height: 0x0102030405}).bytes())},
	}
}

func storageRecordSamples(t *testing.T) []storageSample {
	addr := sampleAddressID(1)
	pk := crypto.PublicKey(testDigest(4))
	digest := testDigest(3)
	blsPK := bls.PublicKey{}
	for i := range blsPK {
		blsPK[i] = byte(i)
	}
	wavesAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pk)
	require.NoError(t, err)
	quantity := new(big.Int).Lsh(big.NewInt(1), 70)
	fees := newFeeDistribution()
	fees.totalWavesFees, fees.currentWavesBlockFees = 1000, 400
	fees.totalFees[digest], fees.currentBlockFees[digest] = 2000, 800
	shV2 := proto.StateHashV2{BlockID: sampleBlockID(), FieldsHashesV1: sampleFieldsHashes(),
		GeneratorsHash: testDigest(10), GeneratorsBalancesHash: testDigest(11)}
	shV2.SumHash = testDigest(12)
	snapshot := proto.WavesBalanceSnapshot{Address: wavesAddr, Balance: 100500}
	marshal := func(m interface{ marshalBinary() ([]byte, error) }) func() ([]byte, error) { return m.marshalBinary }
	return []storageSample{
		{"record/stateInfo", marshal(&stateInfo{Version: StateVersion, Amend: true, HasExtendedAPIData: true,
			HasStateHashes: true, DBCompressionAlgo: keyvalue.CompressionAlgo(1)})},
		{"record/migrationProgress", marshal(&migrationProgress{Version: 26, Step: 1, LastKey: []byte{1, 2}})},
		{"record/wavesBalance",
			marshal(&wavesBalanceRecord{balanceProfile{Balance: 1 << 40, LeaseIn: 2, LeaseOut: 3, Deposit: 4}})},
		{"record/assetBalance", marshal(&assetBalanceRecord{Balance: 1 << 40})},
		{"record/challengedAddress", marshal(&challengedAddressRecord{Heights: heights{1, 1 << 20}})},
		{"record/dataEntry", func() ([]byte, error) {
			return (&proto.StringDataEntry{Key: "key", Value: "value"}).MarshalValue()
		}},
		{"record/alias", marshal(&aliasRecord{info: aliasInfo{stolen: true, addressID: addr}})},
		{"record/addressToAliases", marshal(&addressToAliasesRecord{aliases: []string{"a", "bb"}})},
		{"record/assetConstInfo", marshal(&assetConstInfo{Tail: [proto.AssetIDTailSize]byte{1, 2}, Issuer: pk,
			Decimals: 8, IssueHeight: 1 << 20, IsNFT: true, IssueSequenceInBlock: 3})},
		{"record/assetHistory", marshal(&assetHistoryRecord{assetChangeableInfo{quantity: *quantity, name: "name",
			description: "description", lastNameDescChangeHeight: 1 << 20, reissuable: true}})},
		{"record/commitments", marshal(&commitmentsRecord{Commitments: []commitmentItem{
			{GeneratorPK: pk, EndorserPK: blsPK, TransactionID: digest}}})},
		{"record/activatedFeatures", marshal(&activatedFeaturesRecord{activationHeight: 1 << 20})},
		{"record/approvedFeatures", marshal(&approvedFeaturesRecord{approvalHeight: 1 << 20})},
		{"record/votesFeatures", marshal(&votesFeaturesRecord{votesNum: 1 << 20})},
		{"record/finalization", marshal(&finalizationRecord{FinalizedBlockHeight: 1 << 20, PendingBlockHeight: 2})},
		{"record/generators", marshal(&generatorsRecord{Generators: []generator{
			{Balance: 1 << 40, BanHeight: 1 << 20, AddressID: addr}}, BlockGeneratorIndex: 1, PeriodStart: 1 << 20})},
		{"record/lease", marshal(&leasing{SenderPK: pk, RecipientAddr: wavesAddr, Amount: 1 << 40,
			OriginHeight: 1 << 20, Status: LeaseCancelled, OriginTransactionID: &digest, CancelHeight: 1 << 21,
			CancelTransactionID: &digest})},
		{"record/invokeResult", marshal(&invokeResultRecord{res: &proto.ScriptResult{
			DataEntries: []*proto.DataEntryScriptAction{{Entry: &proto.IntegerDataEntry{Key: "k", Value: 1}}}}})},
		{"record/ordersVolume", marshal(&orderVolumeRecord{feeFilled: 1 << 20, amountFilled: 1 << 40})},
		{"record/sponsorship", marshal(&sponsorshipRecord{assetCost: 100500})},
		{"record/rewardVotes", marshal(&rewardVotesRecord{increase: 1, decrease: 2})},
		{"record/rewardChanges", marshal(&rewardChangesRecords{{Height: 1 << 20, Reward: 6 * proto.PriceConstant}})},
		{"record/patch", marshal(&patchRecord{fixSnapshots: []proto.AtomicSnapshot{snapshot}})},
		{"record/snapshots", func() ([]byte, error) {
			return proto.BlockSnapshot{TxSnapshots: [][]proto.AtomicSnapshot{{snapshot}}}.MarshallBinary()
		}},
		{"record/estimation", marshal(&estimationRecord{EstimatorVersion: 4, Estimation: ride.TreeEstimation{
			Estimation: 100, Verifier: 10, Functions: map[string]int{"f": 50}}})},
		{"record/scriptBasicInfo", marshal(&scriptBasicInfoRecord{PK: pk, ScriptLen: 100,
			LibraryVersion: ast.LibV6, HasVerifier: true, IsDApp: true})},
		{"record/protobufInfo", noErr((&protobufInfo{protobufTxStart: 1, protobufHeadersStart: 2,
			protobufAfterHeight: 3}).marshalBinary())},
		{"record/feeDistribution", noErr(fees.marshalBinary())},
		{"record/batchedStorage", noErr((&record{blockNum: 0x01020304, data: []byte{5, 6}}).marshalBinary())},
		{"record/legacyStateHashV1", (&proto.StateHashV1{BlockID: sampleBlockID(), SumHash: testDigest(12),
			FieldsHashesV1: sampleFieldsHashes()}).MarshalBinary},
		{"record/legacyStateHashV2", shV2.MarshalBinary},
		{"history/fixedSize", func() ([]byte, error) {
			hr := newHistoryRecord(activatedFeature)
			hr.entries = []historyEntry{{data: []byte{1, 2, 3, 4, 5, 6, 7, 8}, blockNum: 0x01020304}}
			return hr.marshalBinary()
		}},
		{"history/variableSize", func() ([]byte, error) {
			hr := newHistoryRecord(wavesBalance)
			hr.entries = []historyEntry{{data: []byte{1, 2}, blockNum: 1}, {data: []byte{3}, blockNum: 2}}
			return hr.marshalBinary()
		}},
	}
}

// storageEntitySamples pins the values of entity types and their properties, which are stored in history records.
func storageEntitySamples() []storageSample {
	res := make([]storageSample, 0, len(properties))
	for e, p := range properties {
		res = append(res, storageSample{
			name: fmt.Sprintf("%s%d", entitySamplePrefix, e),
			data: noErr(fmt.Appendf(nil, "%+v", p)),
		})
	}
	return res
}

func storageFormatGoldenPath() string {
	return filepath.Join("testdata", "storage_format", fmt.Sprintf("v%d.json", StateVersion))
}

func readStorageFormatGolden(t *testing.T) map[string]string {
	data, err := os.ReadFile(storageFormatGoldenPath())
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}
	}
	require.NoError(t, err)
	res := make(map[string]string)
	require.NoError(t, json.Unmarshal(data, &res))
	return res
}

// TestStorageFormat checks that the binary formats of keys and records stored in the state database are not changed
// without the increment of StateVersion. If the test fails, increment StateVersion, add the migration from the previous
// version (see migration.go) and run the test with -update-storage-format flag to create the golden file of the new
// version.
func TestStorageFormat(t *testing.T) {
	samples := slices.Concat(storageKeySamples(), storageRecordSamples(t), storageEntitySamples())
	golden := readStorageFormatGolden(t)
	actual := make(map[string]string, len(samples))
	for _, s := range samples {
		data, err := s.data()
		require.NoError(t, err, s.name)
		require.NotContains(t, actual, s.name, "duplicate sample name")
		if strings.HasPrefix(s.name, entitySamplePrefix) {
			actual[s.name] = string(data)
		} else {
			actual[s.name] = hex.EncodeToString(data)
		}
	}
	updated := false
	for name, value := range actual {
		expected, ok := golden[name]
		if !ok {
			if *updateStorageFormat {
				golden[name] = value
				updated = true
				continue
			}
			assert.Failf(t, "no storage format sample", "sample %q is missing in %s, run the test with "+
				"-update-storage-format flag", name, storageFormatGoldenPath())
			continue
		}
		assert.Equalf(t, expected, value, "storage format of %q is changed: increment StateVersion and add "+
			"the migration, see migration.go", name)
	}
	for name := range golden {
		_, ok := actual[name]
		assert.Truef(t, ok, "sample %q of storage format is removed from the test", name)
	}
	if updated && !t.Failed() {
		data, err := json.MarshalIndent(golden, "", "\t")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(storageFormatGoldenPath(), append(data, '\n'), 0600))
	}
}
