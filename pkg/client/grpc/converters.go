package grpc

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SafeTransactionConverter struct {
	err error
}

func (c *SafeTransactionConverter) address(scheme byte, addr []byte) proto.Address {
	if c.err != nil {
		return proto.Address{}
	}
	a, err := proto.RebuildAddress(scheme, addr)
	if err != nil {
		c.err = err
		return proto.Address{}
	}
	return a
}

func (c *SafeTransactionConverter) uint32(value int32) uint32 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("negative int32 value")
		return 0
	}
	return uint32(value)
}

func (c *SafeTransactionConverter) uint64(value int64) uint64 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("negative int64 value")
		return 0
	}
	return uint64(value)
}

func (c *SafeTransactionConverter) byte(value int32) byte {
	if c.err != nil {
		return 0
	}
	if value < 0 || value > 0xff {
		c.err = errors.New("invalid byte value")
	}
	return byte(value)
}

func (c *SafeTransactionConverter) digest(digest []byte) crypto.Digest {
	if c.err != nil {
		return crypto.Digest{}
	}
	r, err := crypto.NewDigestFromBytes(digest)
	if err != nil {
		c.err = err
		return crypto.Digest{}
	}
	return r
}

func (c *SafeTransactionConverter) optionalAsset(asset *AssetId) proto.OptionalAsset {
	if c.err != nil {
		return proto.OptionalAsset{}
	}
	if asset == nil {
		c.err = errors.New("empty asset")
		return proto.OptionalAsset{}
	}
	switch as := asset.Asset.(type) {
	case *AssetId_Waves:
		return proto.OptionalAsset{}
	case *AssetId_IssuedAsset:
		return proto.OptionalAsset{Present: true, ID: c.digest(as.IssuedAsset)}
	default:
		c.err = errors.New("unsupported asset")
		return proto.OptionalAsset{}
	}
}

func (c *SafeTransactionConverter) convertAmount(amount *Amount) (proto.OptionalAsset, uint64) {
	if c.err != nil {
		return proto.OptionalAsset{}, 0
	}
	return c.extractOptionalAsset(amount), c.amount(amount)
}

func (c *SafeTransactionConverter) convertAssetAmount(aa *AssetAmount) (crypto.Digest, uint64) {
	if c.err != nil {
		return crypto.Digest{}, 0
	}
	if aa == nil {
		c.err = errors.New("empty asset amount")
		return crypto.Digest{}, 0
	}
	id, err := crypto.NewDigestFromBytes(aa.AssetId)
	if err != nil {
		c.err = nil
		return crypto.Digest{}, 0
	}
	return id, c.uint64(aa.Amount)
}

func (c *SafeTransactionConverter) extractOptionalAsset(amount *Amount) proto.OptionalAsset {
	if c.err != nil {
		return proto.OptionalAsset{}
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return proto.OptionalAsset{}
	}
	return c.optionalAsset(amount.AssetId)
}

func (c *SafeTransactionConverter) amount(amount *Amount) uint64 {
	if c.err != nil {
		return 0
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return 0
	}
	if amount.Amount < 0 {
		c.err = errors.New("negative asset amount")
		return 0
	}
	return uint64(amount.Amount)
}

func (c *SafeTransactionConverter) publicKey(pk []byte) crypto.PublicKey {
	if c.err != nil {
		return crypto.PublicKey{}
	}
	r, err := crypto.NewPublicKeyFromBytes(pk)
	if err != nil {
		c.err = err
		return crypto.PublicKey{}
	}
	return r
}

func (c *SafeTransactionConverter) string(bytes []byte) string {
	if c.err != nil {
		return ""
	}
	return string(bytes)
}

func (c *SafeTransactionConverter) script(script *Script) proto.Script {
	if c.err != nil {
		return nil
	}
	if script == nil {
		return nil
	}
	return proto.Script(script.Bytes)
}

func (c *SafeTransactionConverter) alias(scheme byte, alias string) proto.Alias {
	if c.err != nil {
		return proto.Alias{}
	}
	a := proto.NewAlias(scheme, alias)
	_, err := a.Valid()
	if err != nil {
		c.err = err
		return proto.Alias{}
	}
	return *a
}

func (c *SafeTransactionConverter) recipient(scheme byte, recipient *Recipient) proto.Recipient {
	if c.err != nil {
		return proto.Recipient{}
	}
	if recipient == nil {
		c.err = errors.New("empty recipient")
		return proto.Recipient{}
	}
	switch r := recipient.Recipient.(type) {
	case *Recipient_Address:
		return proto.NewRecipientFromAddress(c.address(scheme, r.Address))
	case *Recipient_Alias:
		return proto.NewRecipientFromAlias(c.alias(scheme, r.Alias))
	default:
		c.err = errors.New("invalid recipient")
		return proto.Recipient{}
	}
}

