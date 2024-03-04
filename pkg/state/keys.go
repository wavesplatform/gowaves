package state

// keys.go - database keys.

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// Key sizes.
	minAccountsDataStorKeySize = 1 + 8 + 2 + 1

	wavesBalanceKeySize      = 1 + proto.AddressIDSize
	assetBalanceKeySize      = 1 + proto.AddressIDSize + proto.AssetIDSize
	leaseKeySize             = 1 + crypto.DigestSize
	aliasKeySize             = 1 + 2 + proto.AliasMaxLength
	addressToAliasesKeySize  = 1 + proto.AddressIDSize
	disabledAliasKeySize     = 1 + 2 + proto.AliasMaxLength
	approvedFeaturesKeySize  = 1 + 2
	votesFeaturesKeySize     = 1 + 2
	invokeResultKeySize      = 1 + crypto.DigestSize
	legacyStateHashKeySize   = 1 + 8
	snapshotStateHashKeySize = 1 + 8
	snapshotKeySize          = 1 + 8
)

// Primary prefixes for storage keys
const (
	_ byte = iota // this is a placeholder for a zero value
	// Balances.
	wavesBalanceKeyPrefix
	assetBalanceKeyPrefix

	// Unique block num of the last block.
	lastBlockNumKeyPrefix
	// BlockID --> unique block number.
	// Numbers are increasing sequentially.
	// These numbers are stored in history records instead of long IDs.
	blockIdToNumKeyPrefix
	blockNumToIdKeyPrefix
	// Valid block unique nums.
	validBlockNumKeyPrefix

	// For block storage.
	// IDs of blocks --> offsets in files.
	blockOffsetKeyPrefix
	// IDs of transactions --> offsets in files, heights, failure status.
	txInfoKeyPrefix

	// Minimum height to which rollback is possible.
	rollbackMinHeightKeyPrefix
	// Height of main db.
	dbHeightKeyPrefix

	// Score at height.
	scoreKeyPrefix
	// Assets.
	assetConstKeyPrefix
	assetHistKeyPrefix

	// Leases.
	leaseKeyPrefix

	// Aliases.
	aliasKeyPrefix
	addressToAliasesPrefix
	disabledAliasKeyPrefix

	// Features.
	activatedFeaturesKeyPrefix
	approvedFeaturesKeyPrefix
	votesFeaturesKeyPrefix

	// Orders volume.
	ordersVolumeKeyPrefix

	// Blocks information (fees for now).
	blocksInfoKeyPrefix

	// Unique address number by address.
	// These numbers are only used for accounts data storage.
	lastAccountsStorAddrNumKeyPrefix
	accountStorAddrToNumKeyPrefix
	// Prefix for keys of accounts data entries.
	accountsDataStorKeyPrefix

	// Sponsored assets storage.
	sponsorshipKeyPrefix

	// Scripts.
	accountScriptKeyPrefix
	assetScriptKeyPrefix
	scriptBasicInfoKeyPrefix
	accountScriptComplexityKeyPrefix
	assetScriptComplexityKeyPrefix

	// Block Reward.
	rewardVotesKeyPrefix
	rewardChangesKeyPrefix

	// Batched storage (see batched_storage.go).
	batchedStorKeyPrefix
	// The last batch num by internal key (batched_storage.go).
	lastBatchKeyPrefix

	// Invoke results.
	invokeResultKeyPrefix

	// Information about state: version, API support flag, ...
	stateInfoKeyPrefix

	// Size of TransactionsByAddresses file.
	txsByAddressesFileSizeKeyPrefix

	// Stores protobuf-related info for blockReadWriter.
	rwProtobufInfoKeyPrefix

	// Stores state hashes at height.
	legacyStateHashKeyPrefix
	snapshotStateHashKeyPrefix

	// Hit source data.
	hitSourceKeyPrefix
	// Block snapshots storage.
	snapshotsKeyPrefix
	// Blockchain patches storage.
	patchKeyPrefix
)

var (
	errInvalidDataSize = errors.New("invalid data size")
	errInvalidPrefix   = errors.New("invalid prefix for given key")
)

