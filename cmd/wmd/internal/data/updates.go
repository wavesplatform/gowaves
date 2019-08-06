package data

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type AliasBind struct {
	Alias   proto.Alias
	Address proto.Address
}

type Account struct {
	Address proto.Address
	Alias   proto.Alias
}

func (a *Account) SetFromPublicKey(scheme byte, pk crypto.PublicKey) error {
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return errors.Wrap(err, "failed to convert PublicKey to Address")
	}
	a.Address = addr
	return nil
}

func (a *Account) SetFromRecipient(r proto.Recipient) error {
	if r.Alias != nil {
		a.Alias = *r.Alias
		return nil
	}
	if r.Address != nil {
		a.Address = *r.Address
		return nil
	}
	return errors.New("empty Recipient")
}

type AccountChange struct {
	Account      Account
	Asset        crypto.Digest
	In           uint64
	Out          uint64
	MinersReward bool
}

type IssueChange struct {
	AssetID    crypto.Digest
	Name       string
	Issuer     crypto.PublicKey
	Decimals   uint8
	Reissuable bool
	Quantity   uint64
}

type AssetChange struct {
	AssetID       crypto.Digest
	SetReissuable bool
	Reissuable    bool
	SetSponsored  bool
	Sponsored     bool
	Issued        uint64
	Burned        uint64
}

func FromIssueV1(scheme byte, tx proto.IssueV1) (IssueChange, AccountChange, error) {
	issue := IssueChange{AssetID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable, Quantity: tx.Quantity}
	change := AccountChange{Asset: *tx.ID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return IssueChange{}, AccountChange{}, errors.Wrap(err, "failed to convert IssueV1 to Change")
	}
	return issue, change, nil
}

func FromIssueV2(scheme byte, tx proto.IssueV2) (IssueChange, AccountChange, error) {
	issue := IssueChange{AssetID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable, Quantity: tx.Quantity}
	change := AccountChange{Asset: *tx.ID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return IssueChange{}, AccountChange{}, errors.Wrap(err, "failed to convert IssueV2 to Change")
	}
	return issue, change, nil
}

func FromReissueV1(scheme byte, tx proto.ReissueV1) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert ReissueV1 to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Issued: tx.Quantity, SetReissuable: true, Reissuable: tx.Reissuable}, change, nil
}

func FromReissueV2(scheme byte, tx proto.ReissueV2) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert ReissueV2 to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Issued: tx.Quantity, SetReissuable: true, Reissuable: tx.Reissuable}, change, nil
}

func FromBurnV1(scheme byte, tx proto.BurnV1) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, Out: tx.Amount}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert BurnV1 to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Burned: tx.Amount}, change, nil
}

func FromBurnV2(scheme byte, tx proto.BurnV2) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, Out: tx.Amount}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert BurnV2 to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Burned: tx.Amount}, change, nil
}

