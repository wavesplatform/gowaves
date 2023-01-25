package data

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type AliasBind struct {
	Alias   proto.Alias
	Address proto.WavesAddress
}

type Account struct {
	Address proto.WavesAddress
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

// SetFromAddress set address to Account trying to convert proto.Address interface to proto.WavesAddress structure.
func (a *Account) SetFromAddress(scheme byte, address proto.Address) error {
	wavesAddress, err := address.ToWavesAddress(scheme)
	if err != nil {
		return errors.Wrapf(err, "failed to convert address (%T) to (%T)", address, wavesAddress)
	}
	a.Address = wavesAddress
	return nil
}

func (a *Account) SetFromRecipient(r proto.Recipient) error {
	switch {
	case r.Address() != nil:
		a.Address = *r.Address()
	case r.Alias() != nil:
		a.Alias = *r.Alias()
	default:
		return errors.New("empty Recipient")
	}
	return nil
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

func FromIssueWithSig(scheme byte, tx *proto.IssueWithSig) (IssueChange, AccountChange, error) {
	issue := IssueChange{AssetID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable, Quantity: tx.Quantity}
	change := AccountChange{Asset: *tx.ID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return IssueChange{}, AccountChange{}, errors.Wrap(err, "failed to convert IssueWithSig to Change")
	}
	return issue, change, nil
}

func FromIssueWithProofs(scheme byte, tx *proto.IssueWithProofs) (IssueChange, AccountChange, error) {
	issue := IssueChange{AssetID: *tx.ID, Name: tx.Name, Issuer: tx.SenderPK, Decimals: tx.Decimals, Reissuable: tx.Reissuable, Quantity: tx.Quantity}
	change := AccountChange{Asset: *tx.ID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return IssueChange{}, AccountChange{}, errors.Wrap(err, "failed to convert IssueWithProofs to Change")
	}
	return issue, change, nil
}

func FromReissueWithSig(scheme byte, tx *proto.ReissueWithSig) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert ReissueWithSig to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Issued: tx.Quantity, SetReissuable: true, Reissuable: tx.Reissuable}, change, nil
}

func FromReissueWithProofs(scheme byte, tx *proto.ReissueWithProofs) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, In: tx.Quantity}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert ReissueWithProofs to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Issued: tx.Quantity, SetReissuable: true, Reissuable: tx.Reissuable}, change, nil
}

func FromBurnWithSig(scheme byte, tx *proto.BurnWithSig) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, Out: tx.Amount}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert BurnWithSig to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Burned: tx.Amount}, change, nil
}

func FromBurnWithProofs(scheme byte, tx *proto.BurnWithProofs) (AssetChange, AccountChange, error) {
	change := AccountChange{Asset: tx.AssetID, Out: tx.Amount}
	err := change.Account.SetFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return AssetChange{}, AccountChange{}, errors.Wrap(err, "failed to convert BurnWithProofs to Change")
	}
	return AssetChange{AssetID: tx.AssetID, Burned: tx.Amount}, change, nil
}