func (c *SafeTransactionConverter) assetPair(pair *ExchangeTransactionData_Order_AssetPair) proto.AssetPair {
	if c.err != nil {
		return proto.AssetPair{}
	}
	return proto.AssetPair{
		AmountAsset: c.optionalAsset(pair.AmountAssetId),
		PriceAsset:  c.optionalAsset(pair.PriceAssetId),
	}
}

func (c *SafeTransactionConverter) orderType(side ExchangeTransactionData_Order_Side) proto.OrderType {
	return proto.OrderType(c.byte(int32(side)))
}

func (c *SafeTransactionConverter) proofs(proofs [][]byte) *proto.ProofsV1 {
	if c.err != nil {
		return nil
	}
	r := proto.NewProofs()
	for _, proof := range proofs {
		r.Proofs = append(r.Proofs, proto.B58Bytes(proof))
	}
	return r
}

func (c *SafeTransactionConverter) signature(proofs [][]byte) *crypto.Signature {
	if c.err != nil {
		return nil
	}
	if len(proofs) < 1 {
		c.err = errors.New("empty proofs for signature")
		return nil
	}
	sig, err := crypto.NewSignatureFromBytes(proofs[0])
	if err != nil {
		c.err = err
		return nil
	}
	return &sig
}

func (c *SafeTransactionConverter) extractOrder(orders []*ExchangeTransactionData_Order, side ExchangeTransactionData_Order_Side) proto.Order {
	if c.err != nil {
		return nil
	}
	for _, o := range orders {
		if o.OrderSide == side {
			var order proto.Order
			body := proto.OrderBody{
				SenderPK:   c.publicKey(o.SenderPublicKey),
				MatcherPK:  c.publicKey(o.MatcherPublicKey),
				AssetPair:  c.assetPair(o.AssetPair),
				OrderType:  c.orderType(o.OrderSide),
				Price:      c.uint64(o.Price),
				Amount:     c.uint64(o.Amount),
				Timestamp:  c.uint64(o.Timestamp),
				Expiration: c.uint64(o.Expiration),
				MatcherFee: c.amount(o.MatcherFee), //TODO: support oder fee asset for OrderV3
			}
			switch o.Version {
			case 3:
				//TODO: support order version 3
			case 2:
				order = proto.OrderV2{
					Version:   c.byte(o.Version),
					Proofs:    c.proofs(o.Proofs),
					OrderBody: body,
				}
			default:
				order = proto.OrderV1{
					Signature: c.signature(o.Proofs),
					OrderBody: body,
				}
			}
			return order
		}
	}
	c.err = errors.Errorf("no order of side %s", side.String())
	return nil
}

func (c *SafeTransactionConverter) buyOrder(orders []*ExchangeTransactionData_Order) proto.Order {
	return c.extractOrder(orders, ExchangeTransactionData_Order_BUY)
}

func (c *SafeTransactionConverter) sellOrder(orders []*ExchangeTransactionData_Order) proto.Order {
	return c.extractOrder(orders, ExchangeTransactionData_Order_SELL)
}

func (c *SafeTransactionConverter) transfers(scheme byte, transfers []*MassTransferTransactionData_Transfer) []proto.MassTransferEntry {
	if c.err != nil {
		return nil
	}
	r := make([]proto.MassTransferEntry, len(transfers))
	for i, tr := range transfers {
		if tr == nil {
			c.err = errors.New("empty transfer")
			return nil
		}
		e := proto.MassTransferEntry{
			Recipient: c.recipient(scheme, tr.Address),
			Amount:    c.uint64(tr.Amount),
		}
		if c.err != nil {
			return nil
		}
		r[i] = e
	}
	return r
}

func (c *SafeTransactionConverter) entries(entries []*DataTransactionData_DataEntry) proto.DataEntries {
	if c.err != nil {
		return nil
	}
	r := make([]proto.DataEntry, len(entries))
	for i, e := range entries {
		if e == nil {
			c.err = errors.New("empty data entry")
			return nil
		}
		var entry proto.DataEntry
		switch t := e.Value.(type) {
		case *DataTransactionData_DataEntry_IntValue:
			entry = proto.IntegerDataEntry{Key: e.Key, Value: t.IntValue}
		case *DataTransactionData_DataEntry_BoolValue:
			entry = proto.BooleanDataEntry{Key: e.Key, Value: t.BoolValue}
		case *DataTransactionData_DataEntry_BinaryValue:
			entry = proto.BinaryDataEntry{Key: e.Key, Value: t.BinaryValue}
		case *DataTransactionData_DataEntry_StringValue:
			entry = proto.StringDataEntry{Key: e.Key, Value: t.StringValue}
		}
		r[i] = entry
	}
	return r
}

