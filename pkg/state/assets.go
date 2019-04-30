package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/history"
)

const (
	// maxQuantityLen is maximum length of quantity (it's represented as big.Int) bytes in asset history records.
	maxQuantityLen  = 16
	assetRecordSize = maxQuantityLen + 1 + crypto.SignatureSize
)

type assetInfo struct {
	assetConstInfo
	assetHistoryRecord
}

func (ai *assetInfo) equal(ai1 *assetInfo) bool {
	return ai.assetHistoryRecord.equal(&ai1.assetHistoryRecord) && (ai.assetConstInfo == ai1.assetConstInfo)
}

// assetConstInfo is part of asset info which is constant.
type assetConstInfo struct {
	name        string
	description string
	decimals    int8
}

func (ai *assetConstInfo) marshalBinary() ([]byte, error) {
	nameBuf := make([]byte, 2+len(ai.name))
	proto.PutStringWithUInt16Len(nameBuf, ai.name)
	descriptionBuf := make([]byte, 2+len(ai.description))
	proto.PutStringWithUInt16Len(descriptionBuf, ai.description)
	res := append(nameBuf, descriptionBuf...)
	res = append(res, byte(ai.decimals))
	return res, nil
}

func (ai *assetConstInfo) unmarshalBinary(data []byte) error {
	var err error
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

// assetHistoryRecord is part of asset info which can change.
type assetHistoryRecord struct {
	quantity   big.Int
	reissuable bool
	blockID    crypto.Signature
}

func (r *assetHistoryRecord) equal(r1 *assetHistoryRecord) bool {
	if r.quantity.Cmp(&r1.quantity) != 0 {
		return false
	}
	if r.reissuable != r1.reissuable {
		return false
	}
	if r.blockID != r1.blockID {
		return false
	}
	return true
}

func (r *assetHistoryRecord) marshalBinary() ([]byte, error) {
	quantityBytes := r.quantity.Bytes()
	l := len(quantityBytes)
	if l > maxQuantityLen {
		return nil, errors.Errorf("quantity length %d bytes exceeds maxQuantityLen of %d", l, maxQuantityLen)
	}
	res := make([]byte, maxQuantityLen+1+crypto.SignatureSize)
	copy(res[maxQuantityLen-l:maxQuantityLen], quantityBytes)
	proto.PutBool(res[maxQuantityLen:maxQuantityLen+1], r.reissuable)
	copy(res[maxQuantityLen+1:], r.blockID[:])
	return res, nil
}

func (r *assetHistoryRecord) unmarshalBinary(data []byte) error {
	r.quantity.SetBytes(data[:maxQuantityLen])
	var err error
	r.reissuable, err = proto.Bool(data[maxQuantityLen : maxQuantityLen+1])
	if err != nil {
		return err
	}
	copy(r.blockID[:], data[maxQuantityLen+1:])
	return nil
}

type assets struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	// Local storage for history, is moved to batch after all the changes are made.
	// The motivation for this is inability to read from DB batch.
	localStor map[string][]byte

	// fmt is used for operations on assets history.
	fmt *history.HistoryFormatter
}

func newAssets(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	hInfo heightInfo,
	bInfo blockInfo,
) (*assets, error) {
	fmt, err := history.NewHistoryFormatter(assetRecordSize, crypto.SignatureSize, hInfo, bInfo)
	if err != nil {
		return nil, err
	}
	return &assets{
		db:        db,
		dbBatch:   dbBatch,
		localStor: make(map[string][]byte),
		fmt:       fmt,
	}, nil
}

func (a *assets) addNewRecord(assetID crypto.Digest, record *assetHistoryRecord) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	// Add new record to history.
	histKey := assetHistKey{assetID: assetID}
	history, _ := a.localStor[string(histKey.bytes())]
	history, err = a.fmt.AddRecord(history, recordBytes)
	if err != nil {
		return errors.Errorf("failed to add asset record to history: %v\n", err)
	}
	a.localStor[string(histKey.bytes())] = history
	return nil
}

func (a *assets) issueAsset(assetID crypto.Digest, asset *assetInfo) error {
	assetConstBytes, err := asset.assetConstInfo.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal asset const info: %v\n", err)
	}
	constKey := assetConstKey{assetID: assetID}
	a.dbBatch.Put(constKey.bytes(), assetConstBytes)
	return a.addNewRecord(assetID, &asset.assetHistoryRecord)
}

type assetReissueChange struct {
	reissuable bool
	diff       int64
	blockID    crypto.Signature
}

func (a *assets) reissueAsset(assetID crypto.Digest, ch *assetReissueChange) error {
	info, err := a.newestAssetRecord(assetID)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	info.quantity.Add(&info.quantity, quantityDiff)
	record := &assetHistoryRecord{reissuable: ch.reissuable, quantity: info.quantity, blockID: ch.blockID}
	return a.addNewRecord(assetID, record)
}

type assetBurnChange struct {
	diff    int64
	blockID crypto.Signature
}

func (a *assets) burnAsset(assetID crypto.Digest, ch *assetBurnChange) error {
	info, err := a.newestAssetRecord(assetID)
	if err != nil {
		return errors.Errorf("failed to get asset info: %v\n", err)
	}
	quantityDiff := big.NewInt(ch.diff)
	if info.quantity.Cmp(quantityDiff) == -1 {
		return errors.New("trying to burn more assets than exist at all")
	}
	info.quantity.Sub(&info.quantity, quantityDiff)
	record := &assetHistoryRecord{reissuable: info.reissuable, quantity: info.quantity, blockID: ch.blockID}
	return a.addNewRecord(assetID, record)
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

func (a *assets) lastRecord(history []byte) (*assetHistoryRecord, error) {
	last, err := a.fmt.GetLatest(history)
	if err != nil {
		return nil, errors.Errorf("failed to get the last record: %v\n", err)
	}
	var record assetHistoryRecord
	if err := record.unmarshalBinary(last); err != nil {
		return nil, errors.Errorf("failed to unmarshal history record: %v\n", err)
	}
	return &record, nil
}

// Newest asset record (from local storage, or from DB if given asset has not been changed).
// This is needed for transactions validation.
func (a *assets) newestAssetRecord(assetID crypto.Digest) (*assetHistoryRecord, error) {
	histKey := assetHistKey{assetID: assetID}
	history, err := fullHistory(histKey.bytes(), a.db, a.localStor, a.fmt)
	if err != nil {
		return nil, err
	}
	record, err := a.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// "Stable" asset info from database.
// This should be used by external APIs.
func (a *assets) assetInfo(assetID crypto.Digest) (*assetInfo, error) {
	constInfo, err := a.constInfo(assetID)
	if err != nil {
		return nil, err
	}
	histKey := assetHistKey{assetID: assetID}
	history, err := a.db.Get(histKey.bytes())
	if err != nil {
		return nil, errors.Errorf("failed to retrieve history for given asset: %v\n", err)
	}
	history, err = a.fmt.Normalize(history)
	if err != nil {
		return nil, errors.Errorf("failed to normalize history: %v\n", err)
	}
	record, err := a.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return &assetInfo{assetConstInfo: *constInfo, assetHistoryRecord: *record}, nil
}

func (a *assets) reset() {
	a.localStor = make(map[string][]byte)
}

func (a *assets) flush() error {
	if err := addHistoryToBatch(a.db, a.dbBatch, a.localStor, a.fmt); err != nil {
		return err
	}
	return nil
}
