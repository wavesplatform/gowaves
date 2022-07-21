package state

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// maxQuantityLen is maximum length of quantity (it's represented as big.Int) bytes in asset history records.
	maxQuantityLen = 16
)

type assetInfo struct {
	assetConstInfo
	assetChangeableInfo
}

func (ai *assetInfo) isNFT() bool {
	return ai.quantity.Uint64() == 1 && ai.decimals == 0 && !ai.reissuable
}

func (ai *assetInfo) equal(ai1 *assetInfo) bool {
	return ai.assetChangeableInfo.equal(&ai1.assetChangeableInfo) && (ai.assetConstInfo == ai1.assetConstInfo)
}

// assetConstInfo is part of asset info which is constant.
type assetConstInfo struct {
	tail     [proto.AssetIDTailSize]byte
	issuer   crypto.PublicKey
	decimals int8
}

func (ai *assetConstInfo) marshalBinary() ([]byte, error) {
	res := make([]byte, proto.AssetIDTailSize+crypto.PublicKeySize+1)
	copy(res, ai.tail[:])
	if err := ai.issuer.WriteTo(res[proto.AssetIDTailSize:]); err != nil {
		return nil, err
	}
	res[proto.AssetIDTailSize+crypto.PublicKeySize] = byte(ai.decimals)
	return res, nil
}

func (ai *assetConstInfo) unmarshalBinary(data []byte) error {
	const assetConstInfoSize = proto.AssetIDTailSize + crypto.PublicKeySize + 1
	if len(data) < assetConstInfoSize {
		return errors.Errorf("invalid data size: want %d, got %d", assetConstInfoSize, len(data))
	}
	copy(ai.tail[:], data[:proto.AssetIDTailSize])
	data = data[proto.AssetIDTailSize:]
	err := ai.issuer.UnmarshalBinary(data[:crypto.PublicKeySize])
	if err != nil {
		return err
	}
	data = data[crypto.PublicKeySize:]
	ai.decimals = int8(data[0])
	return nil
}

// assetChangeableInfo is part of asset info which can change.
type assetChangeableInfo struct {
	quantity                 big.Int
	name                     string
	description              string
	lastNameDescChangeHeight uint64
	reissuable               bool
}

func (r *assetChangeableInfo) equal(r1 *assetChangeableInfo) bool {
	if r.quantity.Cmp(&r1.quantity) != 0 {
		return false
	}
	if r.reissuable != r1.reissuable {
		return false
	}
	if r.name != r1.name {
		return false
	}
	if r.description != r1.description {
		return false
	}
	if r.lastNameDescChangeHeight != r1.lastNameDescChangeHeight {
		return false
	}
	return true
}

type assetHistoryRecord struct {
	assetChangeableInfo
}

func (r *assetHistoryRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, maxQuantityLen+4+len(r.name)+len(r.description)+8+1)
	quantityBytes := r.quantity.Bytes()
	l := len(quantityBytes)
	if l > maxQuantityLen {
		return nil, errors.Errorf("quantity length %d bytes exceeds maxQuantityLen of %d", l, maxQuantityLen)
	}
	copy(res[maxQuantityLen-l:maxQuantityLen], quantityBytes)
	pos := maxQuantityLen
	proto.PutStringWithUInt16Len(res[pos:], r.name)
	pos += len(r.name) + 2
	proto.PutStringWithUInt16Len(res[pos:], r.description)
	pos += len(r.description) + 2
	binary.BigEndian.PutUint64(res[pos:], r.lastNameDescChangeHeight)
	pos += 8
	proto.PutBool(res[pos:], r.reissuable)
	return res, nil
}

func (r *assetHistoryRecord) unmarshalBinary(data []byte) error {
	r.quantity.SetBytes(data[:maxQuantityLen])
	data = data[maxQuantityLen:]
	var err error
	r.name, err = proto.StringWithUInt16Len(data)
	if err != nil {
		return err
	}
	data = data[2+len(r.name):]
	r.description, err = proto.StringWithUInt16Len(data)
	if err != nil {
		return err
	}
	data = data[2+len(r.description):]
	r.lastNameDescChangeHeight = binary.BigEndian.Uint64(data[:8])
	data = data[8:]
	r.reissuable, err = proto.Bool(data)
	return err
}

