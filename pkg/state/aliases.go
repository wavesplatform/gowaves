package state

import (
	"bytes"
	"io"
	"math"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// errAliasDisabled is wrapped keyvalue.ErrNotFound which should be used for disabled aliases.
var errAliasDisabled = errors.Wrap(keyvalue.ErrNotFound, "alias was stolen and is now disabled")

const aliasRecordSize = 1 + proto.AddressIDSize

type aliasRecordForStateHashes struct {
	addr  proto.WavesAddress
	alias []byte
}

func (ar *aliasRecordForStateHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(ar.addr[:]); err != nil {
		return err
	}
	if _, err := w.Write(ar.alias); err != nil {
		return err
	}
	return nil
}

func (ar *aliasRecordForStateHashes) less(other stateComponent) bool {
	ar2 := other.(*aliasRecordForStateHashes)
	val := bytes.Compare(ar.addr[:], ar2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(ar.alias, ar2.alias) == -1
	}
	return true
}

type aliasInfo struct {
	stolen    bool
	addressID proto.AddressID
}

type aliasRecord struct {
	info aliasInfo
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, aliasRecordSize)
	proto.PutBool(res[:1], r.info.stolen)
	copy(res[1:1+proto.AddressIDSize], r.info.addressID[:])
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) != aliasRecordSize {
		return errInvalidDataSize
	}
	var err error
	r.info.stolen, err = proto.Bool(data[:1])
	if err != nil {
		return err
	}
	copy(r.info.addressID[:], data[1:1+proto.AddressIDSize])
	return nil
}

type addressToAliasesRecord struct {
	aliases []string
}

func (r *addressToAliasesRecord) marshalBinary() ([]byte, error) {
	var (
		b   []byte
		err error
	)
	for _, a := range r.aliases {
		b, err = appendStringWithUInt8Len(b, a)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func appendStringWithUInt8Len(b []byte, s string) ([]byte, error) {
	l := len(s)
	if l > math.MaxUint8 {
		return nil, errors.New("length of the string is bigger than uint8")
	}
	b = append(b, uint8(l))
	return append(b, s...), nil
}

func (r *addressToAliasesRecord) unmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	rdr := bytes.NewReader(data)
	for {
		al, err := readStringWithUInt8Len(rdr)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
		r.aliases = append(r.aliases, al)
	}
	return nil
}

func readStringWithUInt8Len(r io.Reader) (string, error) {
	var l [1]uint8
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return "", err
	}
	b := make([]byte, l[0])
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *addressToAliasesRecord) removeIfExists(s string) bool {
	for i, al := range r.aliases {
		if al == s {
			r.aliases[i] = r.aliases[len(r.aliases)-1] // replace with the last item
			r.aliases = r.aliases[:len(r.aliases)-1]   // cut duplicate of the last item
			return true
		}
	}
	return false
}

type aliases struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage

	disabled map[string]bool

	scheme          proto.Scheme
	calculateHashes bool
	hasher          *stateHasher
}