func prefixByEntity(entity blockchainEntity) ([]byte, error) {
	switch entity {
	case alias:
		return []byte{aliasKeyPrefix}, nil
	case addressToAliases:
		return []byte{addressToAliasesPrefix}, nil
	case asset:
		return []byte{assetHistKeyPrefix}, nil
	case lease:
		return []byte{leaseKeyPrefix}, nil
	case wavesBalance:
		return []byte{wavesBalanceKeyPrefix}, nil
	case assetBalance:
		return []byte{assetBalanceKeyPrefix}, nil
	case featureVote:
		return []byte{votesFeaturesKeyPrefix}, nil
	case approvedFeature:
		return []byte{approvedFeaturesKeyPrefix}, nil
	case activatedFeature:
		return []byte{activatedFeaturesKeyPrefix}, nil
	case ordersVolume:
		return []byte{ordersVolumeKeyPrefix}, nil
	case sponsorship:
		return []byte{sponsorshipKeyPrefix}, nil
	case dataEntry:
		return []byte{accountsDataStorKeyPrefix}, nil
	case accountScript:
		return []byte{accountScriptKeyPrefix}, nil
	case assetScript:
		return []byte{assetScriptKeyPrefix}, nil
	case scriptBasicInfo:
		return []byte{scriptBasicInfoKeyPrefix}, nil
	case accountScriptComplexity:
		return []byte{accountScriptComplexityKeyPrefix}, nil
	case assetScriptComplexity:
		return []byte{assetScriptComplexityKeyPrefix}, nil
	case rewardVotes:
		return []byte{rewardVotesKeyPrefix}, nil
	case invokeResult:
		return []byte{invokeResultKeyPrefix}, nil
	case score:
		return []byte{scoreKeyPrefix}, nil
	case legacyStateHash:
		return []byte{legacyStateHashKeyPrefix}, nil
	case snapshotStateHash:
		return []byte{snapshotStateHashKeyPrefix}, nil
	case hitSource:
		return []byte{hitSourceKeyPrefix}, nil
	case feeDistr:
		return []byte{blocksInfoKeyPrefix}, nil
	case rewardChanges:
		return []byte{rewardChangesKeyPrefix}, nil
	case snapshots:
		return []byte{snapshotsKeyPrefix}, nil
	case patches:
		return []byte{patchKeyPrefix}, nil
	default:
		return nil, errors.New("bad entity type")
	}
}

type wavesBalanceKey struct {
	address proto.AddressID
}

func (k *wavesBalanceKey) bytes() []byte {
	buf := make([]byte, wavesBalanceKeySize)
	buf[0] = wavesBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	return buf
}

func (k *wavesBalanceKey) unmarshal(data []byte) error {
	if len(data) != wavesBalanceKeySize {
		return errInvalidDataSize
	}
	if data[0] != wavesBalanceKeyPrefix {
		return errInvalidPrefix
	}
	copy(k.address[:], data[1:1+proto.AddressIDSize])
	return nil
}

type assetBalanceKey struct {
	address proto.AddressID
	asset   proto.AssetID
}

func (k *assetBalanceKey) addressPrefix() []byte {
	buf := make([]byte, 1+proto.AddressIDSize)
	buf[0] = assetBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	return buf
}

func (k *assetBalanceKey) bytes() []byte {
	buf := make([]byte, assetBalanceKeySize)
	buf[0] = assetBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.AddressIDSize:], k.asset[:])
	return buf
}

func (k *assetBalanceKey) unmarshal(data []byte) error {
	if len(data) != assetBalanceKeySize {
		return errInvalidDataSize
	}
	if data[0] != assetBalanceKeyPrefix {
		return errInvalidPrefix
	}
	copy(k.address[:], data[1:1+proto.AddressIDSize])
	copy(k.asset[:], data[1+proto.AddressIDSize:])
	return nil
}

type blockIdToNumKey struct {
	blockID proto.BlockID
}

func (k *blockIdToNumKey) bytes() []byte {
	idBytes := k.blockID.Bytes()
	buf := make([]byte, 1+len(idBytes))
	buf[0] = blockIdToNumKeyPrefix
	copy(buf[1:], idBytes)
	return buf
}

type blockNumToIdKey struct {
	blockNum uint32
}

func (k *blockNumToIdKey) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = blockNumToIdKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.blockNum)
	return buf
}

type validBlockNumKey struct {
	blockNum uint32
}

func (k *validBlockNumKey) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = validBlockNumKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.blockNum)
	return buf
}

type blockOffsetKey struct {
	blockID proto.BlockID
}

func (k *blockOffsetKey) bytes() []byte {
	idBytes := k.blockID.Bytes()
	buf := make([]byte, 1+len(idBytes))
	buf[0] = blockOffsetKeyPrefix
	copy(buf[1:], idBytes)
	return buf
}

type txInfoKey struct {
	txID []byte
}

func (k *txInfoKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = txInfoKeyPrefix
	copy(buf[1:], k.txID)
	return buf
}

type scoreKey struct {
	height uint64
}

