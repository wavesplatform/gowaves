package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// maxQuantityLen is maximum length of quantity (it's represented as big.Int) bytes in asset history records.
	maxQuantityLen  = 16
	assetRecordSize = maxQuantityLen + 1
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
	issuer      crypto.PublicKey
	name        string
	description string
	decimals    int8
}

func (ai *assetConstInfo) marshalBinary() ([]byte, error) {
	issuerBytes, err := ai.issuer.MarshalBinary()
	if err != nil {
		return nil, err
	}
	nameBuf := make([]byte, 2+len(ai.name))
	proto.PutStringWithUInt16Len(nameBuf, ai.name)
	res := append(issuerBytes, nameBuf...)
	descriptionBuf := make([]byte, 2+len(ai.description))
	proto.PutStringWithUInt16Len(descriptionBuf, ai.description)
	res = append(res, descriptionBuf...)
	res = append(res, byte(ai.decimals))
	return res, nil
}

func (ai *assetConstInfo) unmarshalBinary(data []byte) error {
	err := ai.issuer.UnmarshalBinary(data[:crypto.PublicKeySize])
	if err != nil {
		return err
	}
	data = data[crypto.PublicKeySize:]
	ai.name, err = proto.StringWithUInt16Len(data)
	if err != nil {
		return err
	}
	data = data[2+len(ai.name):]
	ai.description, err = proto.StringWithUInt16Len(data)
	if err != nil {
		return err
	}
	data = data[2+len(ai.description):]
	ai.decimals = int8(data[0])
	return nil
}

// assetChangeableInfo is part of asset info which can change.
type assetChangeableInfo struct {
	quantity   big.Int
	reissuable bool
}

func (r *assetChangeableInfo) equal(r1 *assetChangeableInfo) bool {
	if r.quantity.Cmp(&r1.quantity) != 0 {
		return false
	}
	if r.reissuable != r1.reissuable {
		return false
	}
	return true
}

type assetHistoryRecord struct {
	assetChangeableInfo
}

func (r *assetHistoryRecord) marshalBinary() ([]byte, error) {
	quantityBytes := r.quantity.Bytes()
	l := len(quantityBytes)
	if l > maxQuantityLen {
		return nil, errors.Errorf("quantity length %d bytes exceeds maxQuantityLen of %d", l, maxQuantityLen)
	}
	res := make([]byte, assetRecordSize)
	copy(res[maxQuantityLen-l:maxQuantityLen], quantityBytes)
	proto.PutBool(res[maxQuantityLen:maxQuantityLen+1], r.reissuable)
	return res, nil
}

func (r *assetHistoryRecord) unmarshalBinary(data []byte) error {
	if len(data) != assetRecordSize {
		return errInvalidDataSize
	}
	r.quantity.SetBytes(data[:maxQuantityLen])
	var err error
	r.reissuable, err = proto.Bool(data[maxQuantityLen : maxQuantityLen+1])
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
}

func newAssets(db keyvalue.KeyValue, dbBatch keyvalue.Batch, hs *historyStorage) (*assets, error) {
	return &assets{
		db:             db,
		dbBatch:        dbBatch,
		hs:             hs,
		freshConstInfo: make(map[crypto.Digest]assetConstInfo),
	}, nil
}

func (a *assets) addNewRecord(assetID crypto.Digest, record *assetHistoryRecord, blockID crypto.Signature) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	// Add new record to history.
	histKey := assetHistKey{assetID: assetID}
	return a.hs.addNewEntry(asset, histKey.bytes(), recordBytes, blockID)
}

func (a *assets) issueAsset(assetID crypto.Digest, asset *assetInfo, blockID crypto.Signature) error {
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

type assetReissueChange struct {
	reissuable bool
	diff       int64
}

func (a *assets) reissueAsset(assetID crypto.Digest, ch *assetReissueChange, blockID crypto.Signature, filter bool) error {
	info, err := a.newestChangeableInfo(assetID, filter)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	newValue := info.quantity.Int64() + ch.diff
	info.quantity.SetInt64(newValue)
	record := &assetHistoryRecord{assetChangeableInfo: assetChangeableInfo{info.quantity, ch.reissuable}}
	return a.addNewRecord(assetID, record, blockID)
}

type assetBurnChange struct {
	diff int64
}

func (a *assets) burnAsset(assetID crypto.Digest, ch *assetBurnChange, blockID crypto.Signature, filter bool) error {
	info, err := a.newestChangeableInfo(assetID, filter)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	if info.quantity.Cmp(quantityDiff) == -1 {
		return errors.New("trying to burn more assets than exist at all")
	}
	info.quantity.Sub(&info.quantity, quantityDiff)
	record := &assetHistoryRecord{assetChangeableInfo: assetChangeableInfo{info.quantity, info.reissuable}}
	return a.addNewRecord(assetID, record, blockID)
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
	info, ok := a.freshConstInfo[assetID]
	if ok {
		return &info, nil
	}
	return a.constInfo(assetID)
}

func (a *assets) newestChangeableInfo(assetID crypto.Digest, filter bool) (*assetChangeableInfo, error) {
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

func (a *assets) reset() {
	a.freshConstInfo = make(map[crypto.Digest]assetConstInfo)
}
