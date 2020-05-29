package state

import (
	"encoding/binary"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

func (ai *assetInfo) equal(ai1 *assetInfo) bool {
	return ai.assetChangeableInfo.equal(&ai1.assetChangeableInfo) && (ai.assetConstInfo == ai1.assetConstInfo)
}

// assetConstInfo is part of asset info which is constant.
type assetConstInfo struct {
	issuer   crypto.PublicKey
	decimals int8
}

func (ai *assetConstInfo) marshalBinary() ([]byte, error) {
	res := make([]byte, crypto.PublicKeySize+1)
	if err := ai.issuer.WriteTo(res); err != nil {
		return nil, err
	}
	res[crypto.PublicKeySize] = byte(ai.decimals)
	return res, nil
}

func (ai *assetConstInfo) unmarshalBinary(data []byte) error {
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
	if err != nil {
		return err
	}
	return nil
}

type assets struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch
	hs      *historyStorage

	freshConstInfo map[crypto.Digest]assetConstInfo

	uncertainAssetInfo map[crypto.Digest]assetInfo
}

func newAssets(db keyvalue.KeyValue, dbBatch keyvalue.Batch, hs *historyStorage) (*assets, error) {
	return &assets{
		db:                 db,
		dbBatch:            dbBatch,
		hs:                 hs,
		freshConstInfo:     make(map[crypto.Digest]assetConstInfo),
		uncertainAssetInfo: make(map[crypto.Digest]assetInfo),
	}, nil
}

func (a *assets) addNewRecord(assetID crypto.Digest, record *assetHistoryRecord, blockID proto.BlockID) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	// Add new record to history.
	histKey := assetHistKey{assetID: assetID}
	return a.hs.addNewEntry(asset, histKey.bytes(), recordBytes, blockID)
}

func (a *assets) storeAssetInfo(assetID crypto.Digest, asset *assetInfo, blockID proto.BlockID) error {
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

func (a *assets) issueAsset(assetID crypto.Digest, asset *assetInfo, blockID proto.BlockID) error {
	return a.storeAssetInfo(assetID, asset, blockID)
}

// issueAssetUncertain() is similar to issueAsset() but the changes can be
// dropped later using dropUncertain() or commited using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) issueAssetUncertain(assetID crypto.Digest, asset *assetInfo) {
	a.uncertainAssetInfo[assetID] = *asset
}

type assetReissueChange struct {
	reissuable bool
	diff       int64
}

func (a *assets) applyReissue(assetID crypto.Digest, ch *assetReissueChange, filter bool) (*assetInfo, error) {
	info, err := a.newestAssetInfo(assetID, filter)
	if err != nil {
		return nil, errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	info.quantity.Add(&info.quantity, quantityDiff)
	info.reissuable = ch.reissuable
	return info, nil
}

func (a *assets) reissueAsset(assetID crypto.Digest, ch *assetReissueChange, blockID proto.BlockID, filter bool) error {
	info, err := a.applyReissue(assetID, ch, filter)
	if err != nil {
		return err
	}
	return a.storeAssetInfo(assetID, info, blockID)
}

// reissueAssetUncertain() is similar to reissueAsset() but the changes can be
// dropped later using dropUncertain() or commited using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) reissueAssetUncertain(assetID crypto.Digest, ch *assetReissueChange, filter bool) error {
	info, err := a.applyReissue(assetID, ch, filter)
	if err != nil {
		return err
	}
	a.uncertainAssetInfo[assetID] = *info
	return nil
}

type assetBurnChange struct {
	diff int64
}