func (k *scoreKey) bytes() []byte {
	buf := make([]byte, 9)
	buf[0] = scoreKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type assetConstKey struct {
	assetID proto.AssetID
}

func (k *assetConstKey) bytes() []byte {
	buf := make([]byte, 1+proto.AssetIDSize)
	buf[0] = assetConstKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type assetHistKey struct {
	assetID proto.AssetID
}

func (k *assetHistKey) bytes() []byte {
	buf := make([]byte, 1+proto.AssetIDSize)
	buf[0] = assetHistKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type leaseKey struct {
	leaseID crypto.Digest
}

func (k *leaseKey) unmarshal(data []byte) error {
	if len(data) != leaseKeySize {
		return errInvalidDataSize
	}
	if data[0] != leaseKeyPrefix {
		return errInvalidPrefix
	}
	var err error
	k.leaseID, err = crypto.NewDigestFromBytes(data[1:])
	return err
}

func (k *leaseKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = leaseKeyPrefix
	copy(buf[1:], k.leaseID[:])
	return buf
}

type aliasKey struct {
	alias string
}

func (k *aliasKey) bytes() []byte {
	buf := make([]byte, aliasKeySize)
	buf[0] = aliasKeyPrefix
	proto.PutStringWithUInt16Len(buf[1:], k.alias)
	return buf
}

func (k *aliasKey) unmarshal(data []byte) error {
	if len(data) != aliasKeySize {
		return errInvalidDataSize
	}
	if data[0] != aliasKeyPrefix {
		return errInvalidPrefix
	}
	var err error
	k.alias, err = proto.StringWithUInt16Len(data[1:])
	return err
}

type addressToAliasesKey struct {
	addressID proto.AddressID
}

func (k *addressToAliasesKey) bytes() []byte {
	buf := make([]byte, addressToAliasesKeySize)
	buf[0] = addressToAliasesPrefix
	copy(buf[1:], k.addressID[:])
	return buf
}

type disabledAliasKey struct {
	alias string
}

func (k *disabledAliasKey) bytes() []byte {
	buf := make([]byte, disabledAliasKeySize)
	buf[0] = disabledAliasKeyPrefix
	proto.PutStringWithUInt16Len(buf[1:], k.alias)
	return buf
}

type activatedFeaturesKey struct {
	featureID int16
}

func (k *activatedFeaturesKey) bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte{activatedFeaturesKeyPrefix}); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, k.featureID); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type approvedFeaturesKey struct {
	featureID int16
}

func (k *approvedFeaturesKey) bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte{approvedFeaturesKeyPrefix}); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, k.featureID); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (k *approvedFeaturesKey) unmarshal(data []byte) error {
	if len(data) != approvedFeaturesKeySize {
		return errInvalidDataSize
	}
	if data[0] != approvedFeaturesKeyPrefix {
		return errInvalidPrefix
	}
	buf := bytes.NewBuffer(data[1:])
	if err := binary.Read(buf, binary.BigEndian, &k.featureID); err != nil {
		return err
	}
	return nil
}

type votesFeaturesKey struct {
	featureID int16
}

func (k *votesFeaturesKey) bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte{votesFeaturesKeyPrefix}); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, k.featureID); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (k *votesFeaturesKey) unmarshal(data []byte) error {
	if len(data) != votesFeaturesKeySize {
		return errInvalidDataSize
	}
	if data[0] != votesFeaturesKeyPrefix {
		return errInvalidPrefix
	}
	buf := bytes.NewBuffer(data[1:])
	if err := binary.Read(buf, binary.BigEndian, &k.featureID); err != nil {
		return err
	}
	return nil
}

type ordersVolumeKey struct {
	orderID []byte
}

func (k *ordersVolumeKey) bytes() []byte {
	buf := make([]byte, 1+len(k.orderID))
	buf[0] = ordersVolumeKeyPrefix
	copy(buf[1:], k.orderID)
	return buf
}

type blocksInfoKey struct {
	blockID proto.BlockID
}

func (k *blocksInfoKey) bytes() []byte {
	idBytes := k.blockID.Bytes()
	buf := make([]byte, 1+len(idBytes))
	buf[0] = blocksInfoKeyPrefix
	copy(buf[1:], idBytes)
	return buf
}

type accountStorAddrToNumKey struct {
	addressID proto.AddressID
}

func (k *accountStorAddrToNumKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressIDSize)
	buf[0] = accountStorAddrToNumKeyPrefix
	copy(buf[1:], k.addressID[:])
	return buf
}

type accountsDataStorKey struct {
	addrNum  uint64
	entryKey string
}

func (k *accountsDataStorKey) accountPrefix() []byte {
	buf := make([]byte, 1+8)
	buf[0] = accountsDataStorKeyPrefix
	binary.BigEndian.PutUint64(buf[1:9], k.addrNum)
	return buf
}

func (k *accountsDataStorKey) bytes() []byte {
	buf := make([]byte, 1+8+2+len(k.entryKey))
	buf[0] = accountsDataStorKeyPrefix
	binary.BigEndian.PutUint64(buf[1:9], k.addrNum)
	proto.PutStringWithUInt16Len(buf[9:], k.entryKey)
	return buf
}