func FromTransferWithSig(scheme byte, tx *proto.TransferWithSig, miner crypto.PublicKey) ([]AccountChange, error) {
	r := make([]AccountChange, 0, 4)
	if tx.AmountAsset.Present {
		ch1 := AccountChange{Asset: tx.AmountAsset.ID, Out: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		ch2 := AccountChange{Asset: tx.AmountAsset.ID, In: tx.Amount}
		err = ch2.Account.SetFromRecipient(tx.Recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if tx.FeeAsset.Present {
		ch1 := AccountChange{Asset: tx.FeeAsset.ID, Out: tx.Fee}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		ch2 := AccountChange{Asset: tx.FeeAsset.ID, In: tx.Fee, MinersReward: true}
		err = ch2.Account.SetFromPublicKey(scheme, miner)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromTransferWithProofs(scheme byte, tx *proto.TransferWithProofs, miner crypto.PublicKey) ([]AccountChange, error) {
	r := make([]AccountChange, 0, 4)
	if tx.AmountAsset.Present {
		ch1 := AccountChange{Asset: tx.AmountAsset.ID, Out: tx.Amount}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithProofs to Change")
		}
		ch2 := AccountChange{Asset: tx.AmountAsset.ID, In: tx.Amount}
		err = ch2.Account.SetFromRecipient(tx.Recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithProofs to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	if tx.FeeAsset.Present {
		ch1 := AccountChange{Asset: tx.FeeAsset.ID, Out: tx.Fee}
		err := ch1.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		ch2 := AccountChange{Asset: tx.FeeAsset.ID, In: tx.Fee, MinersReward: true}
		err = ch2.Account.SetFromPublicKey(scheme, miner)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert TransferWithSig to Change")
		}
		r = append(r, ch1)
		r = append(r, ch2)
	}
	return r, nil
}

func FromExchangeWithSig(scheme byte, tx *proto.ExchangeWithSig) ([]AccountChange, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to convert ExchangeWithSig to Change") }
	accountChanges := make([]AccountChange, 0, 4)

	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return nil, wrapError(err)
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return nil, wrapError(err)
	}

	buyer, err := buyOrder.GetSender(scheme)
	if err != nil {
		return nil, wrapError(err)
	}
	seller, err := sellOrder.GetSender(scheme)
	if err != nil {
		return nil, wrapError(err)
	}

	assetPair := sellOrder.GetAssetPair()
	if assetPair.AmountAsset.Present {
		ch1 := AccountChange{Asset: assetPair.AmountAsset.ID, In: tx.Amount}
		err := ch1.Account.SetFromAddress(scheme, buyer)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: assetPair.AmountAsset.ID, Out: tx.Amount}
		err = ch2.Account.SetFromAddress(scheme, seller)
		if err != nil {
			return nil, wrapError(err)
		}
		accountChanges = append(accountChanges, ch1, ch2)
	}
	if assetPair.PriceAsset.Present {
		priceAssetAmount := adjustAmount(tx.Amount, tx.Price)
		ch1 := AccountChange{Asset: assetPair.PriceAsset.ID, Out: priceAssetAmount}
		err := ch1.Account.SetFromAddress(scheme, buyer)
		if err != nil {
			return nil, wrapError(err)
		}
		ch2 := AccountChange{Asset: assetPair.PriceAsset.ID, In: priceAssetAmount}
		err = ch2.Account.SetFromAddress(scheme, seller)
		if err != nil {
			return nil, wrapError(err)
		}
		accountChanges = append(accountChanges, ch1, ch2)
	}
	return accountChanges, nil
}

func FromExchangeWithProofs(scheme byte, tx *proto.ExchangeWithProofs) ([]AccountChange, error) {
	wrapError := func(err error) error { return errors.Wrapf(err, "failed to convert ExchangeWithProofs to Change") }
	r := make([]AccountChange, 0, 4)
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return nil, wrapError(err)
	}
	ap, buyer, _, err := extractOrderParameters(bo)
	if err != nil {
		return nil, wrapError(err)
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return nil, wrapError(err)
	}
	_, seller, _, err := extractOrderParameters(so)
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

func FromMassTransferWithProofs(scheme byte, tx *proto.MassTransferWithProofs) ([]AccountChange, error) {
	changes := make([]AccountChange, 0, len(tx.Transfers)+1)
	if tx.Asset.Present {
		var spent uint64
		for _, tr := range tx.Transfers {
			spent += tr.Amount
			ch := AccountChange{Asset: tx.Asset.ID, In: tr.Amount}
			err := ch.Account.SetFromRecipient(tr.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert MassTransferWithProofs to Change")
			}
			changes = append(changes, ch)
		}
		ch := AccountChange{Asset: tx.Asset.ID, Out: spent}
		err := ch.Account.SetFromPublicKey(scheme, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert MassTransferWithProofs to StateUpdates")
		}
		changes = append(changes, ch)
		return changes, nil
	}
	return nil, nil
}

func FromSponsorshipWithProofs(tx *proto.SponsorshipWithProofs) AssetChange {
	return AssetChange{AssetID: tx.AssetID, SetSponsored: true, Sponsored: tx.MinAssetFee > 0}
}

func FromCreateAliasWithSig(scheme byte, tx *proto.CreateAliasWithSig) (AliasBind, error) {
	a := &tx.Alias
	if tx.Alias.Scheme != scheme {
		a = proto.NewAlias(scheme, tx.Alias.Alias)
		ok, err := a.Valid(scheme)
		if !ok {
			return AliasBind{}, errors.Wrap(err, "failed to create AliasBind from CreateAliasWithSig")
		}
	}
	ad, _ := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	return AliasBind{Alias: *a, Address: ad}, nil
}

func FromCreateAliasWithProofs(scheme byte, tx *proto.CreateAliasWithProofs) (AliasBind, error) {
	a := &tx.Alias
	if tx.Alias.Scheme != scheme {
		a = proto.NewAlias(scheme, tx.Alias.Alias)
		ok, err := a.Valid(scheme)
		if !ok {
			return AliasBind{}, errors.Wrap(err, "failed to create AliasBind from CreateAliasWithProofs")
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
		orderV1, ok := o.(*proto.OrderV1)
		if !ok {
			return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
		}
		ap = orderV1.AssetPair
		spk = orderV1.SenderPK
		ts = orderV1.Timestamp
	case 2:
		orderV2, ok := o.(*proto.OrderV2)
		if !ok {
			return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
		}
		ap = orderV2.AssetPair
		spk = orderV2.SenderPK
		ts = orderV2.Timestamp
	case 3:
		orderV3, ok := o.(*proto.OrderV3)
		if !ok {
			return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
		}
		ap = orderV3.AssetPair
		spk = orderV3.SenderPK
		ts = orderV3.Timestamp
	case 4:
		orderV4, ok := o.(*proto.OrderV4)
		if !ok {
			return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("failed to extract order parameters")
		}
		ap = orderV4.AssetPair
		spk = orderV4.SenderPK
		ts = orderV4.Timestamp
	default:
		return proto.AssetPair{}, crypto.PublicKey{}, 0, errors.New("unsupported order type")
	}
	return ap, spk, ts, nil
}