func newAliases(hs *historyStorage, scheme proto.Scheme, calcHashes bool) *aliases {
	return &aliases{
		db:              hs.db,
		dbBatch:         hs.dbBatch,
		hs:              hs,
		disabled:        make(map[string]bool),
		scheme:          scheme,
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

func (a *aliases) createAlias(aliasStr string, addr proto.WavesAddress, blockID proto.BlockID) error {
	key := aliasKey{aliasStr}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	r := aliasRecord{
		info: aliasInfo{
			stolen:    a.exists(aliasStr),
			addressID: addr.ID(),
		},
	}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	if a.calculateHashes {
		ar := &aliasRecordForStateHashes{
			addr:  addr,
			alias: []byte(aliasStr),
		}
		if err := a.hasher.push(keyStr, ar, blockID); err != nil {
			return err
		}
	}
	if err := a.hs.addNewEntry(alias, keyBytes, recordBytes, blockID); err != nil {
		return errors.Wrapf(err, "failed to add alias record %q for addr %q, blockID %q", aliasStr, addr.String(), blockID.String())
	}
	return a.addOrUpdateAddressToAliasesRecord(aliasStr, addr, blockID)
}

func (a *aliases) addOrUpdateAddressToAliasesRecord(aliasStr string, addr proto.WavesAddress, blockID proto.BlockID) error {
	key := addressToAliasesKey{addressID: addr.ID()}
	keyBytes := key.bytes()
	recordBytes, err := a.hs.newestTopEntryData(keyBytes)
	if err != nil {
		if !isNotFoundInHistoryOrDBErr(err) { // unexpected error
			return errors.Wrapf(err, "failed to add alias record %q for addr %q, blockID %q", aliasStr, addr.String(), blockID.String())
		}
		record := addressToAliasesRecord{aliases: []string{aliasStr}}
		recordBytes, err = record.marshalBinary()
	} else {
		recordBytes, err = appendStringWithUInt8Len(recordBytes, aliasStr)
	}
	if err != nil {
		return err
	}
	if err := a.hs.addNewEntry(addressToAliasesKeySize, keyBytes, recordBytes, blockID); err != nil {
		return errors.Wrapf(err, "failed to add address to aliases record with new alias %q for addr %q, blockID %q",
			aliasStr, addr.String(), blockID.String(),
		)
	}
	return nil
}

func (a *aliases) exists(aliasStr string) bool {
	key := aliasKey{alias: aliasStr}
	if _, err := a.hs.newestTopEntryData(key.bytes()); err != nil {
		return false
	}
	return true
}

func (a *aliases) newestIsDisabled(aliasStr string) (bool, error) {
	if _, ok := a.disabled[aliasStr]; ok {
		return true, nil
	}
	return a.isDisabled(aliasStr)
}

func (a *aliases) isDisabled(aliasStr string) (bool, error) {
	key := disabledAliasKey{alias: aliasStr}
	return a.db.Has(key.bytes())
}

func (a *aliases) newestAddrByAlias(aliasStr string) (proto.WavesAddress, error) {
	disabled, err := a.newestIsDisabled(aliasStr)
	if err != nil {
		return proto.WavesAddress{}, err
	}
	if disabled {
		return proto.WavesAddress{}, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	record, err := a.newestRecordByAlias(key.bytes())
	if err != nil {
		return proto.WavesAddress{}, err
	}
	return record.info.addressID.ToWavesAddress(a.scheme)
}

func (a *aliases) newestRecordByAlias(key []byte) (aliasRecord, error) {
	recordBytes, err := a.hs.newestTopEntryData(key)
	if err != nil {
		return aliasRecord{}, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return aliasRecord{}, errors.Wrap(err, "failed to unmarshal record")
	}
	return record, nil
}

func (a *aliases) recordByAlias(key []byte) (aliasRecord, error) {
	recordBytes, err := a.hs.topEntryData(key)
	if err != nil {
		return aliasRecord{}, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return aliasRecord{}, errors.Wrap(err, "failed to unmarshal record")
	}
	return record, nil
}

func (a *aliases) addrByAlias(aliasStr string) (proto.WavesAddress, error) {
	disabled, err := a.isDisabled(aliasStr)
	if err != nil {
		return proto.WavesAddress{}, err
	}
	if disabled {
		return proto.WavesAddress{}, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	record, err := a.recordByAlias(key.bytes())
	if err != nil {
		return proto.WavesAddress{}, err
	}
	return record.info.addressID.ToWavesAddress(a.scheme)
}

func (a *aliases) disableStolenAliases(blockID proto.BlockID) error {
	// TODO: this action can not be rolled back now, do we need it?
	iter, err := a.hs.newNewestTopEntryIterator(alias)
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()
	for iter.Next() {
		keyBytes := iter.Key()
		recordBytes := iter.Value()
		var record aliasRecord
		if err := record.unmarshalBinary(recordBytes); err != nil {
			return errors.Wrap(err, "failed to unmarshal alias record")
		}
		var key aliasKey
		if err := key.unmarshal(keyBytes); err != nil {
			return errors.Wrap(err, "failed to unmarshal alias key")
		}
		if !record.info.stolen { // skip if alias is not stolen
			continue
		}
		zap.S().Debugf("Forbidding stolen alias %s", key.alias)
		a.disabled[key.alias] = true
		if err := a.removeAliasByAddressID(record.info.addressID, key.alias, blockID); err != nil {
			return errors.Wrap(err, "failed to disable aliases")
		}
	}
	return nil
}

func (a *aliases) removeAliasByAddressID(id proto.AddressID, aliasStr string, blockID proto.BlockID) (err error) {
	defer func() {
		if err != nil {
			addr, addrErr := id.ToWavesAddress(a.scheme)
			if addrErr != nil {
				err = errors.Wrapf(err, "failed to rebuild address: %v", addrErr)
			}
			err = errors.Wrapf(err, "failed to remove alias %q for address %q", aliasStr, addr)
		}
	}()
	key := addressToAliasesKey{addressID: id}
	keyBytes := key.bytes()
	recordBytes, err := a.hs.newestTopEntryData(keyBytes)
	if err != nil {
		addr, addrErr := id.ToWavesAddress(a.scheme)
		if addrErr != nil {
			return errors.Wrapf(err, "failed to rebuild address: %v", addrErr)
		}
		return errors.Wrapf(err, "failed to remove alias %q for address %q", aliasStr, addr.String())
	}
	var record addressToAliasesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return errors.Wrap(err, "failed to unmarshal address to aliases record")
	}
	if ok := record.removeIfExists(aliasStr); !ok {
		return errors.Errorf("alias %q is not found for the given address", aliasStr)
	}
	if err := a.hs.addNewEntry(addressToAliasesKeySize, keyBytes, recordBytes, blockID); err != nil {
		return errors.Wrap(err, "failed to update address to aliases record")
	}
	return nil
}

func (a *aliases) prepareHashes() error {
	return a.hasher.stop()
}

func (a *aliases) flush() {
	for alias := range a.disabled {
		disabledKey := disabledAliasKey{alias}
		a.dbBatch.Put(disabledKey.bytes(), void)
	}
}

func (a *aliases) reset() {
	a.disabled = make(map[string]bool)
	a.hasher.reset()
}

func (a *aliases) aliasesByAddr(addr proto.WavesAddress) ([]string, error) {
	key := addressToAliasesKey{addressID: addr.ID()}
	recordBytes, err := a.hs.topEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, nil // means that there's no aliases for the given address
		}
		return nil, errors.Wrapf(err, "failed to get address to aliases record from history by addr %q", addr.String())
	}
	var record addressToAliasesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal address to aliases record for address %q", addr.String())
	}
	return record.aliases, nil
}