type assets struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch
	hs      *historyStorage

	freshConstInfo map[proto.AssetID]assetConstInfo

	uncertainAssetInfo map[proto.AssetID]assetInfo
}

func newAssets(db keyvalue.KeyValue, dbBatch keyvalue.Batch, hs *historyStorage) *assets {
	return &assets{
		db:                 db,
		dbBatch:            dbBatch,
		hs:                 hs,
		freshConstInfo:     make(map[proto.AssetID]assetConstInfo),
		uncertainAssetInfo: make(map[proto.AssetID]assetInfo),
	}
}

func (a *assets) addNewRecord(assetID proto.AssetID, record *assetHistoryRecord, blockID proto.BlockID) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	// Add new record to history.
	histKey := assetHistKey{assetID: assetID}
	return a.hs.addNewEntry(asset, histKey.bytes(), recordBytes, blockID)
}

func (a *assets) storeAssetInfo(assetID proto.AssetID, asset *assetInfo, blockID proto.BlockID) error {
	assetConstBytes, err := asset.assetConstInfo.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal asset const info: %v\n", err)
	}
	constKey := assetConstKey{assetID}
	a.dbBatch.Put(constKey.bytes(), assetConstBytes)
	a.freshConstInfo[assetID] = asset.assetConstInfo
	r := &assetHistoryRecord{asset.assetChangeableInfo}
	return a.addNewRecord(assetID, r, blockID)
}

func (a *assets) issueAsset(assetID proto.AssetID, asset *assetInfo, blockID proto.BlockID) error {
	return a.storeAssetInfo(assetID, asset, blockID)
}

// issueAssetUncertain() is similar to issueAsset() but the changes can be
// dropped later using dropUncertain() or committed using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) issueAssetUncertain(assetID proto.AssetID, asset *assetInfo) {
	a.uncertainAssetInfo[assetID] = *asset
}

type assetReissueChange struct {
	reissuable bool
	diff       int64
}

func (a *assets) applyReissue(assetID proto.AssetID, ch *assetReissueChange) (*assetInfo, error) {
	info, err := a.newestAssetInfo(assetID)
	if err != nil {
		return nil, errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	info.quantity.Add(&info.quantity, quantityDiff)
	info.reissuable = ch.reissuable
	return info, nil
}

func (a *assets) reissueAsset(assetID proto.AssetID, ch *assetReissueChange, blockID proto.BlockID) error {
	info, err := a.applyReissue(assetID, ch)
	if err != nil {
		return err
	}
	return a.storeAssetInfo(assetID, info, blockID)
}

// reissueAssetUncertain() is similar to reissueAsset() but the changes can be
// dropped later using dropUncertain() or committed using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) reissueAssetUncertain(assetID proto.AssetID, ch *assetReissueChange) error {
	info, err := a.applyReissue(assetID, ch)
	if err != nil {
		return err
	}
	a.uncertainAssetInfo[assetID] = *info
	return nil
}

type assetBurnChange struct {
	diff int64
}

func (a *assets) applyBurn(assetID proto.AssetID, ch *assetBurnChange) (*assetInfo, error) {
	info, err := a.newestAssetInfo(assetID)
	if err != nil {
		return nil, errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	info.quantity.Sub(&info.quantity, quantityDiff)
	return info, nil
}

func (a *assets) burnAsset(assetID proto.AssetID, ch *assetBurnChange, blockID proto.BlockID) error {
	info, err := a.applyBurn(assetID, ch)
	if err != nil {
		return err
	}
	return a.storeAssetInfo(assetID, info, blockID)
}

// burnAssetUncertain() is similar to burnAsset() but the changes can be
// dropped later using dropUncertain() or committed using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) burnAssetUncertain(assetID proto.AssetID, ch *assetBurnChange) error {
	info, err := a.applyBurn(assetID, ch)
	if err != nil {
		return err
	}
	a.uncertainAssetInfo[assetID] = *info
	return nil
}

type assetInfoChange struct {
	newName        string
	newDescription string
	newHeight      uint64
}

