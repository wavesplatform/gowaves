package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
)

type ProtobufConverter struct {
	err error
}

func (c *ProtobufConverter) Error() error {
	return c.err
}

func (c *ProtobufConverter) Address(scheme byte, addr []byte) Address {
	if c.err != nil {
		return Address{}
	}
	a, err := RebuildAddress(scheme, addr)
	if err != nil {
		c.err = err
		return Address{}
	}
	return a
}

func (c *ProtobufConverter) Uint64(value int64) uint64 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("negative int64 value")
		return 0
	}
	return uint64(value)
}

func (c *ProtobufConverter) Byte(value int32) byte {
	if c.err != nil {
		return 0
	}
	if value < 0 || value > 0xff {
		c.err = errors.New("invalid byte value")
	}
	return byte(value)
}

func (c *ProtobufConverter) Digest(digest []byte) crypto.Digest {
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

func (c *ProtobufConverter) OptionalAsset(asset []byte) OptionalAsset {
	if c.err != nil {
		return OptionalAsset{}
	}
	if len(asset) == 0 {
		return OptionalAsset{}
	}
	return OptionalAsset{Present: true, ID: c.Digest(asset)}
}

func (c *ProtobufConverter) ConvertAmount(amount *g.Amount) (OptionalAsset, uint64) {
	if c.err != nil {
		return OptionalAsset{}, 0
	}
	return c.ExtractOptionalAsset(amount), c.Amount(amount)
}

func (c *ProtobufConverter) ConvertAssetAmount(aa *g.Amount) (crypto.Digest, uint64) {
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

func (c *ProtobufConverter) ExtractOptionalAsset(amount *g.Amount) OptionalAsset {
	if c.err != nil {
		return OptionalAsset{}
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return OptionalAsset{}
	}
	return c.OptionalAsset(amount.AssetId)
}

func (c *ProtobufConverter) Amount(amount *g.Amount) uint64 {
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

func (c *ProtobufConverter) PublicKey(pk []byte) crypto.PublicKey {
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

func (c *ProtobufConverter) String(bytes []byte) string {
	if c.err != nil {
		return ""
	}
	return string(bytes)
}

func (c *ProtobufConverter) Script(script *g.Script) Script {
	if c.err != nil {
		return nil
	}
	if script == nil {
		return nil
	}
	return Script(script.Bytes)
}

func (c *ProtobufConverter) Alias(scheme byte, alias string) Alias {
	if c.err != nil {
		return Alias{}
	}
	a := NewAlias(scheme, alias)
	_, err := a.Valid()
	if err != nil {
		c.err = err
		return Alias{}
	}
	return *a
}

func (c *ProtobufConverter) Recipient(scheme byte, recipient *g.Recipient) Recipient {
	if c.err != nil {
		return Recipient{}
	}
	if recipient == nil {
		c.err = errors.New("empty recipient")
		return Recipient{}
	}
	switch r := recipient.Recipient.(type) {
	case *g.Recipient_Address:
		return NewRecipientFromAddress(c.Address(scheme, r.Address))
	case *g.Recipient_Alias:
		return NewRecipientFromAlias(c.Alias(scheme, r.Alias))
	default:
		c.err = errors.New("invalid recipient")
		return Recipient{}
	}
}

func (c *ProtobufConverter) AssetPair(pair *g.AssetPair) AssetPair {
	if c.err != nil {
		return AssetPair{}
	}
	return AssetPair{
		AmountAsset: c.OptionalAsset(pair.AmountAssetId),
		PriceAsset:  c.OptionalAsset(pair.PriceAssetId),
	}
}

func (c *ProtobufConverter) OrderType(side g.Order_Side) OrderType {
	return OrderType(c.Byte(int32(side)))
}

func (c *ProtobufConverter) Proofs(proofs [][]byte) *ProofsV1 {
	if c.err != nil {
		return nil
	}
	r := NewProofs()
	for _, proof := range proofs {
		r.Proofs = append(r.Proofs, B58Bytes(proof))
	}
	return r
}

func (c *ProtobufConverter) Proof(proofs [][]byte) *crypto.Signature {
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

func (c *ProtobufConverter) Signature(data []byte) crypto.Signature {
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

func (c *ProtobufConverter) ExtractOrder(orders []*g.Order, side g.Order_Side) Order {
	if c.err != nil {
		return nil
	}
	for _, o := range orders {
		if o.OrderSide == side {
			var order Order
			body := OrderBody{
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
				order = &OrderV3{
					Version:         c.Byte(o.Version),
					Proofs:          c.Proofs(o.Proofs),
					OrderBody:       body,
					MatcherFeeAsset: c.ExtractOptionalAsset(o.MatcherFee),
				}
			case 2:
				order = &OrderV2{
					Version:   c.Byte(o.Version),
					Proofs:    c.Proofs(o.Proofs),
					OrderBody: body,
				}
			default:
				order = &OrderV1{
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

func (c *ProtobufConverter) BuyOrder(orders []*g.Order) Order {
	return c.ExtractOrder(orders, g.Order_BUY)
}

func (c *ProtobufConverter) SellOrder(orders []*g.Order) Order {
	return c.ExtractOrder(orders, g.Order_SELL)
}

func (c *ProtobufConverter) Transfers(scheme byte, transfers []*g.MassTransferTransactionData_Transfer) []MassTransferEntry {
	if c.err != nil {
		return nil
	}
	r := make([]MassTransferEntry, len(transfers))
	for i, tr := range transfers {
		if tr == nil {
			c.err = errors.New("empty transfer")
			return nil
		}
		e := MassTransferEntry{
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

func (c *ProtobufConverter) entry(entry *g.DataTransactionData_DataEntry) DataEntry {
	if c.err != nil {
		return nil
	}
	if entry == nil {
		c.err = errors.New("empty data entry")
		return nil
	}
	var e DataEntry
	switch t := entry.Value.(type) {
	case *g.DataTransactionData_DataEntry_IntValue:
		e = &IntegerDataEntry{Key: entry.Key, Value: t.IntValue}
	case *g.DataTransactionData_DataEntry_BoolValue:
		e = &BooleanDataEntry{Key: entry.Key, Value: t.BoolValue}
	case *g.DataTransactionData_DataEntry_BinaryValue:
		e = &BinaryDataEntry{Key: entry.Key, Value: t.BinaryValue}
	case *g.DataTransactionData_DataEntry_StringValue:
		e = &StringDataEntry{Key: entry.Key, Value: t.StringValue}
	}
	return e
}

func (c *ProtobufConverter) Entry(entry *g.DataTransactionData_DataEntry) (DataEntry, error) {
	e := c.entry(entry)
	if c.err != nil {
		return nil, c.err
	}
	return e, nil
}

func (c *ProtobufConverter) Entries(entries []*g.DataTransactionData_DataEntry) DataEntries {
	if c.err != nil {
		return nil
	}
	r := make([]DataEntry, len(entries))
	for i, e := range entries {
		r[i] = c.entry(e)
	}
	return r
}

func (c *ProtobufConverter) FunctionCall(data []byte) FunctionCall {
	if c.err != nil {
		return FunctionCall{}
	}
	fc := FunctionCall{}
	err := fc.UnmarshalBinary(data)
	if err != nil {
		c.err = err
		return FunctionCall{}
	}
	return fc
}

func (c *ProtobufConverter) Payments(payments []*g.Amount) ScriptPayments {
	if payments == nil {
		return ScriptPayments(nil)
	}
	result := make([]ScriptPayment, len(payments))
	for i, p := range payments {
		asset, amount := c.ConvertAmount(p)
		result[i] = ScriptPayment{Asset: asset, Amount: amount}
	}
	return result
}

func (c *ProtobufConverter) Reset() {
	c.err = nil
}

func (c *ProtobufConverter) Transaction(tx *g.Transaction) (Transaction, error) {
	ts := c.Uint64(tx.Timestamp)
	scheme := c.Byte(tx.ChainId)
	v := c.Byte(tx.Version)
	if c.err != nil {
		return nil, c.err
	}
	var rtx Transaction
	switch d := tx.Data.(type) {
	case *g.Transaction_Genesis:
		rtx = &Genesis{
			Type:      GenesisTransaction,
			Version:   v,
			Timestamp: ts,
			Recipient: c.Address(scheme, d.Genesis.RecipientAddress),
			Amount:    uint64(d.Genesis.Amount),
		}

	case *g.Transaction_Payment:
		rtx = &Payment{
			Type:      PaymentTransaction,
			Version:   v,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Recipient: c.Address(scheme, d.Payment.RecipientAddress),
			Amount:    c.Uint64(d.Payment.Amount),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_Issue:
		pi := Issue{
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
			rtx = &IssueV2{
				Type:    IssueTransaction,
				Version: v,
				ChainID: scheme,
				Script:  c.Script(d.Issue.Script),
				Issue:   pi,
			}
		default:
			rtx = &IssueV1{
				Type:    IssueTransaction,
				Version: v,
				Issue:   pi,
			}
		}

	case *g.Transaction_Transfer:
		aa, amount := c.ConvertAmount(d.Transfer.Amount)
		fa, fee := c.ConvertAmount(tx.Fee)
		pt := Transfer{
			SenderPK:    c.PublicKey(tx.SenderPublicKey),
			AmountAsset: aa,
			FeeAsset:    fa,
			Timestamp:   ts,
			Amount:      amount,
			Fee:         fee,
			Recipient:   c.Recipient(scheme, d.Transfer.Recipient),
			Attachment:  Attachment(c.String(d.Transfer.Attachment)),
		}
		switch tx.Version {
		case 2:
			rtx = &TransferV2{
				Type:     TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		default:
			rtx = &TransferV1{
				Type:     TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		}

	case *g.Transaction_Reissue:
		id, quantity := c.ConvertAssetAmount(d.Reissue.AssetAmount)
		pr := Reissue{
			SenderPK:   c.PublicKey(tx.SenderPublicKey),
			AssetID:    id,
			Quantity:   quantity,
			Reissuable: d.Reissue.Reissuable,
			Timestamp:  ts,
			Fee:        c.Amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			rtx = &ReissueV2{
				Type:    ReissueTransaction,
				Version: v,
				ChainID: scheme,
				Reissue: pr,
			}
		default:
			rtx = &ReissueV1{
				Type:    ReissueTransaction,
				Version: v,
				Reissue: pr,
			}
		}

	case *g.Transaction_Burn:
		id, amount := c.ConvertAssetAmount(d.Burn.AssetAmount)
		pb := Burn{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			AssetID:   id,
			Amount:    amount,
			Timestamp: ts,
			Fee:       c.Amount(tx.Fee),
		}
		switch tx.Version {
		case 2:
			rtx = &BurnV2{
				Type:    BurnTransaction,
				Version: v,
				ChainID: scheme,
				Burn:    pb,
			}
		default:
			rtx = &BurnV1{
				Type:    BurnTransaction,
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
			rtx = &ExchangeV2{
				Type:           ExchangeTransaction,
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
			bo1, ok := bo.(*OrderV1)
			if !ok {
				return nil, errors.New("invalid pointer to OrderV1")
			}
			so1, ok := so.(*OrderV1)
			if !ok {
				return nil, errors.New("invalid pointer to OrderV1")
			}

			rtx = &ExchangeV1{
				Type:           ExchangeTransaction,
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
		pl := Lease{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Recipient: c.Recipient(scheme, d.Lease.Recipient),
			Amount:    c.Uint64(d.Lease.Amount),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &LeaseV2{
				Type:    LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		default:
			rtx = &LeaseV1{
				Type:    LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		}

	case *g.Transaction_LeaseCancel:
		plc := LeaseCancel{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			LeaseID:   c.Digest(d.LeaseCancel.LeaseId),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &LeaseCancelV2{
				Type:        LeaseCancelTransaction,
				Version:     v,
				ChainID:     scheme,
				LeaseCancel: plc,
			}
		default:
			rtx = &LeaseCancelV1{
				Type:        LeaseCancelTransaction,
				Version:     v,
				LeaseCancel: plc,
			}
		}

	case *g.Transaction_CreateAlias:
		pca := CreateAlias{
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Alias:     c.Alias(scheme, d.CreateAlias.Alias),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}
		switch tx.Version {
		case 2:
			rtx = &CreateAliasV2{
				Type:        CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		default:
			rtx = &CreateAliasV1{
				Type:        CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		}

	case *g.Transaction_MassTransfer:
		rtx = &MassTransferV1{
			Type:       MassTransferTransaction,
			Version:    v,
			SenderPK:   c.PublicKey(tx.SenderPublicKey),
			Asset:      c.OptionalAsset(d.MassTransfer.AssetId),
			Transfers:  c.Transfers(scheme, d.MassTransfer.Transfers),
			Timestamp:  ts,
			Fee:        c.Amount(tx.Fee),
			Attachment: Attachment(c.String(d.MassTransfer.Attachment)),
		}

	case *g.Transaction_DataTransaction:
		rtx = &DataV1{
			Type:      DataTransaction,
			Version:   v,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Entries:   c.Entries(d.DataTransaction.Data),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SetScript:
		rtx = &SetScriptV1{
			Type:      SetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.PublicKey(tx.SenderPublicKey),
			Script:    c.Script(d.SetScript.Script),
			Fee:       c.Amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SponsorFee:
		asset, amount := c.ConvertAssetAmount(d.SponsorFee.MinFee)
		rtx = &SponsorshipV1{
			Type:        SponsorshipTransaction,
			Version:     v,
			SenderPK:    c.PublicKey(tx.SenderPublicKey),
			AssetID:     asset,
			MinAssetFee: amount,
			Fee:         c.Amount(tx.Fee),
			Timestamp:   ts,
		}

	case *g.Transaction_SetAssetScript:
		rtx = &SetAssetScriptV1{
			Type:      SetAssetScriptTransaction,
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
		rtx = &InvokeScriptV1{
			Type:            InvokeScriptTransaction,
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

func (c *ProtobufConverter) ExtractFirstSignature(proofs *ProofsV1) *crypto.Signature {
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

func (c *ProtobufConverter) SignedTransaction(stx *g.SignedTransaction) (Transaction, error) {
	tx, err := c.Transaction(stx.Transaction)
	if err != nil {
		return nil, err
	}
	proofs := c.Proofs(stx.Proofs)
	if c.err != nil {
		return nil, c.err
	}
	switch t := tx.(type) {
	case *Genesis:
		sig := c.ExtractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *Payment:
		sig := c.ExtractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *IssueV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *IssueV2:
		t.Proofs = proofs
		return t, nil
	case *TransferV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *TransferV2:
		t.Proofs = proofs
		return t, nil
	case *ReissueV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *ReissueV2:
		t.Proofs = proofs
		return t, nil
	case *BurnV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *BurnV2:
		t.Proofs = proofs
		return t, nil
	case *ExchangeV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *ExchangeV2:
		t.Proofs = proofs
		return t, nil
	case *LeaseV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *LeaseV2:
		t.Proofs = proofs
		return t, nil
	case *LeaseCancelV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *LeaseCancelV2:
		t.Proofs = proofs
		return t, nil
	case *CreateAliasV1:
		t.Signature = c.ExtractFirstSignature(proofs)
		return t, c.err
	case *CreateAliasV2:
		t.Proofs = proofs
		return t, nil
	case *MassTransferV1:
		t.Proofs = proofs
		return t, nil
	case *DataV1:
		t.Proofs = proofs
		return t, nil
	case *SetScriptV1:
		t.Proofs = proofs
		return t, nil
	case *SponsorshipV1:
		t.Proofs = proofs
		return t, nil
	case *SetAssetScriptV1:
		t.Proofs = proofs
		return t, nil
	case *InvokeScriptV1:
		t.Proofs = proofs
		return t, nil
	default:
		panic("unsupported transaction")
	}
}

func (c *ProtobufConverter) BlockTransactions(block *g.BlockWithHeight) ([]Transaction, error) {
	if c.err != nil {
		return nil, c.err
	}
	txs := make([]Transaction, len(block.Block.Transactions))
	for i, stx := range block.Block.Transactions {
		tx, err := c.SignedTransaction(stx)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}

func (c *ProtobufConverter) Features(features []uint32) []int16 {
	r := make([]int16, len(features))
	for i, f := range features {
		r[i] = int16(f)
	}
	return r
}

func (c *ProtobufConverter) Consensus(header *g.Block_Header) NxtConsensus {
	if c.err != nil {
		return NxtConsensus{}
	}
	return NxtConsensus{
		GenSignature: c.Digest(header.GenerationSignature),
		BaseTarget:   c.Uint64(header.BaseTarget),
	}
}

func (c *ProtobufConverter) BlockHeader(block *g.BlockWithHeight) (BlockHeader, error) {
	if c.err != nil {
		return BlockHeader{}, c.err
	}
	features := c.Features(block.Block.Header.FeatureVotes)
	if c.err != nil {
		return BlockHeader{}, c.err
	}
	return BlockHeader{
		Version:          BlockVersion(c.Byte(block.Block.Header.Version)),
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