func (c *SafeTransactionConverter) functionCall(data []byte) proto.FunctionCall {
	panic("not implemented")
}

func (c *SafeTransactionConverter) payments(payments []*Amount) proto.ScriptPayments {
	if payments == nil {
		return proto.ScriptPayments(nil)
	}
	result := make([]proto.ScriptPayment, len(payments))
	for i, p := range payments {
		asset, amount := c.convertAmount(p)
		result[i] = proto.ScriptPayment{Asset: asset, Amount: amount}
	}
	return result
}

func (c *SafeTransactionConverter) ConvertTransaction(tx Transaction) (proto.Transaction, error) {
	ts := c.uint64(tx.Timestamp)
	scheme := c.byte(tx.ChainId)
	v := c.byte(tx.Version)
	if c.err != nil {
		return nil, c.err
	}
	switch d := tx.Data.(type) {
	case *Transaction_Genesis:
		return &proto.Genesis{
			Type:      proto.GenesisTransaction,
			Version:   v,
			Timestamp: ts,
			Recipient: c.address(scheme, d.Genesis.RecipientAddress),
			Amount:    uint64(d.Genesis.Amount),
		}, c.err

	case *Transaction_Payment:
		return &proto.Payment{
			Type:      proto.PaymentTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Recipient: c.address(scheme, d.Payment.RecipientAddress),
			Amount:    c.uint64(d.Payment.Amount),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}, nil

	case *Transaction_Issue:
		pi := proto.Issue{
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			Name:        c.string(d.Issue.Name),
			Description: c.string(d.Issue.Description),
			Quantity:    c.uint64(d.Issue.Amount),
			Decimals:    c.byte(d.Issue.Decimals),
			Reissuable:  d.Issue.Reissuable,
			Timestamp:   ts,
			Fee:         c.amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			return &proto.IssueV2{
				Type:    proto.IssueTransaction,
				Version: v,
				ChainID: scheme,
				Script:  c.script(d.Issue.Script),
				Issue:   pi,
			}, nil
		default:
			return &proto.IssueV1{
				Type:    proto.IssueTransaction,
				Version: v,
				Issue:   pi,
			}, nil
		}

	case *Transaction_Transfer:
		aa, amount := c.convertAmount(d.Transfer.Amount)
		fa, fee := c.convertAmount(tx.Fee)
		pt := proto.Transfer{
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AmountAsset: aa,
			FeeAsset:    fa,
			Timestamp:   ts,
			Amount:      amount,
			Fee:         fee,
			Recipient:   c.recipient(scheme, d.Transfer.Recipient),
			Attachment:  proto.Attachment(c.string(d.Transfer.Attachment)),
		}
		switch tx.Version {
		case 2:
			return &proto.TransferV2{
				Type:     proto.TransferTransaction,
				Version:  v,
				Transfer: pt,
			}, nil
		default:
			return &proto.TransferV1{
				Type:     proto.TransferTransaction,
				Version:  v,
				Transfer: pt,
			}, nil
		}

	case *Transaction_Reissue:
		id, quantity := c.convertAssetAmount(d.Reissue.AssetAmount)
		pr := proto.Reissue{
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			AssetID:    id,
			Quantity:   quantity,
			Reissuable: d.Reissue.Reissuable,
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			return &proto.ReissueV2{
				Type:    proto.ReissueTransaction,
				Version: v,
				ChainID: scheme,
				Reissue: pr,
			}, nil
		default:
			return &proto.ReissueV1{
				Type:    proto.ReissueTransaction,
				Version: v,
				Reissue: pr,
			}, nil
		}

	case *Transaction_Burn:
		id, amount := c.convertAssetAmount(d.Burn.AssetAmount)
		pb := proto.Burn{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			AssetID:   id,
			Amount:    amount,
			Timestamp: ts,
			Fee:       c.amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			return &proto.BurnV2{
				Type:    proto.BurnTransaction,
				Version: v,
				ChainID: scheme,
				Burn:    pb,
			}, nil
		default:
			return &proto.BurnV1{
				Type:    proto.BurnTransaction,
				Version: v,
				Burn:    pb,
			}, nil
		}

	case *Transaction_Exchange:
		fee := c.amount(tx.Fee)
		bo := c.buyOrder(d.Exchange.Orders)
		so := c.sellOrder(d.Exchange.Orders)
		switch tx.Version {
		case 2:
			return &proto.ExchangeV2{
				Type:           proto.ExchangeTransaction,
				Version:        v,
				SenderPK:       c.publicKey(tx.SenderPublicKey),
				BuyOrder:       bo,
				SellOrder:      so,
				Price:          c.uint64(d.Exchange.Price),
				Amount:         c.uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}, nil
		default:
			if bo.GetVersion() != 1 || so.GetVersion() != 1 {
				return nil, errors.New("unsupported order version")
			}
			return &proto.ExchangeV1{
				Type:           proto.ExchangeTransaction,
				Version:        v,
				SenderPK:       c.publicKey(tx.SenderPublicKey),
				BuyOrder:       bo.(proto.OrderV1),
				SellOrder:      so.(proto.OrderV1),
				Price:          c.uint64(d.Exchange.Price),
				Amount:         c.uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}, nil
		}

	case *Transaction_Lease:
		pl := proto.Lease{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Recipient: c.recipient(scheme, d.Lease.Recipient),
			Amount:    c.uint64(d.Lease.Amount),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			return &proto.LeaseV2{
				Type:    proto.LeaseTransaction,
				Version: v,
				Lease:   pl,
			}, nil
		default:
			return &proto.LeaseV1{
				Type:    proto.LeaseTransaction,
				Version: v,
				Lease:   pl,
			}, nil
		}

	case *Transaction_LeaseCancel:
		plc := proto.LeaseCancel{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			LeaseID:   c.digest(d.LeaseCancel.LeaseId),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			return &proto.LeaseCancelV2{
				Type:        proto.LeaseCancelTransaction,
				Version:     v,
				ChainID:     scheme,
				LeaseCancel: plc,
			}, nil
		default:
			return &proto.LeaseCancelV1{
				Type:        proto.LeaseCancelTransaction,
				Version:     v,
				LeaseCancel: plc,
			}, nil
		}

	case *Transaction_CreateAlias:
		pca := proto.CreateAlias{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Alias:     c.alias(scheme, d.CreateAlias.Alias),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			return &proto.CreateAliasV2{
				Type:        proto.CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}, nil
		default:
			return &proto.CreateAliasV1{
				Type:        proto.CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}, nil
		}

	case *Transaction_MassTransfer:
		return &proto.MassTransferV1{
			Type:       proto.MassTransferTransaction,
			Version:    v,
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			Asset:      c.optionalAsset(d.MassTransfer.AssetId),
			Transfers:  c.transfers(scheme, d.MassTransfer.Transfers),
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
			Attachment: proto.Attachment(c.string(d.MassTransfer.Attachment)),
		}, nil

	case *Transaction_DataTransaction:
		return &proto.DataV1{
			Type:      proto.DataTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Entries:   c.entries(d.DataTransaction.Data),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}, nil

	case *Transaction_SetScript:
		return &proto.SetScriptV1{
			Type:      proto.SetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Script:    c.script(d.SetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}, nil

	case *Transaction_SponsorFee:
		asset, amount := c.convertAssetAmount(d.SponsorFee.MinFee)
		return &proto.SponsorshipV1{
			Type:        proto.SponsorshipTransaction,
			Version:     v,
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AssetID:     asset,
			MinAssetFee: amount,
			Fee:         c.amount(tx.Fee),
			Timestamp:   ts,
		}, nil

	case *Transaction_SetAssetScript:
		return &proto.SetAssetScriptV1{
			Type:      proto.SetAssetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			AssetID:   c.digest(d.SetAssetScript.AssetId),
			Script:    c.script(d.SetAssetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}, nil

	case *Transaction_InvokeScript:
		feeAsset, feeAmount := c.convertAmount(tx.Fee)
		dAppAddr := c.address(scheme, d.InvokeScript.DappAddress)
		return &proto.InvokeScriptV1{
			Type:            proto.InvokeScriptTransaction,
			Version:         v,
			ChainID:         scheme,
			SenderPK:        c.publicKey(tx.SenderPublicKey),
			ScriptRecipient: proto.NewRecipientFromAddress(dAppAddr),
			FunctionCall:    c.functionCall(d.InvokeScript.FunctionCall),
			Payments:        c.payments(d.InvokeScript.Payments),
			FeeAsset:        feeAsset,
			Fee:             feeAmount,
			Timestamp:       ts,
		}, nil

	}
	return nil, errors.New("unsupported transaction")
}