func (a *assets) updateAssetInfo(asset crypto.Digest, ch *assetInfoChange, blockID proto.BlockID) error {
	info, err := a.newestChangeableInfo(asset)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	info.name = ch.newName
	info.description = ch.newDescription
	info.lastNameDescChangeHeight = ch.newHeight
	record := &assetHistoryRecord{assetChangeableInfo: *info}
	return a.addNewRecord(proto.AssetIDFromDigest(asset), record, blockID)
}

func (a *assets) newestLastUpdateHeight(assetID proto.AssetID) (uint64, error) {
	assetInfo, err := a.newestAssetInfo(assetID)
	if err != nil {
		return 0, err
	}
	return assetInfo.lastNameDescChangeHeight, nil
}

func (a *assets) constInfo(assetID proto.AssetID) (*assetConstInfo, error) {
	constKey := assetConstKey{assetID: assetID}
	constInfoBytes, err := a.db.Get(constKey.bytes())
	if err != nil {
		return nil, errs.NewUnknownAsset(fmt.Sprintf("failed to retrieve const info for given asset: %v", err))
	}
	var constInfo assetConstInfo
	if err := constInfo.unmarshalBinary(constInfoBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal const info: %v\n", err)
	}
	return &constInfo, nil
}

func (a *assets) newestConstInfo(assetID proto.AssetID) (*assetConstInfo, error) {
	if info, ok := a.uncertainAssetInfo[assetID]; ok {
		return &info.assetConstInfo, nil
	}
	if info, ok := a.freshConstInfo[assetID]; ok {
		return &info, nil
	}
	return a.constInfo(assetID)
}

func (a *assets) newestChangeableInfo(asset crypto.Digest) (*assetChangeableInfo, error) {
	assetID := proto.AssetIDFromDigest(asset)
	if info, ok := a.uncertainAssetInfo[assetID]; ok {
		return &info.assetChangeableInfo, nil
	}
	histKey := assetHistKey{assetID: assetID}
	recordBytes, err := a.hs.newestTopEntryData(histKey.bytes())
	if err != nil {
		return nil, err
	}
	var record assetHistoryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.assetChangeableInfo, nil
}

func (a *assets) newestAssetExists(asset proto.OptionalAsset) bool {
	if !asset.Present {
		// Waves.
		return true
	}
	assetID := proto.AssetIDFromDigest(asset.ID)
	if _, err := a.newestAssetInfo(assetID); err != nil { // TODO: check error type
		return false
	}
	return true
}

// Newest asset info (from local storage, or from DB if given asset has not been changed).
// This is needed for transactions validation.
func (a *assets) newestAssetInfo(assetID proto.AssetID) (*assetInfo, error) {
	constInfo, err := a.newestConstInfo(assetID)
	if err != nil {
		return nil, err
	}
	changeableInfo, err := a.newestChangeableInfo(proto.ReconstructDigest(assetID, constInfo.tail))
	if err != nil {
		return nil, err
	}
	return &assetInfo{*constInfo, *changeableInfo}, nil
}

// "Stable" asset info from database.
// This should be used by external APIs.
func (a *assets) assetInfo(assetID proto.AssetID) (*assetInfo, error) {
	constInfo, err := a.constInfo(assetID) // `errs.UnknownAsset` error here
	if err != nil {
		return nil, err
	}
	histKey := assetHistKey{assetID: assetID}
	recordBytes, err := a.hs.topEntryData(histKey.bytes())
	if err != nil {
		return nil, err // `keyvalue.ErrNotFound` or "empty history" errors here
	}
	var record assetHistoryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &assetInfo{assetConstInfo: *constInfo, assetChangeableInfo: record.assetChangeableInfo}, nil
}

// commitUncertain() moves all uncertain changes to historyStorage.
func (a *assets) commitUncertain(blockID proto.BlockID) error {
	for assetID, info := range a.uncertainAssetInfo {
		infoCpy := info // prevent implicit memory aliasing in for loop
		if err := a.storeAssetInfo(assetID, &infoCpy, blockID); err != nil {
			return err
		}
	}
	return nil
}

// dropUncertain() removes all uncertain changes.
func (a *assets) dropUncertain() {
	a.uncertainAssetInfo = make(map[proto.AssetID]assetInfo)
}

func (a *assets) reset() {
	a.freshConstInfo = make(map[proto.AssetID]assetConstInfo)
}
