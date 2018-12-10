package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	AssetInfoSize = crypto.DigestSize + 1 + 1 + proto.AddressSize + 2
	AssetDiffSize = 1 + 1 + 8 + 8
)

type AssetInfo struct {
	ID         crypto.Digest
	Name       string
	Issuer     proto.Address
	Decimals   uint8
	Reissuable bool
}

func (a *AssetInfo) marshalBinary() []byte {
	buf := make([]byte, AssetInfoSize+len(a.Name))
	p := 0
	copy(buf[p:], a.ID[:])
	p += crypto.DigestSize
	proto.PutStringWithUInt16Len(buf[p:], a.Name)
	p += 2 + len(a.Name)
	copy(buf[p:], a.Issuer[:])
	p += proto.AddressSize
	buf[p] = a.Decimals
	p++
	proto.PutBool(buf[p:], a.Reissuable)
	return buf
}

func (a *AssetInfo) unmarshalBinary(data []byte) error {
	if l := len(data); l < AssetInfoSize {
		return errors.Errorf("%d bytes is not enough for AssetInfo, expected %d", l, AssetInfoSize)
	}
	copy(a.ID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	s, err := proto.StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal AssetInfo from bytes")
	}
	a.Name = s
	data = data[2+len(s):]
	copy(a.Issuer[:], data[:proto.AddressSize])
	if ok, _ := a.Issuer.Validate(); !ok {
		return errors.New("invalid Issuer address")
	}
	data = data[proto.AddressSize:]
	a.Decimals = data[0]
	data = data[1:]
	a.Reissuable, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal AssetInfo from bytes")
	}
	return nil
}

type AssetUpdate struct {
	Info AssetInfo
	Diff AssetDiff
}

type AssetDiff struct {
	Created  bool
	Disabled bool
	Issued   uint64
	Burned   uint64
}

func (d *AssetDiff) Add(b AssetDiff) *AssetDiff {
	d.Created = d.Created || b.Created
	d.Disabled = d.Disabled || b.Disabled
	d.Issued = d.Issued + b.Issued
	d.Burned = d.Burned + b.Burned
	return d
}

func (d *AssetDiff) marshalBinary() []byte {
	buf := make([]byte, AssetDiffSize)
	if d.Created {
		buf[0] = 1
	}
	if d.Disabled {
		buf[1] = 1
	}
	binary.BigEndian.PutUint64(buf[2:], d.Issued)
	binary.BigEndian.PutUint64(buf[2+8:], d.Burned)
	return buf
}

func (d *AssetDiff) unmarshalBinary(data []byte) error {
	if l := len(data); l < AssetDiffSize {
		return errors.Errorf("%d is not enough bytes for assetDiff, expected %d", l, AssetDiffSize)
	}
	d.Created = data[0] == 1
	d.Disabled = data[1] == 1
	d.Issued = binary.BigEndian.Uint64(data[2:])
	d.Burned = binary.BigEndian.Uint64(data[2+8:])
	return nil
}

func AssetUpdateFromIssueV1(tx proto.IssueV1, scheme byte) (AssetUpdate, error) {
	ad, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetUpdate{}, err
	}
	info := AssetInfo{ID: *tx.ID, Name: tx.Name, Issuer: ad, Decimals: tx.Decimals, Reissuable: tx.Reissuable}
	diff := AssetDiff{Created: true, Disabled: !tx.Reissuable, Issued: tx.Quantity}
	return AssetUpdate{Info: info, Diff: diff}, nil
}

func AssetUpdateFromIssueV2(tx proto.IssueV2, scheme byte) (AssetUpdate, error) {
	ad, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetUpdate{}, err
	}
	info := AssetInfo{ID: *tx.ID, Name: tx.Name, Issuer: ad, Decimals: tx.Decimals, Reissuable: tx.Reissuable}
	diff := AssetDiff{Created: true, Disabled: !tx.Reissuable, Issued: tx.Quantity}
	return AssetUpdate{Info: info, Diff: diff}, nil
}

func AssetUpdateFromReissueV1(tx proto.ReissueV1) AssetUpdate {
	return AssetUpdate{Info: AssetInfo{ID: tx.AssetID}, Diff: AssetDiff{Issued: tx.Quantity, Disabled: !tx.Reissuable}}
}

func AssetUpdateFromReissueV2(tx proto.ReissueV2) AssetUpdate {
	return AssetUpdate{Info: AssetInfo{ID: tx.AssetID}, Diff: AssetDiff{Issued: tx.Quantity, Disabled: !tx.Reissuable}}
}

func AssetUpdateFromBurnV1(tx proto.BurnV1) AssetUpdate {
	return AssetUpdate{Info: AssetInfo{ID: tx.AssetID}, Diff: AssetDiff{Burned: tx.Amount}}
}

func AssetUpdateFromBurnV2(tx proto.BurnV2) AssetUpdate {
	return AssetUpdate{Info: AssetInfo{ID: tx.AssetID}, Diff: AssetDiff{Burned: tx.Amount}}
}
