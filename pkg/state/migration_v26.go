package state

import (
	"bytes"
	"encoding/binary"
	"io"
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Migration of the state storage from version 26 to version 27.
//
// Version 27 changes the formats of the following records:
//   - Waves balances history: fixed size binary records (balance, leaseIn, leaseOut) are replaced with
//     variable size CBOR records (Balance, LeaseIn, LeaseOut, Deposit).
//   - Asset balances history: fixed size binary records (balance) are replaced with variable size CBOR records.
//   - Legacy state hashes history: the order of fields hashes in the binary representation is changed.
//     The order of fields used to calculate the SumHash is not changed.
//
// The formats were changed without the version increment, so a database of version 26 may contain records of both
// formats: records written by the node before the change and records written after it. The format of each record
// is detected, and only the records of the old format are converted. A balance history, which is valid in both
// formats, can't be converted reliably, the migration fails in this case. The format of a legacy state hash is
// detected by recalculating its SumHash from the SumHash of the previous height using both orders of fields.
const (
	stateVersionV26 = 26

	v26WavesBalanceRecordSize   = 8 + 8 + 8
	v26AssetBalanceRecordSize   = 8
	v26LegacyStateHashDigestNum = 10 // SumHash and 9 fields hashes.
	legacyStateHashV2DigestNum  = 12 // SumHash, 9 fields hashes, GeneratorsHash and GeneratorsBalancesHash.
	historyBlockNumSize         = 4
)

// migrationStepsFromV26 returns the steps of the migration from version 26 to version 27.
func migrationStepsFromV26(db *keyvalue.KeyVal) []migrationStep {
	return []migrationStep{
		{
			name:    "waves balances",
			prefix:  wavesBalanceKeyPrefix,
			convert: func(_, data []byte) ([]byte, error) { return convertV26WavesBalanceHistory(data) },
		},
		{
			name:    "asset balances",
			prefix:  assetBalanceKeyPrefix,
			convert: func(_, data []byte) ([]byte, error) { return convertV26AssetBalanceHistory(data) },
		},
		{
			name:   "legacy state hashes",
			prefix: legacyStateHashKeyPrefix,
			convert: func(key, data []byte) ([]byte, error) {
				return convertV26LegacyStateHashHistory(db, key, data)
			},
		},
	}
}

// convertV26FixedSizeHistory converts the history of fixed size records of version 26 into the history of
// variable size records, converting each record with the given function. Block numbers of the history entries
// must strictly increase.
func convertV26FixedSizeHistory(
	data []byte, entity blockchainEntity, recordSize int, convert func([]byte) ([]byte, error),
) ([]byte, error) {
	if len(data) < 1 || blockchainEntity(data[0]) != entity {
		return nil, errors.Errorf("invalid history record: expected entity type %d", entity)
	}
	entrySize := recordSize + historyBlockNumSize
	body := data[1:]
	if len(body) == 0 || len(body)%entrySize != 0 {
		return nil, errors.Errorf("invalid history record size %d for entries of size %d", len(body), entrySize)
	}
	hr := newHistoryRecord(entity)
	hr.entries = make([]historyEntry, 0, len(body)/entrySize)
	for i := 0; i < len(body); i += entrySize {
		var entry historyEntry
		if err := entry.unmarshalBinary(body[i : i+entrySize]); err != nil {
			return nil, err
		}
		if n := len(hr.entries); n > 0 && hr.entries[n-1].blockNum >= entry.blockNum {
			return nil, errors.Errorf("block numbers of history entries are not increasing: %d after %d",
				entry.blockNum, hr.entries[n-1].blockNum)
		}
		converted, err := convert(entry.data)
		if err != nil {
			return nil, err
		}
		hr.entries = append(hr.entries, historyEntry{data: converted, blockNum: entry.blockNum})
	}
	return hr.marshalBinary()
}

// isV27BalanceHistory reports whether the data is a valid history of variable size records of the given entity.
// The check is strict: the sizes of records must match the size of data exactly, block numbers must strictly
// increase, and each record must be in the canonical form, i.e. it is decoded and encoded back by the given
// function to the same bytes.
func isV27BalanceHistory(data []byte, entity blockchainEntity, recode func([]byte) ([]byte, error)) bool {
	if len(data) < 1 || blockchainEntity(data[0]) != entity {
		return false
	}
	body := data[1:]
	if len(body) == 0 {
		return false
	}
	const sizeLen = 4
	var prevBlockNum uint32
	for first := true; len(body) > 0; first = false {
		if len(body) < sizeLen {
			return false
		}
		size := uint64(binary.BigEndian.Uint32(body[:sizeLen]))
		body = body[sizeLen:]
		if size <= historyBlockNumSize || size > uint64(len(body)) {
			return false
		}
		record := body[:size-historyBlockNumSize]
		blockNum := binary.BigEndian.Uint32(body[size-historyBlockNumSize : size])
		if !first && blockNum <= prevBlockNum {
			return false
		}
		prevBlockNum = blockNum
		recoded, err := recode(record)
		if err != nil || !bytes.Equal(recoded, record) {
			return false
		}
		body = body[size:]
	}
	return true
}

// convertV26BalanceHistory detects the format of the balance history and converts it to the format of version 27.
// The data can be a valid history in both formats, for example, the asset balance history
// 0x0600000008a100182a00000001 is either the history of balance 37060876330 at block 1 in the format of version 26
// or the history of balance 42 at block 1 in the format of version 27. There is no way to choose the right
// interpretation, so an error is returned in this case.
func convertV26BalanceHistory(
	data []byte, entity blockchainEntity, recordSize int,
	recode func([]byte) ([]byte, error), convert func([]byte) ([]byte, error),
) ([]byte, error) {
	isV27 := isV27BalanceHistory(data, entity, recode)
	converted, err := convertV26FixedSizeHistory(data, entity, recordSize, convert)
	switch {
	case isV27 && err == nil:
		return nil, errors.New("ambiguous history, it is valid in formats of both versions 26 and 27, " +
			"the state has to be re-imported")
	case isV27:
		return data, nil
	default:
		return converted, err
	}
}

func convertV26WavesBalanceHistory(data []byte) ([]byte, error) {
	recode := func(b []byte) ([]byte, error) {
		var r wavesBalanceRecord
		if err := r.unmarshalBinary(b); err != nil {
			return nil, err
		}
		return r.marshalBinary()
	}
	return convertV26BalanceHistory(data, wavesBalance, v26WavesBalanceRecordSize, recode,
		convertV26WavesBalanceRecord)
}

func convertV26WavesBalanceRecord(data []byte) ([]byte, error) {
	if len(data) != v26WavesBalanceRecordSize {
		return nil, errors.Errorf("invalid waves balance record size %d", len(data))
	}
	r := wavesBalanceRecord{balanceProfile{
		Balance: binary.BigEndian.Uint64(data[:8]),
		// Lease balances were stored as uint64 representation of int64 values.
		LeaseIn:  int64(binary.BigEndian.Uint64(data[8:16])),  // #nosec G115 // see above.
		LeaseOut: int64(binary.BigEndian.Uint64(data[16:24])), // #nosec G115 // see above.
	}}
	return r.marshalBinary()
}

func convertV26AssetBalanceHistory(data []byte) ([]byte, error) {
	recode := func(b []byte) ([]byte, error) {
		var r assetBalanceRecord
		if err := r.unmarshalBinary(b); err != nil {
			return nil, err
		}
		return r.marshalBinary()
	}
	return convertV26BalanceHistory(data, assetBalance, v26AssetBalanceRecordSize, recode,
		convertV26AssetBalanceRecord)
}

func convertV26AssetBalanceRecord(data []byte) ([]byte, error) {
	if len(data) != v26AssetBalanceRecordSize {
		return nil, errors.Errorf("invalid asset balance record size %d", len(data))
	}
	r := assetBalanceRecord{Balance: binary.BigEndian.Uint64(data)}
	return r.marshalBinary()
}

// convertV26LegacyStateHashHistory converts entries of the legacy state hash history. Entries of the blocks removed
// by rollbacks are discarded the same way as historyFormatter.filter does, because they can't be validated against
// the SumHashes of the previous height: the previous history could be already rewritten by another chain.
// The history without valid entries is removed, the previous height is not checked in this case.
func convertV26LegacyStateHashHistory(db *keyvalue.KeyVal, key, data []byte) ([]byte, error) {
	if len(key) != legacyStateHashKeySize {
		return nil, errors.Errorf("invalid legacy state hash key size %d", len(key))
	}
	height := binary.BigEndian.Uint64(key[1:])
	hr, err := readValidLegacyStateHashHistory(db, data)
	if err != nil {
		return nil, errors.Wrapf(err, "legacy state hash at height %d", height)
	}
	if len(hr.entries) == 0 {
		return nil, nil // all entries belong to the removed blocks, remove the history
	}
	prevSums, err := legacyStateHashSums(db, height-1)
	if err != nil {
		return nil, err
	}
	for i := range hr.entries {
		converted, cErr := convertV26LegacyStateHash(hr.entries[i].data, prevSums)
		if cErr != nil {
			return nil, errors.Wrapf(cErr, "legacy state hash at height %d", height)
		}
		hr.entries[i].data = converted
	}
	return hr.marshalBinary()
}

func readLegacyStateHashHistory(data []byte) (*historyRecord, error) {
	hr, err := newHistoryRecordFromBytes(data) // history of legacy state hashes was of variable size already
	if err != nil {
		return nil, err
	}
	if hr.entityType != legacyStateHash {
		return nil, errors.Errorf("invalid history record: expected entity type %d, got %d",
			legacyStateHash, hr.entityType)
	}
	return hr, nil
}

// readValidLegacyStateHashHistory reads the legacy state hash history and removes the entries of the blocks
// removed by rollbacks from the end of the history, like historyFormatter.filter does.
func readValidLegacyStateHashHistory(db *keyvalue.KeyVal, data []byte) (*historyRecord, error) {
	hr, err := readLegacyStateHashHistory(data)
	if err != nil {
		return nil, err
	}
	for i, e := range slices.Backward(hr.entries) {
		valid, vErr := db.Has((&validBlockNumKey{blockNum: e.blockNum}).bytes())
		if vErr != nil {
			return nil, errors.Wrap(vErr, "failed to check validity of block")
		}
		if valid {
			break
		}
		hr.entries = hr.entries[:i]
	}
	return hr, nil
}

// legacyStateHashSums returns SumHashes of all valid entries of the legacy state hash history at the given height.
// The position of SumHash is the same in both formats, so the history can be already converted.
// For the zero height the nil SumHash is returned, it is used to calculate the SumHash of the genesis block.
func legacyStateHashSums(db *keyvalue.KeyVal, height uint64) ([][]byte, error) {
	if height == 0 {
		return [][]byte{nil}, nil
	}
	data, err := db.Get((&legacyStateHashKey{height: height}).bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read legacy state hash at height %d", height)
	}
	hr, err := readValidLegacyStateHashHistory(db, data)
	if err != nil {
		return nil, errors.Wrapf(err, "legacy state hash at height %d", height)
	}
	if len(hr.entries) == 0 {
		return nil, errors.Errorf("no valid legacy state hash at height %d", height)
	}
	sums := make([][]byte, 0, len(hr.entries))
	for _, e := range hr.entries {
		if len(e.data) < 1 || len(e.data) < 1+int(e.data[0])+crypto.DigestSize {
			return nil, errors.Errorf("invalid legacy state hash size %d at height %d", len(e.data), height)
		}
		pos := 1 + int(e.data[0])
		sums = append(sums, e.data[pos:pos+crypto.DigestSize])
	}
	return sums, nil
}

// convertV26LegacyStateHash converts the legacy state hash of version 26 into the binary representation of
// proto.StateHashV1. Version 26 layout: block ID size, block ID, SumHash, DataEntryHash, AccountScriptHash,
// AssetScriptHash, LeaseStatusHash, SponsorshipHash, AliasesHash, WavesBalanceHash, AssetBalanceHash,
// LeaseBalanceHash. The format is detected by recalculation of SumHash with one of the given previous SumHashes.
// The legacy state hash of the new format is returned unchanged.
func convertV26LegacyStateHash(data []byte, prevSums [][]byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("empty legacy state hash")
	}
	idLen := int(data[0])
	switch len(data) {
	case 1 + idLen + crypto.DigestSize*v26LegacyStateHashDigestNum:
	case 1 + idLen + crypto.DigestSize*legacyStateHashV2DigestNum:
		return data, nil // only StateHashV1 existed before the change of format, StateHashV2 is of the new format
	default:
		return nil, errors.Errorf("invalid legacy state hash size %d", len(data))
	}
	var actual proto.StateHashV1
	if err := actual.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	if ok, err := matchesSumHash(actual, prevSums); err != nil || ok {
		return data, err // already of the new format
	}
	converted, err := readV26LegacyStateHash(data, idLen)
	if err != nil {
		return nil, err
	}
	ok, err := matchesSumHash(*converted, prevSums)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("SumHash doesn't match fields hashes in any order, unknown format")
	}
	return converted.MarshalBinary()
}