func FromTransferV1(scheme byte, tx proto.TransferV1, miner crypto.PublicKey) ([]AccountChange, error) {
	r := make([]AccountChange, 0, 4)
	if tx.AmountAsset.Present {
		ch1 := AccountChange{Asset: tx.AmountAsset.ID, Out: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		ch2 := AccountChange{Asset: tx.AmountAsset.ID, In: tx.Amount}
		err = ch2.Account.SetFromRecipient(tx.Recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if tx.FeeAsset.Present {
		ch1 := AccountChange{Asset: tx.FeeAsset.ID, Out: tx.Fee}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		ch2 := AccountChange{Asset: tx.FeeAsset.ID, In: tx.Fee, MinersReward: true}
		err = ch2.Account.SetFromPublicKey(scheme, miner)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromTransferV2(scheme byte, tx proto.TransferV2, miner crypto.PublicKey) ([]AccountChange, error) {
	r := make([]AccountChange, 0, 4)
	if tx.AmountAsset.Present {
		ch1 := AccountChange{Asset: tx.AmountAsset.ID, Out: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV2 to Change")
		}
		ch2 := AccountChange{Asset: tx.AmountAsset.ID, In: tx.Amount}
		err = ch2.Account.SetFromRecipient(tx.Recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV2 to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if tx.FeeAsset.Present {
		ch1 := AccountChange{Asset: tx.FeeAsset.ID, Out: tx.Fee}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		ch2 := AccountChange{Asset: tx.FeeAsset.ID, In: tx.Fee, MinersReward: true}
		err = ch2.Account.SetFromPublicKey(scheme, miner)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferV1 to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromExchangeV1(scheme byte, tx proto.ExchangeV1) ([]AccountChange, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to convert ExchangeV1 to Change") }
	r := make([]AccountChange, 0, 4)
	ap := tx.SellOrder.AssetPair
	if ap.AmountAsset.Present {
		ch1 := AccountChange{Asset: ap.AmountAsset.ID, In: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.BuyOrder.SenderPK)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: ap.AmountAsset.ID, Out: tx.Amount}
		err = ch2.Account.SetFromPublicKey(scheme, tx.SellOrder.SenderPK)
		if err != nil {
			return nil, wrapError(err)
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if ap.PriceAsset.Present {
		priceAssetAmount := adjustAmount(tx.Amount, tx.Price)
		ch1 := AccountChange{Asset: ap.PriceAsset.ID, Out: priceAssetAmount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.BuyOrder.SenderPK)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: ap.PriceAsset.ID, In: priceAssetAmount}
		err = ch2.Account.SetFromPublicKey(scheme, tx.SellOrder.SenderPK)
		if err != nil {
			return nil, wrapError(err)
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromExchangeV2(scheme byte, tx proto.ExchangeV2) ([]AccountChange, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to convert ExchangeV2 to Change") }
	r := make([]AccountChange, 0, 4)
	ap, buyer, _, err := extractOrderParameters(tx.BuyOrder)
	if err != nil {
		return nil, wrapError(err)
	}
	_, seller, _, err := extractOrderParameters(tx.SellOrder)
	if err != nil {
		return nil, wrapError(err)
	}
	if ap.AmountAsset.Present {
		ch1 := AccountChange{Asset: ap.AmountAsset.ID, In: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, buyer)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: ap.AmountAsset.ID, Out: tx.Amount}
		err = ch2.Account.SetFromPublicKey(scheme, seller)
		if err != nil {
			return nil, wrapError(err)
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if ap.PriceAsset.Present {
		priceAssetAmount := adjustAmount(tx.Amount, tx.Price)
		ch1 := AccountChange{Asset: ap.PriceAsset.ID, Out: priceAssetAmount}
		err := ch1.Account.SetFromPublicKey(scheme, buyer)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: ap.PriceAsset.ID, In: priceAssetAmount}
		err = ch2.Account.SetFromPublicKey(scheme, seller)
		if err != nil {
			return nil, wrapError(err)
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromMassTransferV1(scheme byte, tx proto.MassTransferV1) ([]AccountChange, error) {
	changes := make([]AccountChange, 0, len(tx.Transfers)+1)
	if tx.Asset.Present {
		var spent uint64
		for _, tr := range tx.Transfers {
			spent += tr.Amount
			ch := AccountChange{Asset: tx.Asset.ID, In: tr.Amount}
			err := ch.Account.SetFromRecipient(tr.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert MassTransferV1 to Change")
			}
			changes = append(changes, ch)
		}
		ch := AccountChange{Asset: tx.Asset.ID, Out: spent}
		err := ch.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert MassTransferV1 to StateUpdates")
		}
		changes = append(changes, ch)
		return changes, nil
	}
	return nil, nil
}

func FromSponsorshipV1(tx proto.SponsorshipV1) AssetChange {
	return AssetChange{AssetID: tx.AssetID, SetSponsored: true, Sponsored: tx.MinAssetFee > 0}
}

func FromCreateAliasV1(scheme byte, tx proto.CreateAliasV1) (AliasBind, error) {
	a := &tx.Alias
	if tx.Alias.Scheme != scheme {
		a = proto.NewAlias(scheme, tx.Alias.Alias)
		ok, err := a.Valid()
		if !ok {
			return AliasBind{}, errors.Wrap(err, "failed to create AliasBind from CreateAliasV1")
		}
	}
	ad, _ := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	return AliasBind{Alias: *a, Address: ad}, nil
}

func FromCreateAliasV2(scheme byte, tx proto.CreateAliasV2) (AliasBind, error) {
	a := &tx.Alias
	if tx.Alias.Scheme != scheme {
		a = proto.NewAlias(scheme, tx.Alias.Alias)
		ok, err := a.Valid()
		if !ok {
			return AliasBind{}, errors.Wrap(err, "failed to create AliasBind from CreateAliasV2")
		}
	}
	ad, _ := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	return AliasBind{Alias: *a, Address: ad}, nil
}

const PriceConstant = 100000000

var pc = big.NewInt(PriceConstant)

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

func extractOrderParameters(o proto.Order) (proto.AssetPair, crypto.PublicKey, uint64, error) {
	var ap proto.AssetPair
	var spk crypto.PublicKey
	var ts uint64

	switch o.GetVersion() {
	case 1:
		orderV1, ok := o.(proto.OrderV1)
		if !ok {
			p, ok := o.(*proto.OrderV1)
			if !ok {
				return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
			}
			orderV1 = *p
		}
		ap = orderV1.AssetPair
		spk = orderV1.SenderPK
		ts = orderV1.Timestamp
	case 2:
		orderV2, ok := o.(proto.OrderV2)
		if !ok {
			p, ok := o.(*proto.OrderV2)
			if !ok {
				return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
			}
			orderV2 = *p
		}
		ap = orderV2.AssetPair
		spk = orderV2.SenderPK
		ts = orderV2.Timestamp
	default:
		return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("unsupported order type")
	}
	return ap, spk, ts, nil
}
