package state

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// Key sizes.
	wavesBalanceKeySize = 1 + proto.AddressSize
	assetBalanceKeySize = 1 + proto.AddressSize + crypto.DigestSize

	approvedFeaturesKeySize = 1 + 2
	votesFeaturesKeySize    = 1 + 2

	// Balances.
	wavesBalanceKeyPrefix byte = iota
	assetBalanceKeyPrefix

	// Valid block IDs.
	blockIdKeyPrefix

	// For block storage.
	// IDs of blocks and transactions --> offsets in files.
	blockOffsetKeyPrefix
	txOffsetKeyPrefix

	// Minimum height to which rollback is possible.
	rollbackMinHeightKeyPrefix
	// Min height of blockReadWriter's files.
	rwHeightKeyPrefix
	// Height of main db.
	dbHeightKeyPrefix

	// Score at height.
	scoreKeyPrefix
	// Assets.
	assetConstKeyPrefix
	assetHistKeyPrefix

	// Leases.
	leaseKeyPrefix
	// Known peers.
	knownPeersPrefix

	// Aliases.
	aliasKeyPrefix

	// Features.
	activatedFeaturesKeyPrefix
	approvedFeaturesKeyPrefix
	votesFeaturesKeyPrefix
)

type wavesBalanceKey struct {
	address proto.Address
}

func (k *wavesBalanceKey) bytes() []byte {
	buf := make([]byte, wavesBalanceKeySize)
	buf[0] = wavesBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	return buf
}

func (k *wavesBalanceKey) unmarshal(data []byte) error {
	if len(data) != wavesBalanceKeySize {
		return errors.New("invalid data size")
	}
	var err error
	if k.address, err = proto.NewAddressFromBytes(data[1 : 1+proto.AddressSize]); err != nil {
		return err
	}
	return nil
}

type assetBalanceKey struct {
	address proto.Address
	asset   []byte
}

func (k *assetBalanceKey) bytes() []byte {
	buf := make([]byte, assetBalanceKeySize)
	buf[0] = assetBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.AddressSize:], k.asset)
	return buf
}

func (k *assetBalanceKey) unmarshal(data []byte) error {
	if len(data) != assetBalanceKeySize {
		return errors.New("invalid data size")
	}
	var err error
	if k.address, err = proto.NewAddressFromBytes(data[1 : 1+proto.AddressSize]); err != nil {
		return err
	}
	k.asset = make([]byte, crypto.DigestSize)
	copy(k.asset, data[1+proto.AddressSize:])
	return nil
}

type blockIdKey struct {
	blockID crypto.Signature
}

func (k *blockIdKey) bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = blockIdKeyPrefix
	copy(buf[1:], k.blockID[:])
	return buf
}

type blockOffsetKey struct {
	blockID crypto.Signature
}

func (k *blockOffsetKey) bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = blockOffsetKeyPrefix
	copy(buf[1:], k.blockID[:])
	return buf
}

type txOffsetKey struct {
	txID []byte
}

func (k *txOffsetKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = txOffsetKeyPrefix
	copy(buf[1:], k.txID)
	return buf
}

type scoreKey struct {
	height uint64
}

func (k *scoreKey) bytes() []byte {
	buf := make([]byte, 9)
	buf[0] = scoreKeyPrefix
	binary.LittleEndian.PutUint64(buf[1:], k.height)
	return buf
}

type assetConstKey struct {
	assetID crypto.Digest
}

func (k *assetConstKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = assetConstKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type assetHistKey struct {
	assetID crypto.Digest
}

func (k *assetHistKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = assetHistKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type leaseKey struct {
	leaseID crypto.Digest
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
	aliasBytes := []byte(k.alias)
	buf := make([]byte, 1+len(aliasBytes))
	buf[0] = aliasKeyPrefix
	copy(buf[1:], aliasBytes[:])
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
		return errors.New("invalid data size")
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
		return errors.New("invalid data size")
	}
	buf := bytes.NewBuffer(data[1:])
	if err := binary.Read(buf, binary.BigEndian, &k.featureID); err != nil {
		return err
	}
	return nil
}
