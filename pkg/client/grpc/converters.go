package grpc

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SafeConverter struct {
	err error
}

func (c *SafeConverter) address(scheme byte, addr []byte) proto.Address {
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

func (c *SafeConverter) uint64(value int64) uint64 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("negative int64 value")
		return 0
	}
	return uint64(value)
}

func (c *SafeConverter) byte(value int32) byte {
	if c.err != nil {
		return 0
	}
	if value < 0 || value > 0xff {
		c.err = errors.New("invalid byte value")
	}
	return byte(value)
}

func (c *SafeConverter) digest(digest []byte) crypto.Digest {
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

func (c *SafeConverter) optionalAsset(asset *AssetId) proto.OptionalAsset {
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

func (c *SafeConverter) convertAmount(amount *Amount) (proto.OptionalAsset, uint64) {
	if c.err != nil {
		return proto.OptionalAsset{}, 0
	}
	return c.extractOptionalAsset(amount), c.amount(amount)
}

func (c *SafeConverter) convertAssetAmount(aa *AssetAmount) (crypto.Digest, uint64) {
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

func (c *SafeConverter) extractOptionalAsset(amount *Amount) proto.OptionalAsset {
	if c.err != nil {
		return proto.OptionalAsset{}
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return proto.OptionalAsset{}
	}
	return c.optionalAsset(amount.AssetId)
}

func (c *SafeConverter) amount(amount *Amount) uint64 {
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

func (c *SafeConverter) publicKey(pk []byte) crypto.PublicKey {
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

func (c *SafeConverter) string(bytes []byte) string {
	if c.err != nil {
		return ""
	}
	return string(bytes)
}

func (c *SafeConverter) script(script *Script) proto.Script {
	if c.err != nil {
		return nil
	}
	if script == nil {
		return nil
	}
	return proto.Script(script.Bytes)
}

func (c *SafeConverter) alias(scheme byte, alias string) proto.Alias {
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

func (c *SafeConverter) recipient(scheme byte, recipient *Recipient) proto.Recipient {
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

func (c *SafeConverter) assetPair(pair *ExchangeTransactionData_Order_AssetPair) proto.AssetPair {
	if c.err != nil {
		return proto.AssetPair{}
	}
	return proto.AssetPair{
		AmountAsset: c.optionalAsset(pair.AmountAssetId),
		PriceAsset:  c.optionalAsset(pair.PriceAssetId),
	}
}

func (c *SafeConverter) orderType(side ExchangeTransactionData_Order_Side) proto.OrderType {
	return proto.OrderType(c.byte(int32(side)))
}

func (c *SafeConverter) proofs(proofs [][]byte) *proto.ProofsV1 {
	if c.err != nil {
		return nil
	}
	r := proto.NewProofs()
	for _, proof := range proofs {
		r.Proofs = append(r.Proofs, proto.B58Bytes(proof))
	}
	return r
}

func (c *SafeConverter) proof(proofs [][]byte) *crypto.Signature {
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

func (c *SafeConverter) signature(data []byte) crypto.Signature {
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

func (c *SafeConverter) extractOrder(orders []*ExchangeTransactionData_Order, side ExchangeTransactionData_Order_Side) proto.Order {
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
				MatcherFee: c.amount(o.MatcherFee),
			}
			switch o.Version {
			case 3:
				order = &proto.OrderV3{
					Version:         c.byte(o.Version),
					Proofs:          c.proofs(o.Proofs),
					OrderBody:       body,
					MatcherFeeAsset: c.extractOptionalAsset(o.MatcherFee),
				}
			case 2:
				order = &proto.OrderV2{
					Version:   c.byte(o.Version),
					Proofs:    c.proofs(o.Proofs),
					OrderBody: body,
				}
			default:
				order = &proto.OrderV1{
					Signature: c.proof(o.Proofs),
					OrderBody: body,
				}
			}
			return order
		}
	}
	c.err = errors.Errorf("no order of side %s", side.String())
	return nil
}

func (c *SafeConverter) buyOrder(orders []*ExchangeTransactionData_Order) proto.Order {
	return c.extractOrder(orders, ExchangeTransactionData_Order_BUY)
}

func (c *SafeConverter) sellOrder(orders []*ExchangeTransactionData_Order) proto.Order {
	return c.extractOrder(orders, ExchangeTransactionData_Order_SELL)
}

func (c *SafeConverter) transfers(scheme byte, transfers []*MassTransferTransactionData_Transfer) []proto.MassTransferEntry {
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

func (c *SafeConverter) entries(entries []*DataTransactionData_DataEntry) proto.DataEntries {
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
			entry = &proto.IntegerDataEntry{Key: e.Key, Value: t.IntValue}
		case *DataTransactionData_DataEntry_BoolValue:
			entry = &proto.BooleanDataEntry{Key: e.Key, Value: t.BoolValue}
		case *DataTransactionData_DataEntry_BinaryValue:
			entry = &proto.BinaryDataEntry{Key: e.Key, Value: t.BinaryValue}
		case *DataTransactionData_DataEntry_StringValue:
			entry = &proto.StringDataEntry{Key: e.Key, Value: t.StringValue}
		}
		r[i] = entry
	}
	return r
}

func (c *SafeConverter) functionCall(data []byte) proto.FunctionCall {
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

func (c *SafeConverter) payments(payments []*Amount) proto.ScriptPayments {
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

func (c *SafeConverter) Reset() {
	c.err = nil
}

func (c *SafeConverter) Transaction(tx *Transaction) (proto.Transaction, error) {
	ts := c.uint64(tx.Timestamp)
	scheme := c.byte(tx.ChainId)
	v := c.byte(tx.Version)
	if c.err != nil {
		return nil, c.err
	}
	var rtx proto.Transaction
	switch d := tx.Data.(type) {
	case *Transaction_Genesis:
		rtx = &proto.Genesis{
			Type:      proto.GenesisTransaction,
			Version:   v,
			Timestamp: ts,
			Recipient: c.address(scheme, d.Genesis.RecipientAddress),
			Amount:    uint64(d.Genesis.Amount),
		}

	case *Transaction_Payment:
		rtx = &proto.Payment{
			Type:      proto.PaymentTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Recipient: c.address(scheme, d.Payment.RecipientAddress),
			Amount:    c.uint64(d.Payment.Amount),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

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
			rtx = &proto.IssueV2{
				Type:    proto.IssueTransaction,
				Version: v,
				ChainID: scheme,
				Script:  c.script(d.Issue.Script),
				Issue:   pi,
			}
		default:
			rtx = &proto.IssueV1{
				Type:    proto.IssueTransaction,
				Version: v,
				Issue:   pi,
			}
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

	case *Transaction_Exchange:
		fee := c.amount(tx.Fee)
		bo := c.buyOrder(d.Exchange.Orders)
		so := c.sellOrder(d.Exchange.Orders)
		switch tx.Version {
		case 2:
			rtx = &proto.ExchangeV2{
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
				SenderPK:       c.publicKey(tx.SenderPublicKey),
				BuyOrder:       bo1,
				SellOrder:      so1,
				Price:          c.uint64(d.Exchange.Price),
				Amount:         c.uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}
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

	case *Transaction_LeaseCancel:
		plc := proto.LeaseCancel{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			LeaseID:   c.digest(d.LeaseCancel.LeaseId),
			Fee:       c.amount(tx.Fee),
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

	case *Transaction_CreateAlias:
		pca := proto.CreateAlias{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Alias:     c.alias(scheme, d.CreateAlias.Alias),
			Fee:       c.amount(tx.Fee),
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

	case *Transaction_MassTransfer:
		rtx = &proto.MassTransferV1{
			Type:       proto.MassTransferTransaction,
			Version:    v,
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			Asset:      c.optionalAsset(d.MassTransfer.AssetId),
			Transfers:  c.transfers(scheme, d.MassTransfer.Transfers),
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
			Attachment: proto.Attachment(c.string(d.MassTransfer.Attachment)),
		}

	case *Transaction_DataTransaction:
		rtx = &proto.DataV1{
			Type:      proto.DataTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Entries:   c.entries(d.DataTransaction.Data),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *Transaction_SetScript:
		rtx = &proto.SetScriptV1{
			Type:      proto.SetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Script:    c.script(d.SetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *Transaction_SponsorFee:
		asset, amount := c.convertAssetAmount(d.SponsorFee.MinFee)
		rtx = &proto.SponsorshipV1{
			Type:        proto.SponsorshipTransaction,
			Version:     v,
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AssetID:     asset,
			MinAssetFee: amount,
			Fee:         c.amount(tx.Fee),
			Timestamp:   ts,
		}

	case *Transaction_SetAssetScript:
		rtx = &proto.SetAssetScriptV1{
			Type:      proto.SetAssetScriptTransaction,
			Version:   v,
			ChainID:   scheme,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			AssetID:   c.digest(d.SetAssetScript.AssetId),
			Script:    c.script(d.SetAssetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *Transaction_InvokeScript:
		feeAsset, feeAmount := c.convertAmount(tx.Fee)
		rtx = &proto.InvokeScriptV1{
			Type:            proto.InvokeScriptTransaction,
			Version:         v,
			ChainID:         scheme,
			SenderPK:        c.publicKey(tx.SenderPublicKey),
			ScriptRecipient: c.recipient(scheme, d.InvokeScript.DApp),
			FunctionCall:    c.functionCall(d.InvokeScript.FunctionCall),
			Payments:        c.payments(d.InvokeScript.Payments),
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

func (c *SafeConverter) extractFirstSignature(proofs *proto.ProofsV1) *crypto.Signature {
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

func (c *SafeConverter) SignedTransaction(stx *SignedTransaction) (proto.Transaction, error) {
	tx, err := c.Transaction(stx.Transaction)
	if err != nil {
		return nil, err
	}
	proofs := c.proofs(stx.Proofs)
	if c.err != nil {
		return nil, c.err
	}
	switch t := tx.(type) {
	case *proto.Genesis:
		sig := c.extractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *proto.Payment:
		sig := c.extractFirstSignature(proofs)
		t.Signature = sig
		t.ID = sig
		return t, c.err
	case *proto.IssueV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.IssueV2:
		t.Proofs = proofs
		return t, nil
	case *proto.TransferV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.TransferV2:
		t.Proofs = proofs
		return t, nil
	case *proto.ReissueV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.ReissueV2:
		t.Proofs = proofs
		return t, nil
	case *proto.BurnV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.BurnV2:
		t.Proofs = proofs
		return t, nil
	case *proto.ExchangeV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.ExchangeV2:
		t.Proofs = proofs
		return t, nil
	case *proto.LeaseV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.LeaseV2:
		t.Proofs = proofs
		return t, nil
	case *proto.LeaseCancelV1:
		t.Signature = c.extractFirstSignature(proofs)
		return t, c.err
	case *proto.LeaseCancelV2:
		t.Proofs = proofs
		return t, nil
	case *proto.CreateAliasV1:
		t.Signature = c.extractFirstSignature(proofs)
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

func (c *SafeConverter) BlockTransactions(block *BlockWithHeight) ([]proto.Transaction, error) {
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

func (c *SafeConverter) features(features []uint32) []int16 {
	r := make([]int16, len(features))
	for i, f := range features {
		r[i] = int16(f)
	}
	return r
}

func (c *SafeConverter) consensus(header *Block_Header) proto.NxtConsensus {
	if c.err != nil {
		return proto.NxtConsensus{}
	}
	return proto.NxtConsensus{
		GenSignature: c.digest(header.GenerationSignature),
		BaseTarget:   c.uint64(header.BaseTarget),
	}
}

func (c *SafeConverter) BlockHeader(block *BlockWithHeight) (proto.BlockHeader, error) {
	if c.err != nil {
		return proto.BlockHeader{}, c.err
	}
	features := c.features(block.Block.Header.FeatureVotes)
	if c.err != nil {
		return proto.BlockHeader{}, c.err
	}
	return proto.BlockHeader{
		Version:          proto.BlockVersion(c.byte(block.Block.Header.Version)),
		Timestamp:        c.uint64(block.Block.Header.Timestamp),
		Parent:           c.signature(block.Block.Header.Reference),
		FeaturesCount:    len(features),
		Features:         features,
		NxtConsensus:     c.consensus(block.Block.Header),
		TransactionCount: len(block.Block.Transactions),
		GenPublicKey:     c.publicKey(block.Block.Header.Generator),
		BlockSignature:   c.signature(block.Block.Signature),
	}, nil
}
