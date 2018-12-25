package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math/big"
)

const (
	AssetInfoSize = crypto.DigestSize + 2 + crypto.PublicKeySize + 1 + 1 + 1
	AssetDiffSize = 1 + 1 + 8 + 8
	PriceConstant = 100000000
)

var (
	wavesID        = crypto.Digest{}
	lastID         = crypto.Digest{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	wavesAssetInfo = AssetInfo{ID: wavesID, Name: "WAVES", Issuer: crypto.PublicKey{}, Decimals: 8, Reissuable: false}
)

type AssetInfo struct {
	ID         crypto.Digest
	Name       string
	Issuer     crypto.PublicKey
	Decimals   uint8
	Reissuable bool
	Sponsored  bool
}

func (a *AssetInfo) marshalBinary() []byte {
	buf := make([]byte, AssetInfoSize+len(a.Name))
	p := 0
	copy(buf[p:], a.ID[:])
	p += crypto.DigestSize
	proto.PutStringWithUInt16Len(buf[p:], a.Name)
	p += 2 + len(a.Name)
	copy(buf[p:], a.Issuer[:])
	p += crypto.PublicKeySize
	buf[p] = a.Decimals
	p++
	proto.PutBool(buf[p:], a.Reissuable)
	p++
	proto.PutBool(buf[p:], a.Sponsored)
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
	copy(a.Issuer[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	a.Decimals = data[0]
	data = data[1:]
	a.Reissuable, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal AssetInfo from bytes")
	}
	data = data[1:]
	a.Sponsored, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal AssetInfo from bytes")
	}
	return nil
}

type StateUpdate struct {
	Info    AssetInfo
	Diff    AssetDiff
	Changes []AccountChange
}

type Account struct {
	PublicKey crypto.PublicKey
	Address   proto.Address
	Alias     proto.Alias
}

func (a *Account) FromRecipient(r proto.Recipient) {
	if r.Alias != nil {
		a.Alias = *r.Alias
	}
	if r.Address != nil {
		a.Address = *r.Address
	}
}

type AccountChange struct {
	Account Account
	In      uint64
	Out     uint64
}

type AssetDiff struct {
	Created   bool
	Disabled  bool
	Sponsored bool
	Issued    uint64
	Burned    uint64
}

func (d *AssetDiff) Add(b AssetDiff) *AssetDiff {
	d.Created = d.Created || b.Created
	d.Disabled = d.Disabled || b.Disabled
	d.Sponsored = d.Sponsored && b.Sponsored //TODO: implement correctly
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

func StateUpdateFromIssueV1(tx proto.IssueV1) StateUpdate {
	info := AssetInfo{ID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable}
	diff := AssetDiff{Created: true, Disabled: !tx.Reissuable, Issued: tx.Quantity}
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, In: tx.Quantity}
	return StateUpdate{Info: info, Diff: diff, Changes: []AccountChange{change}}
}

func StateUpdateFromIssueV2(tx proto.IssueV2) StateUpdate {
	info := AssetInfo{ID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable}
	diff := AssetDiff{Created: true, Disabled: !tx.Reissuable, Issued: tx.Quantity}
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, In: tx.Quantity}
	return StateUpdate{Info: info, Diff: diff, Changes: []AccountChange{change}}
}

func StateUpdateFromReissueV1(tx proto.ReissueV1) StateUpdate {
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, In: tx.Quantity}
	return StateUpdate{
		Info:    AssetInfo{ID: tx.AssetID},
		Diff:    AssetDiff{Issued: tx.Quantity, Disabled: !tx.Reissuable},
		Changes: []AccountChange{change},
	}
}

func StateUpdateFromReissueV2(tx proto.ReissueV2) StateUpdate {
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, In: tx.Quantity}
	return StateUpdate{
		Info:    AssetInfo{ID: tx.AssetID},
		Diff:    AssetDiff{Issued: tx.Quantity, Disabled: !tx.Reissuable},
		Changes: []AccountChange{change},
	}
}

func StateUpdateFromBurnV1(tx proto.BurnV1) StateUpdate {
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Amount}
	return StateUpdate{
		Info:    AssetInfo{ID: tx.AssetID},
		Diff:    AssetDiff{Burned: tx.Amount},
		Changes: []AccountChange{change},
	}
}

func StateUpdateFromBurnV2(tx proto.BurnV2) StateUpdate {
	change := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Amount}
	return StateUpdate{
		Info:    AssetInfo{ID: tx.AssetID},
		Diff:    AssetDiff{Burned: tx.Amount},
		Changes: []AccountChange{change},
	}
}