func matchesSumHash(sh proto.StateHashV1, prevSums [][]byte) (bool, error) {
	expected := sh.SumHash
	for _, prev := range prevSums {
		if err := sh.GenerateSumHash(prev); err != nil {
			return false, err
		}
		if sh.SumHash == expected {
			return true, nil
		}
	}
	return false, nil
}

func readV26LegacyStateHash(data []byte, idLen int) (*proto.StateHashV1, error) {
	id, err := proto.NewBlockIDFromBytes(data[1 : 1+idLen])
	if err != nil {
		return nil, errors.Wrap(err, "invalid block ID of legacy state hash")
	}
	r := bytes.NewReader(data[1+idLen:])
	sh := proto.StateHashV1{BlockID: id}
	for _, d := range []*crypto.Digest{
		&sh.SumHash, &sh.DataEntryHash, &sh.AccountScriptHash, &sh.AssetScriptHash, &sh.LeaseStatusHash,
		&sh.SponsorshipHash, &sh.AliasesHash, &sh.WavesBalanceHash, &sh.AssetBalanceHash, &sh.LeaseBalanceHash,
	} {
		if _, rErr := io.ReadFull(r, d[:]); rErr != nil {
			return nil, errors.Wrap(rErr, "failed to read legacy state hash")
		}
	}
	return &sh, nil
}
