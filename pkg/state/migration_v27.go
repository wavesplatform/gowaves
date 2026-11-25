package state

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

// Migration of the state storage from version 27 to version 28.
//
// Version 28 changes the encoding of variable size history records: the size of each entry, which precedes
// the entry data, is stored as uvarint instead of fixed 4 bytes big-endian uint32. The sizes of entries are
// small, so most of them take one or two bytes instead of four. Fixed size history records are not changed.
//
// The migration re-encodes all histories of variable size entities. Each record is converted according to the
// entity type stored in its first byte, not according to the key prefix, records of fixed size entities are
// left unchanged. The conversion is idempotent: a history, which is already of the new format, is returned
// unchanged, so an interrupted migration can be safely continued. The formats can't be confused in practice:
// the first byte of a history of version 27 is the highest byte of the size of the first entry, which is zero
// for entries shorter than 16 MiB, and the zero uvarint size is invalid in version 28; while the first byte of
// a history of version 28 makes the size of the first entry in version 27 at least 80 MiB. A history, which is
// valid in both formats, is reported as an error.
const (
	stateVersionV27 = 27

	v27EntrySizeLen = 4
)

// migrationStepsFromV27 returns the steps of the migration from version 27 to version 28, one step for the key
// prefix of each variable size history entity.
func migrationStepsFromV27(_ *keyvalue.KeyVal) []migrationStep {
	steps := make([]migrationStep, 0, len(properties))
	for i := range properties {
		entity := blockchainEntity(i) // #nosec G115 // the number of entities is less than 256.
		p, ok := entityProperties(entity)
		if !ok || p.fixedSize {
			continue
		}
		prefix, err := prefixByEntity(entity)
		if err != nil || len(prefix) != 1 {
			panic(fmt.Sprintf("no key prefix for variable size history entity %d", entity)) // programming error
		}
		steps = append(steps, migrationStep{
			name:    fmt.Sprintf("histories of entity %d", entity),
			prefix:  prefix[0],
			convert: func(_, data []byte) ([]byte, error) { return convertV27History(data) },
		})
	}
	return steps
}

// convertV27History converts the variable size history of version 27 into the format of version 28.
// Histories of fixed size entities and histories already of the new format are returned unchanged.
func convertV27History(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.Wrap(errInvalidDataSize, "empty history record")
	}
	entity := blockchainEntity(data[0])
	p, ok := entityProperties(entity)
	if !ok {
		return nil, errors.Errorf("bad entity type %v", entity)
	}
	if p.fixedSize {
		return data, nil
	}
	old, oldErr := decodeV27History(data)
	_, newErr := variableSizeEntriesCount(data)
	switch {
	case oldErr == nil && newErr == nil:
		if len(old.entries) == 0 {
			return data, nil // the empty history is the same in both formats
		}
		return nil, errors.New("ambiguous history, it is valid in formats of both versions 27 and 28")
	case oldErr == nil:
		return old.marshalBinary()
	case newErr == nil:
		return data, nil // already of the new format
	default:
		return nil, errors.Wrapf(oldErr, "invalid history of entity %d", entity)
	}
}

// decodeV27History decodes the variable size history of version 27, the size of each entry is stored as 4 bytes
// big-endian uint32. The layout is checked strictly: the entries must end exactly at the end of the data.
// The data of decoded entries points into the given slice.
func decodeV27History(data []byte) (*historyRecord, error) {
	if len(data) < 1 {
		return nil, errors.Wrap(errInvalidDataSize, "empty history record")
	}
	entity := blockchainEntity(data[0])
	p, ok := entityProperties(entity)
	if !ok {
		return nil, errors.Errorf("bad entity type %v", entity)
	}
	if p.fixedSize {
		return nil, errors.Errorf("entity type %v is of fixed size", entity)
	}
	hr := &historyRecord{entityType: entity}
	for body := data[1:]; len(body) > 0; {
		if len(body) < v27EntrySizeLen {
			return nil, errors.Wrap(errInvalidDataSize, "truncated size of history entry")
		}
		size := uint64(binary.BigEndian.Uint32(body[:v27EntrySizeLen]))
		body = body[v27EntrySizeLen:]
		if size < historyBlockNumSize || size > uint64(len(body)) {
			return nil, errors.Wrapf(errInvalidDataSize, "invalid size %d of history entry", size)
		}
		dataEnd := int(size) - historyBlockNumSize // #nosec G115 // size is not greater than len(body).
		hr.entries = append(hr.entries, historyEntry{
			data:     body[:dataEnd:dataEnd],
			blockNum: binary.BigEndian.Uint32(body[dataEnd:size]),
		})
		body = body[size:]
	}
	return hr, nil
}

// marshalV27History encodes the history in the format of version 27. It is used by the migration from version 26,
// which produces the state of version 27.
func marshalV27History(hr *historyRecord) ([]byte, error) {
	p, ok := entityProperties(hr.entityType)
	if !ok {
		return nil, errors.Errorf("bad entity type %v", hr.entityType)
	}
	if p.fixedSize {
		return hr.marshalBinary() // the format of fixed size histories is not changed
	}
	size := 1
	for i := range hr.entries {
		size += v27EntrySizeLen + hr.entries[i].size()
	}
	data := make([]byte, 1, size)
	data[0] = byte(hr.entityType)
	for i := range hr.entries {
		e := &hr.entries[i]
		data = binary.BigEndian.AppendUint32(data, uint32(e.size())) // #nosec G115 // DB value size.
		data = append(data, e.data...)
		data = binary.BigEndian.AppendUint32(data, e.blockNum)
	}
	return data, nil
}