func (k *accountsDataStorKey) unmarshal(data []byte) error {
	if len(data) < minAccountsDataStorKeySize {
		return errInvalidDataSize
	}
	if data[0] != accountsDataStorKeyPrefix {
		return errInvalidPrefix
	}
	k.addrNum = binary.BigEndian.Uint64(data[1:9])
	var err error
	k.entryKey, err = proto.StringWithUInt16Len(data[9:])
	if err != nil {
		return errors.Wrap(err, "StringWithUInt16Len() failed")
	}
	return nil
}

type sponsorshipKey struct {
	assetID proto.AssetID
}

func (k *sponsorshipKey) bytes() []byte {
	buf := make([]byte, 1+proto.AssetIDSize)
	buf[0] = sponsorshipKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type scriptKey interface {
	scriptKeyMarker()
	bytes() []byte
}

type accountScriptKey struct {
	addr proto.AddressID
}

func (accountScriptKey) scriptKeyMarker() {}

func (k *accountScriptKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressIDSize)
	buf[0] = accountScriptKeyPrefix
	copy(buf[1:], k.addr[:])
	return buf
}

type assetScriptKey struct {
	assetID proto.AssetID
}

func (assetScriptKey) scriptKeyMarker() {}

func (k *assetScriptKey) bytes() []byte {
	buf := make([]byte, 1+proto.AssetIDSize)
	buf[0] = assetScriptKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type scriptBasicInfoKey struct {
	scriptKey scriptKey
}

func (k *scriptBasicInfoKey) bytes() []byte {
	scriptKeyBytes := k.scriptKey.bytes()
	buf := make([]byte, 1+len(scriptKeyBytes))
	buf[0] = scriptBasicInfoKeyPrefix
	copy(buf[1:], scriptKeyBytes)
	return buf
}

type accountScriptComplexityKey struct {
	addressID proto.AddressID
}

func (k *accountScriptComplexityKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressIDSize)
	buf[0] = accountScriptComplexityKeyPrefix
	copy(buf[1:], k.addressID[:])
	return buf
}

type assetScriptComplexityKey struct {
	asset proto.AssetID
}

func (k *assetScriptComplexityKey) bytes() []byte {
	buf := make([]byte, 1+proto.AssetIDSize)
	buf[0] = assetScriptComplexityKeyPrefix
	copy(buf[1:], k.asset[:])
	return buf
}

type batchedStorKey struct {
	prefix      byte
	internalKey []byte
	batchNum    uint32
}

func (k *batchedStorKey) prefixUntilBatch() []byte {
	buf := make([]byte, 2+len(k.internalKey))
	buf[0] = batchedStorKeyPrefix
	buf[1] = k.prefix
	copy(buf[2:], k.internalKey[:])
	return buf
}

func (k *batchedStorKey) bytes() []byte {
	buf := make([]byte, 2+len(k.internalKey)+4)
	buf[0] = batchedStorKeyPrefix
	buf[1] = k.prefix
	copy(buf[2:], k.internalKey[:])
	pos := 2 + len(k.internalKey)
	binary.BigEndian.PutUint32(buf[pos:], k.batchNum)
	return buf
}

type lastBatchKey struct {
	prefix      byte
	internalKey []byte
}

func (k *lastBatchKey) bytes() []byte {
	buf := make([]byte, 2+len(k.internalKey)+4)
	buf[0] = lastBatchKeyPrefix
	buf[1] = k.prefix
	copy(buf[2:], k.internalKey[:])
	return buf
}

type invokeResultKey struct {
	invokeID crypto.Digest
}

func (k *invokeResultKey) bytes() []byte {
	res := make([]byte, invokeResultKeySize)
	res[0] = invokeResultKeyPrefix
	copy(res[1:], k.invokeID[:])
	return res
}

type legacyStateHashKey struct {
	height uint64
}

func (k *legacyStateHashKey) bytes() []byte {
	buf := make([]byte, legacyStateHashKeySize)
	buf[0] = legacyStateHashKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type snapshotStateHashKey struct {
	height uint64
}

func (k *snapshotStateHashKey) bytes() []byte {
	buf := make([]byte, snapshotStateHashKeySize)
	buf[0] = snapshotStateHashKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type hitSourceKey struct {
	height uint64
}

func (k *hitSourceKey) bytes() []byte {
	buf := make([]byte, 9)
	buf[0] = hitSourceKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type snapshotsKey struct {
	height uint64
}

func (k *snapshotsKey) bytes() []byte {
	buf := make([]byte, snapshotKeySize)
	buf[0] = snapshotsKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type patchKey struct {
	blockID proto.BlockID
}

func (k *patchKey) bytes() []byte {
	idBytes := k.blockID.Bytes()
	buf := make([]byte, 1+len(idBytes))
	buf[0] = patchKeyPrefix
	copy(buf[1:], idBytes)
	return buf
}