func StateUpdatesFromTransferV1(tx proto.TransferV1, miner crypto.PublicKey) []StateUpdate {
	r := make([]StateUpdate, 0, 2)
	if tx.AmountAsset.Present {
		info := AssetInfo{ID: tx.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Amount}
		var a Account
		a.FromRecipient(tx.Recipient)
		ch2 := AccountChange{Account: a, In: tx.Amount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	if tx.FeeAsset.Present {
		info := AssetInfo{ID: tx.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Fee}
		ch2 := AccountChange{Account: Account{PublicKey: miner}, In: tx.Fee}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	return r
}

func StateUpdatesFromTransferV2(tx proto.TransferV2, miner crypto.PublicKey) []StateUpdate {
	r := make([]StateUpdate, 0, 2)
	if tx.AmountAsset.Present {
		info := AssetInfo{ID: tx.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Amount}
		var a Account
		a.FromRecipient(tx.Recipient)
		ch2 := AccountChange{Account: a, In: tx.Amount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	if tx.FeeAsset.Present {
		info := AssetInfo{ID: tx.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: tx.Fee}
		ch2 := AccountChange{Account: Account{PublicKey: miner}, In: tx.Fee}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	return r
}

func StateUpdatesFromExchangeV1(tx proto.ExchangeV1) []StateUpdate {
	r := make([]StateUpdate, 0, 2)
	ap := tx.SellOrder.AssetPair
	priceAssetAmount := adjustAmount(tx.Amount, tx.Price)
	if ap.AmountAsset.Present {
		info := AssetInfo{ID: ap.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.BuyOrder.SenderPK}, In: tx.Amount}
		ch2 := AccountChange{Account: Account{PublicKey: tx.SellOrder.SenderPK}, Out: priceAssetAmount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	if ap.PriceAsset.Present {
		info := AssetInfo{ID: ap.PriceAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: tx.BuyOrder.SenderPK}, Out: priceAssetAmount}
		ch2 := AccountChange{Account: Account{PublicKey: tx.SellOrder.SenderPK}, In: tx.Amount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	return r
}

func StateUpdatesFromExchangeV2(tx proto.ExchangeV2) []StateUpdate {
	r := make([]StateUpdate, 0, 2)
	ap, buyer := extractOrderParameters(tx.BuyOrder)
	_, seller := extractOrderParameters(tx.SellOrder)
	priceAssetAmount := adjustAmount(tx.Amount, tx.Price)
	if ap.AmountAsset.Present {
		info := AssetInfo{ID: ap.AmountAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: buyer}, In: tx.Amount}
		ch2 := AccountChange{Account: Account{PublicKey: seller}, Out: priceAssetAmount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	if ap.PriceAsset.Present {
		info := AssetInfo{ID: ap.PriceAsset.ID}
		ch1 := AccountChange{Account: Account{PublicKey: buyer}, Out: priceAssetAmount}
		ch2 := AccountChange{Account: Account{PublicKey: seller}, In: tx.Amount}
		u := StateUpdate{Info: info, Changes: []AccountChange{ch1, ch2}}
		r = append(r, u)
	}
	return r
}

func StateUpdateFromMassTransferV1(tx proto.MassTransferV1) StateUpdate {
	changes := make([]AccountChange, 0, len(tx.Transfers)+1)
	if tx.Asset.Present {
		info := AssetInfo{ID: tx.Asset.ID}
		var spent uint64
		for _, tr := range tx.Transfers {
			spent += tr.Amount
			var a Account
			a.FromRecipient(tr.Recipient)
			changes = append(changes, AccountChange{Account: a, In: tr.Amount})

		}
		changes = append(changes, AccountChange{Account: Account{PublicKey: tx.SenderPK}, Out: spent})
		return StateUpdate{Info: info, Changes: changes}
	}
	return StateUpdate{}
}

func StateUpdateFromSponsorshipV1(tx proto.SponsorshipV1) StateUpdate {
	sp := tx.MinAssetFee > 0
	info := AssetInfo{ID: tx.AssetID, Sponsored: sp}
	diff := AssetDiff{Sponsored: sp}
	return StateUpdate{Info: info, Diff: diff}
}

func StateUpdateFromCreateAliasV1(tx proto.CreateAliasV1) AliasBind {
	ad, _ := proto.NewAddressFromPublicKey(tx.Alias.Scheme, tx.SenderPK)
	return AliasBind{Alias: tx.Alias, Address: ad}
}

func StateUpdateFromCreateAliasV2(tx proto.CreateAliasV2) AliasBind {
	ad, _ := proto.NewAddressFromPublicKey(tx.Alias.Scheme, tx.SenderPK)
	return AliasBind{Alias: tx.Alias, Address: ad}
}

var (
	pc = big.NewInt(PriceConstant)
)

func adjustAmount(amount, price uint64) uint64 {
	var a big.Int
	a.SetUint64(amount)
	var p big.Int
	p.SetUint64(price)
	var ap big.Int
	ap.Mul(&a, &p)
	var r big.Int
	r.Div(&ap, pc)
	return r.Uint64()
}

func extractOrderParameters(o proto.Order) (proto.AssetPair, crypto.PublicKey) {
	var ap proto.AssetPair
	var spk crypto.PublicKey

	switch o.GetVersion() {
	case 1:
		orderV1 := o.(proto.OrderV1)
		ap = orderV1.AssetPair
		spk = orderV1.SenderPK
	case 2:
		orderV2 := o.(proto.OrderV2)
		ap = orderV2.AssetPair
		spk = orderV2.SenderPK
	default:
		panic("unsupported order type")
	}
	return ap, spk
}
