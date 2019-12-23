package client

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SafeConverter struct {
	err error
}

func (c *SafeConverter) Error() error {
	return c.err
}

func (c *SafeConverter) Address(scheme byte, addr []byte) proto.Address {
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

func (c *SafeConverter) Uint64(value int64) uint64 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("negative int64 value")
		return 0
	}
	return uint64(value)
}

func (c *SafeConverter) Byte(value int32) byte {
	if c.err != nil {
		return 0
	}
	if value < 0 || value > 0xff {
		c.err = errors.New("invalid byte value")
	}
	return byte(value)
}

func (c *SafeConverter) Digest(digest []byte) crypto.Digest {
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

func (c *SafeConverter) OptionalAsset(asset []byte) proto.OptionalAsset {
	if c.err != nil {
		return proto.OptionalAsset{}
	}
	if len(asset) == 0 {
		return proto.OptionalAsset{}
	}
	return proto.OptionalAsset{Present: true, ID: c.Digest(asset)}
}

func (c *SafeConverter) ConvertAmount(amount *g.Amount) (proto.OptionalAsset, uint64) {
	if c.err != nil {
		return proto.OptionalAsset{}, 0
	}
	return c.ExtractOptionalAsset(amount), c.Amount(amount)
}

func (c *SafeConverter) ConvertAssetAmount(aa *g.Amount) (crypto.Digest, uint64) {
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
	return id, c.Uint64(aa.Amount)
}

func (c *SafeConverter) ExtractOptionalAsset(amount *g.Amount) proto.OptionalAsset {
	if c.err != nil {
		return proto.OptionalAsset{}
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return proto.OptionalAsset{}
	}
	return c.OptionalAsset(amount.AssetId)
}

func (c *SafeConverter) Amount(amount *g.Amount) uint64 {
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

func (c *SafeConverter) PublicKey(pk []byte) crypto.PublicKey {
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

func (c *SafeConverter) String(bytes []byte) string {
	if c.err != nil {
		return ""
	}
	return string(bytes)
}

func (c *SafeConverter) Script(script *g.Script) proto.Script {
	if c.err != nil {
		return nil
	}
	if script == nil {
		return nil
	}
	return proto.Script(script.Bytes)
}

func (c *SafeConverter) Alias(scheme byte, alias string) proto.Alias {
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

func (c *SafeConverter) Recipient(scheme byte, recipient *g.Recipient) proto.Recipient {
	if c.err != nil {
		return proto.Recipient{}
	}
	if recipient == nil {
		c.err = errors.New("empty recipient")
		return proto.Recipient{}
	}
	switch r := recipient.Recipient.(type) {
	case *g.Recipient_Address:
		return proto.NewRecipientFromAddress(c.Address(scheme, r.Address))
	case *g.Recipient_Alias:
		return proto.NewRecipientFromAlias(c.Alias(scheme, r.Alias))
	default:
		c.err = errors.New("invalid recipient")
		return proto.Recipient{}
	}
}

func (c *SafeConverter) AssetPair(pair *g.AssetPair) proto.AssetPair {
	if c.err != nil {
		return proto.AssetPair{}
	}
	return proto.AssetPair{
		AmountAsset: c.OptionalAsset(pair.AmountAssetId),
		PriceAsset:  c.OptionalAsset(pair.PriceAssetId),
	}
}

func (c *SafeConverter) OrderType(side g.Order_Side) proto.OrderType {
	return proto.OrderType(c.Byte(int32(side)))
}

func (c *SafeConverter) Proofs(proofs [][]byte) *proto.ProofsV1 {
	if c.err != nil {
		return nil
	}
	r := proto.NewProofs()
	for _, proof := range proofs {
		r.Proofs = append(r.Proofs, proto.B58Bytes(proof))
	}
	return r
}

func (c *SafeConverter) Proof(proofs [][]byte) *crypto.Signature {
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

func (c *SafeConverter) Signature(data []byte) crypto.Signature {
	if c.err != nil {
		return crypto.Signature{}
	}
	sig, err := crypto.NewSignatureFromBytes(data)
	if err != nil {
		c.err = err
		return crypto.Signature{}
	}
	return sig
}

func (c *SafeConverter) ExtractOrder(orders []*g.Order, side g.Order_Side) proto.Order {
	if c.err != nil {
		return nil
	}
	for _, o := range orders {
		if o.OrderSide == side {
			var order proto.Order
			body := proto.OrderBody{
				SenderPK:   c.PublicKey(o.SenderPublicKey),
				MatcherPK:  c.PublicKey(o.MatcherPublicKey),
				AssetPair:  c.AssetPair(o.AssetPair),
				OrderType:  c.OrderType(o.OrderSide),
				Price:      c.Uint64(o.Price),
				Amount:     c.Uint64(o.Amount),
				Timestamp:  c.Uint64(o.Timestamp),
				Expiration: c.Uint64(o.Expiration),
				MatcherFee: c.Amount(o.MatcherFee),
			}
			switch o.Version {
			case 3:
				order = &proto.OrderV3{
					Version:         c.Byte(o.Version),
					Proofs:          c.Proofs(o.Proofs),
					OrderBody:       body,
					MatcherFeeAsset: c.ExtractOptionalAsset(o.MatcherFee),
				}
			case 2:
				order = &proto.OrderV2{
					Version:   c.Byte(o.Version),
					Proofs:    c.Proofs(o.Proofs),
					OrderBody: body,
				}
			default:
				order = &proto.OrderV1{
					Signature: c.Proof(o.Proofs),
					OrderBody: body,
				}
			}
			return order
		}
	}
	c.err = errors.Errorf("no order of side %s", side.String())
	return nil
}

func (c *SafeConverter) BuyOrder(orders []*g.Order) proto.Order {
	return c.ExtractOrder(orders, g.Order_BUY)
}

func (c *SafeConverter) SellOrder(orders []*g.Order) proto.Order {
	return c.ExtractOrder(orders, g.Order_SELL)
}

func (c *SafeConverter) Transfers(scheme byte, transfers []*g.MassTransferTransactionData_Transfer) []proto.MassTransferEntry {
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
			Recipient: c.Recipient(scheme, tr.Address),
			Amount:    c.Uint64(tr.Amount),
		}
		if c.err != nil {
			return nil
		}
		r[i] = e
	}
	return r
}

func (c *SafeConverter) entry(entry *g.DataTransactionData_DataEntry) proto.DataEntry {
	if c.err != nil {
		return nil
	}
	if entry == nil {
		c.err = errors.New("empty data entry")
		return nil
	}
	var e proto.DataEntry
	switch t := entry.Value.(type) {
	case *g.DataTransactionData_DataEntry_IntValue:
		e = &proto.IntegerDataEntry{Key: entry.Key, Value: t.IntValue}
	case *g.DataTransactionData_DataEntry_BoolValue:
		e = &proto.BooleanDataEntry{Key: entry.Key, Value: t.BoolValue}
	case *g.DataTransactionData_DataEntry_BinaryValue:
		e = &proto.BinaryDataEntry{Key: entry.Key, Value: t.BinaryValue}
	case *g.DataTransactionData_DataEntry_StringValue:
		e = &proto.StringDataEntry{Key: entry.Key, Value: t.StringValue}
	}
	return e
}

func (c *SafeConverter) Entry(entry *g.DataTransactionData_DataEntry) (proto.DataEntry, error) {
	e := c.entry(entry)
	if c.err != nil {
		return nil, c.err
	}
	return e, nil
}

func (c *SafeConverter) Entries(entries []*g.DataTransactionData_DataEntry) proto.DataEntries {
	if c.err != nil {
		return nil
	}
	r := make([]proto.DataEntry, len(entries))
	for i, e := range entries {
		r[i] = c.entry(e)
	}
	return r
}

func (c *SafeConverter) FunctionCall(data []byte) proto.FunctionCall {
	if c.err != nil {
		return proto.FunctionCall{}
	}
	fc := proto.FunctionCall{}
	err := fc.UnmarshalBinary(data)
	if err != nil {
		c.err = err
		return proto.FunctionCall{}
	}
	return fc
}

func (c *SafeConverter) Payments(payments []*g.Amount) proto.ScriptPayments {
	if payments == nil {
		return proto.ScriptPayments(nil)
	}
	result := make([]proto.ScriptPayment, len(payments))
	for i, p := range payments {
		asset, amount := c.ConvertAmount(p)
		result[i] = proto.ScriptPayment{Asset: asset, Amount: amount}
	}
	return result
}

func (c *SafeConverter) Reset() {
	c.err = nil
}

func (c *SafeConverter) Transaction(tx *g.Transaction) (proto.Transaction, error) {
	ts := c.Uint64(tx.Timestamp)
	scheme := c.Byte(tx.ChainId)
	v := c.Byte(tx.Version)
	if c.err != nil {
		return nil, c.err
	}
	var rtx proto.Transaction
	switch d := tx.Data.(type) {
	case *g.Transaction_Genesis:
		rtx = &proto.Genesis{
			Type:      proto.GenesisTransaction,
			Version:   v,
			Timestamp: ts,
			Recipient: c.Address(scheme, d.Genesis.RecipientAddress),
			Amount:    uint64(d.Genesis.Amount),
		}

	case *g.Transaction_Payment:
		rtx = &proto.Payment{
			Type:      proto.PaymentTransaction,
			Version:   v,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Recipient: c.Address(scheme, d.Payment.RecipientAddress),
			Amount:    c.Uint64(d.Payment.Amount),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_Issue:
		pi := proto.Issue{
			SenderPK:    c.PublicKey(tx.SenderPublicKey),
			Name:        c.String(d.Issue.Name),
			Description: c.String(d.Issue.Description),
			Quantity:    c.Uint64(d.Issue.Amount),
			Decimals:    c.Byte(d.Issue.Decimals),
			Reissuable:  d.Issue.Reissuable,
			Timestamp:   ts,
			Fee:         c.Amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			rtx = &proto.IssueV2{
				Type:    proto.IssueTransaction,
				Version: v,
				ChainID: scheme,
				Script:  c.Script(d.Issue.Script),
				Issue:   pi,
			}
		default:
			rtx = &proto.IssueV1{
				Type:    proto.IssueTransaction,
				Version: v,
				Issue:   pi,
			}
		}

	case *g.Transaction_Transfer:
		aa, amount := c.ConvertAmount(d.Transfer.Amount)
		fa, fee := c.ConvertAmount(tx.Fee)
		pt := proto.Transfer{
			SenderPK:    c.PublicKey(tx.SenderPublicKey),
			AmountAsset: aa,
			FeeAsset:    fa,
			Timestamp:   ts,
			Amount:      amount,
			Fee:         fee,
			Recipient:   c.Recipient(scheme, d.Transfer.Recipient),
			Attachment:  proto.Attachment(c.String(d.Transfer.Attachment)),
		}
		switch tx.Version {
		case 2:
			rtx = &proto.TransferV2{
				Type:     proto.TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		default:
			rtx = &proto.TransferV1{
				Type:     proto.TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		}

	case *g.Transaction_Reissue:
		id, quantity := c.ConvertAssetAmount(d.Reissue.AssetAmount)
		pr := proto.Reissue{
			SenderPK:   c.PublicKey(tx.SenderPublicKey),
			AssetID:    id,
			Quantity:   quantity,
			Reissuable: d.Reissue.Reissuable,
			Timestamp:  ts,
			Fee:        c.Amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			rtx = &proto.ReissueV2{
				Type:    proto.ReissueTransaction,
				Version: v,
				ChainID: scheme,
				Reissue: pr,
			}
		default:
			rtx = &proto.ReissueV1{
				Type:    proto.ReissueTransaction,
				Version: v,
				Reissue: pr,
			}
		}

	case *g.Transaction_Burn:
		id, amount := c.ConvertAssetAmount(d.Burn.AssetAmount)
		pb := proto.Burn{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			AssetID:   id,
			Amount:    amount,
			Timestamp: ts,
			Fee:       c.Amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			rtx = &proto.BurnV2{
				Type:    proto.BurnTransaction,
				Version: v,
				ChainID: scheme,
				Burn:    pb,
			}
		default:
			rtx = &proto.BurnV1{
				Type:    proto.BurnTransaction,
				Version: v,
				Burn:    pb,
			}
		}

	case *g.Transaction_Exchange:
		fee := c.Amount(tx.Fee)
		bo := c.BuyOrder(d.Exchange.Orders)
		so := c.SellOrder(d.Exchange.Orders)
		switch tx.Version {
		case 2:
			rtx = &proto.ExchangeV2{
				Type:           proto.ExchangeTransaction,
				Version:        v,
				SenderPK:       c.PublicKey(tx.SenderPublicKey),
				BuyOrder:       bo,
				SellOrder:      so,
				Price:          c.Uint64(d.Exchange.Price),
				Amount:         c.Uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.Uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.Uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}
		default:
			if bo.GetVersion() != 1 || so.GetVersion() != 1 {
				return nil, errors.New("unsupported order version")
			}
			bo1, ok := bo.(*proto.OrderV1)
			if !ok {
				return nil, errors.New("invalid pointer to OrderV1")
			}
			so1, ok := so.(*proto.OrderV1)
			if !ok {
				return nil, errors.New("invalid pointer to OrderV1")
			}

			rtx = &proto.ExchangeV1{
				Type:           proto.ExchangeTransaction,
				Version:        v,
				SenderPK:       c.PublicKey(tx.SenderPublicKey),
				BuyOrder:       bo1,
				SellOrder:      so1,
				Price:          c.Uint64(d.Exchange.Price),
				Amount:         c.Uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.Uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.Uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}
		}

	case *g.Transaction_Lease:
		pl := proto.Lease{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Recipient: c.Recipient(scheme, d.Lease.Recipient),
			Amount:    c.Uint64(d.Lease.Amount),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &proto.LeaseV2{
				Type:    proto.LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		default:
			rtx = &proto.LeaseV1{
				Type:    proto.LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		}

	case *g.Transaction_LeaseCancel:
		plc := proto.LeaseCancel{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			LeaseID:   c.Digest(d.LeaseCancel.LeaseId),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &proto.LeaseCancelV2{
				Type:        proto.LeaseCancelTransaction,
				Version:     v,
				ChainID:     scheme,
				LeaseCancel: plc,
			}
		default:
			rtx = &proto.LeaseCancelV1{
				Type:        proto.LeaseCancelTransaction,
				Version:     v,
				LeaseCancel: plc,
			}
		}

	case *g.Transaction_CreateAlias:
		pca := proto.CreateAlias{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Alias:     c.Alias(scheme, d.CreateAlias.Alias),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &proto.CreateAliasV2{
				Type:        proto.CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		default:
			rtx = &proto.CreateAliasV1{
				Type:        proto.CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		}

	case *g.Transaction_MassTransfer:
		rtx = &proto.MassTransferV1{
			Type:       proto.MassTransferTransaction,
			Version:    v,
			SenderPK:   c.PublicKey(tx.SenderPublicKey),
			Asset:      c.OptionalAsset(d.MassTransfer.AssetId),
			Transfers:  c.Transfers(scheme, d.MassTransfer.Transfers),
			Timestamp:  ts,
			Fee:        c.Amount(tx.Fee),
			Attachment: proto.Attachment(c.String(d.MassTransfer.Attachment)),
		}

	case *g.Transaction_DataTransaction:
		rtx = &proto.DataV1{
			Type:      proto.DataTransaction,
			Version:   v,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Entries:   c.Entries(d.DataTransaction.Data),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SetScript:
		rtx = &proto.SetScriptV1{
			Type:      proto.SetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Script:    c.Script(d.SetScript.Script),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SponsorFee:
		asset, amount := c.ConvertAssetAmount(d.SponsorFee.MinFee)
		rtx = &proto.SponsorshipV1{
			Type:        proto.SponsorshipTransaction,
			Version:     v,
			SenderPK:    c.PublicKey(tx.SenderPublicKey),
			AssetID:     asset,
			MinAssetFee: amount,
			Fee:         c.Amount(tx.Fee),
			Timestamp:   ts,
		}

	case *g.Transaction_SetAssetScript:
		rtx = &proto.SetAssetScriptV1{
			Type:      proto.SetAssetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			AssetID:   c.Digest(d.SetAssetScript.AssetId),
			Script:    c.Script(d.SetAssetScript.Script),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_InvokeScript:
		feeAsset, feeAmount := c.ConvertAmount(tx.Fee)
		rtx = &proto.InvokeScriptV1{
			Type:            proto.InvokeScriptTransaction,
			Version:         v,
			ChainID:         scheme,
			SenderPK:        c.PublicKey(tx.SenderPublicKey),
			ScriptRecipient: c.Recipient(scheme, d.InvokeScript.DApp),
			FunctionCall:    c.FunctionCall(d.InvokeScript.FunctionCall),
			Payments:        c.Payments(d.InvokeScript.Payments),
			FeeAsset:        feeAsset,
			Fee:             feeAmount,
			Timestamp:       ts,
		}
	default:
		return nil, errors.New("unsupported transaction")
	}
	rtx.GenerateID()
	return rtx, nil
}

func (c *SafeConverter) ExtractFirstSignature(proofs *proto.ProofsV1) *crypto.Signature {
	if c.err != nil {
		return nil
	}
	if len(proofs.Proofs) == 0 {
		c.err = errors.New("unable to extract Signature from empty ProofsV1")
		return nil
	}
	s, err := crypto.NewSignatureFromBytes(proofs.Proofs[0])
	if err != nil {
		c.err = err
		return nil
	}
	return &s
}

func (c *SafeConverter) SignedTransaction(stx *g.SignedTransaction) (proto.Transaction, error) {
	tx, err := c.Transaction(stx.Transaction)
	if err != nil {
		return nil, err
	}
	proofs := c.Proofs(stx.Proofs)
	if c.err != nil {
		return nil, c.err
	}
	switch t := tx.(type) {
	case *proto.Genesis:
		sig := c.ExtractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *proto.Payment:
		sig := c.ExtractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *proto.IssueV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.IssueV2:
		t.Proofs = proofs
		return t, nil
	case *proto.TransferV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.TransferV2:
		t.Proofs = proofs
		return t, nil
	case *proto.ReissueV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.ReissueV2:
		t.Proofs = proofs
		return t, nil
	case *proto.BurnV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.BurnV2:
		t.Proofs = proofs
		return t, nil
	case *proto.ExchangeV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.ExchangeV2:
		t.Proofs = proofs
		return t, nil
	case *proto.LeaseV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.LeaseV2:
		t.Proofs = proofs
		return t, nil
	case *proto.LeaseCancelV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.LeaseCancelV2:
		t.Proofs = proofs
		return t, nil
	case *proto.CreateAliasV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *proto.CreateAliasV2:
		t.Proofs = proofs
		return t, nil
	case *proto.MassTransferV1:
		t.Proofs = proofs
		return t, nil
	case *proto.DataV1:
		t.Proofs = proofs
		return t, nil
	case *proto.SetScriptV1:
		t.Proofs = proofs
		return t, nil
	case *proto.SponsorshipV1:
		t.Proofs = proofs
		return t, nil
	case *proto.SetAssetScriptV1:
		t.Proofs = proofs
		return t, nil
	case *proto.InvokeScriptV1:
		t.Proofs = proofs
		return t, nil
	default:
		panic("unsupported transaction")
	}
}

func (c *SafeConverter) BlockTransactions(block *g.BlockWithHeight) ([]proto.Transaction, error) {
	if c.err != nil {
		return nil, c.err
	}
	txs := make([]proto.Transaction, len(block.Block.Transactions))
	for i, stx := range block.Block.Transactions {
		tx, err := c.SignedTransaction(stx)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}

func (c *SafeConverter) Features(features []uint32) []int16 {
	r := make([]int16, len(features))
	for i, f := range features {
		r[i] = int16(f)
	}
	return r
}

func (c *SafeConverter) Consensus(header *g.Block_Header) proto.NxtConsensus {
	if c.err != nil {
		return proto.NxtConsensus{}
	}
	return proto.NxtConsensus{
		GenSignature: c.Digest(header.GenerationSignature),
		BaseTarget:   c.Uint64(header.BaseTarget),
	}
}

func (c *SafeConverter) BlockHeader(block *g.BlockWithHeight) (proto.BlockHeader, error) {
	if c.err != nil {
		return proto.BlockHeader{}, c.err
	}
	features := c.Features(block.Block.Header.FeatureVotes)
	if c.err != nil {
		return proto.BlockHeader{}, c.err
	}
	return proto.BlockHeader{
		Version:          proto.BlockVersion(c.Byte(block.Block.Header.Version)),
		Timestamp:        c.Uint64(block.Block.Header.Timestamp),
		Parent:           c.Signature(block.Block.Header.Reference),
		FeaturesCount:    len(features),
		Features:         features,
		RewardVote:       block.Block.Header.RewardVote,
		NxtConsensus:     c.Consensus(block.Block.Header),
		TransactionCount: len(block.Block.Transactions),
		GenPublicKey:     c.PublicKey(block.Block.Header.Generator),
		BlockSignature:   c.Signature(block.Block.Signature),
	}, nil
}