func (a *assets) applyBurn(assetID crypto.Digest, ch *assetBurnChange, filter bool) (*assetInfo, error) {
	info, err := a.newestAssetInfo(assetID, filter)
	if err != nil {
		return nil, errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	if info.quantity.Cmp(quantityDiff) == -1 {
		return nil, errors.New("trying to burn more assets than exist at all")
	}
	info.quantity.Sub(&info.quantity, quantityDiff)
	return info, nil
}

func (a *assets) burnAsset(assetID crypto.Digest, ch *assetBurnChange, blockID proto.BlockID, filter bool) error {
	info, err := a.applyBurn(assetID, ch, filter)
	if err != nil {
		return err
	}
	return a.storeAssetInfo(assetID, info, blockID)
}

// burnAssetUncertain() is similar to burnAsset() but the changes can be
// dropped later using dropUncertain() or commited using commitUncertain().
// newest*() functions will take changes into account even before commitUncertain().
func (a *assets) burnAssetUncertain(assetID crypto.Digest, ch *assetBurnChange, filter bool) error {
	info, err := a.applyBurn(assetID, ch, filter)
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

func (a *assets) updateAssetInfo(assetID crypto.Digest, ch *assetInfoChange, blockID proto.BlockID, filter bool) error {
	info, err := a.newestChangeableInfo(assetID, filter)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	info.name = ch.newName
	info.description = ch.newDescription
	info.lastNameDescChangeHeight = ch.newHeight
	record := &assetHistoryRecord{assetChangeableInfo: *info}
	return a.addNewRecord(assetID, record, blockID)
}

func (a *assets) newestLastUpdateHeight(assetID crypto.Digest, filter bool) (uint64, error) {
	assetInfo, err := a.newestAssetInfo(assetID, filter)
	if err != nil {
		return 0, err
	}
	return assetInfo.lastNameDescChangeHeight, nil
}

func (a *assets) constInfo(assetID crypto.Digest) (*assetConstInfo, error) {
	constKey := assetConstKey{assetID: assetID}
	constInfoBytes, err := a.db.Get(constKey.bytes())
	if err != nil {
		return nil, errors.Errorf("failed to retrieve const info for given asset: %v\n", err)
	}
	var constInfo assetConstInfo
	if err := constInfo.unmarshalBinary(constInfoBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal const info: %v\n", err)
	}
	return &constInfo, nil
}

func (a *assets) newestConstInfo(assetID crypto.Digest) (*assetConstInfo, error) {
	if info, ok := a.uncertainAssetInfo[assetID]; ok {
		return &info.assetConstInfo, nil
	}
	if info, ok := a.freshConstInfo[assetID]; ok {
		return &info, nil
	}
	return a.constInfo(assetID)
}

func (a *assets) newestChangeableInfo(assetID crypto.Digest, filter bool) (*assetChangeableInfo, error) {
	if info, ok := a.uncertainAssetInfo[assetID]; ok {
		return &info.assetChangeableInfo, nil
	}
	histKey := assetHistKey{assetID: assetID}
	recordBytes, err := a.hs.freshLatestEntryData(histKey.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record assetHistoryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.assetChangeableInfo, nil
}

func (a *assets) newestAssetExists(asset proto.OptionalAsset, filter bool) bool {
	if !asset.Present {
		// Waves.
		return true
	}
	if _, err := a.newestAssetInfo(asset.ID, filter); err != nil {
		return false
	}
	return true
}

// Newest asset info (from local storage, or from DB if given asset has not been changed).
// This is needed for transactions validation.
func (a *assets) newestAssetInfo(assetID crypto.Digest, filter bool) (*assetInfo, error) {
	constInfo, err := a.newestConstInfo(assetID)
	if err != nil {
		return nil, err
	}
	changeableInfo, err := a.newestChangeableInfo(assetID, filter)
	if err != nil {
		return nil, err
	}
	return &assetInfo{*constInfo, *changeableInfo}, nil
}

// "Stable" asset info from database.
// This should be used by external APIs.
func (a *assets) assetInfo(assetID crypto.Digest, filter bool) (*assetInfo, error) {
	constInfo, err := a.constInfo(assetID)
	if err != nil {
		return nil, err
	}
	histKey := assetHistKey{assetID: assetID}
	recordBytes, err := a.hs.latestEntryData(histKey.bytes(), filter)
	if err != nil {
		return nil, err
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
		if err := a.storeAssetInfo(assetID, &info, blockID); err != nil {
			return err
		}
	}
	return nil
}

// dropUncertain() removes all uncertain changes.
func (a *assets) dropUncertain() {
	a.uncertainAssetInfo = make(map[crypto.Digest]assetInfo)
}

func (a *assets) reset() {
	a.freshConstInfo = make(map[crypto.Digest]assetConstInfo)
}
