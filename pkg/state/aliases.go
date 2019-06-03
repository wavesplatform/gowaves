package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	aliasRecordSize = proto.AddressSize + crypto.SignatureSize
)

type aliasRecord struct {
	addr    proto.Address
	blockID crypto.Signature
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, aliasRecordSize)
	copy(res[:proto.AddressSize], r.addr[:])
	copy(res[proto.AddressSize:], r.blockID[:])
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) != aliasRecordSize {
		return errors.New("invalid data size")
	}
	copy(r.addr[:], data[:proto.AddressSize])
	copy(r.blockID[:], data[proto.AddressSize:])
	return nil
}

type aliases struct {
	hs *historyStorage
}

func newAliases(hs *historyStorage) (*aliases, error) {
	return &aliases{hs}, nil
}

func (a *aliases) createAlias(aliasStr string, r *aliasRecord) error {
	key := aliasKey{alias: aliasStr}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	return a.hs.set(alias, key.bytes(), recordBytes)
}

func (a *aliases) newestAddrByAlias(aliasStr string, filter bool) (*proto.Address, error) {
	key := aliasKey{alias: aliasStr}
	recordBytes, err := a.hs.getFresh(alias, key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.addr, nil
}

func (a *aliases) addrByAlias(aliasStr string, filter bool) (*proto.Address, error) {
	key := aliasKey{alias: aliasStr}
	recordBytes, err := a.hs.get(alias, key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.addr, nil
}
