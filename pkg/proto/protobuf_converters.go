package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	protobuf "google.golang.org/protobuf/proto"
)

func MarshalToProtobufDeterministic(pb protobuf.Message) ([]byte, error) {
	b, err := protobuf.MarshalOptions{Deterministic: true}.Marshal(pb)
	if err != nil {
		return nil, err
	}
	if len(b) > 10*1024*1024 {
		return nil, errors.New("failed to marshal protobuf message, marshaled value exceeds 10MB limit")
	}
	return b, nil
}

func MarshalTxDeterministic(tx Transaction, scheme Scheme) ([]byte, error) {
	pbTx, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	return pbTx.MarshalVTStrict()
}

func MarshalSignedTxDeterministic(tx Transaction, scheme Scheme) ([]byte, error) {
	pbTx, err := tx.ToProtobufSigned(scheme)
	if err != nil {
		return nil, err
	}
	return pbTx.MarshalVTStrict()
}

func TxFromProtobuf(data []byte) (Transaction, error) {
	var pbTx g.Transaction
	if err := protobuf.Unmarshal(data, &pbTx); err != nil {
		return nil, err
	}
	var c ProtobufConverter
	res, err := c.Transaction(&pbTx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func SignedTxFromProtobuf(data []byte) (Transaction, error) {
	var pbTx g.SignedTransaction
	if err := protobuf.Unmarshal(data, &pbTx); err != nil {
		return nil, err
	}
	var c ProtobufConverter
	res, err := c.SignedTransaction(&pbTx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type ProtobufConverter struct {
	FallbackChainID byte
	err             error
}

func (c *ProtobufConverter) Address(scheme byte, addr []byte) (WavesAddress, error) {
	a, err := RebuildAddress(scheme, addr)
	if err != nil {
		return WavesAddress{}, err
	}
	return a, nil
}

func (c *ProtobufConverter) uint64(value int64) uint64 {
	if c.err != nil {
		return 0
	}
	if value < 0 {
		c.err = errors.New("non-positive amount")
		return 0
	}
	return uint64(value)
}

func (c *ProtobufConverter) uint64v(value int64, validators ...func(int64) error) uint64 {
	if c.err != nil {
		return 0
	}
	for _, v := range validators {
		err := v(value)
		if err != nil {
			c.err = err
			return 0
		}
	}
	for _, v := range validators {
		err := v(value)
		if err != nil {
			c.err = err
			return 0
		}
	}
	return uint64(value)
}

func (c *ProtobufConverter) byte(value int32, context ...string) byte {
	if c.err != nil {
		return 0
	}
	if value < 0 || value > 0xff {
		if len(context) > 0 {
			c.err = errors.Errorf("invalid byte value, context: %s", context[0])
		} else {
			c.err = errors.New("invalid byte value")
		}
	}
	return byte(value)
}

func (c *ProtobufConverter) byteValidated(value int32, validators ...func(int32) error) byte {
	if c.err != nil {
		return 0
	}
	for _, v := range validators {
		err := v(value)
		if err != nil {
			c.err = err
			return 0
		}
	}
	return byte(value)
}

func (c *ProtobufConverter) digest(digest []byte) crypto.Digest {
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

func (c *ProtobufConverter) optionalAsset(asset []byte) OptionalAsset {
	if c.err != nil {
		return OptionalAsset{}
	}
	if len(asset) == 0 {
		return NewOptionalAssetWaves()
	}
	return *NewOptionalAssetFromDigest(c.digest(asset))
}

func (c *ProtobufConverter) convertAmount(amount *g.Amount) (OptionalAsset, uint64) {
	if c.err != nil {
		return OptionalAsset{}, 0
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return OptionalAsset{}, 0
	}
	return c.extractOptionalAsset(amount), c.amount(amount)
}

func (c *ProtobufConverter) convertAssetAmount(aa *g.Amount) (crypto.Digest, uint64) {
	if c.err != nil {
		return crypto.Digest{}, 0
	}
	if aa == nil {
		c.err = errors.New("empty asset amount")
		return crypto.Digest{}, 0
	}
	id, err := crypto.NewDigestFromBytes(aa.AssetId)
	if err != nil {
		c.err = err
		return crypto.Digest{}, 0
	}
	return id, c.uint64(aa.Amount)
}

func (c *ProtobufConverter) extractOptionalAsset(amount *g.Amount) OptionalAsset {
	if c.err != nil {
		return OptionalAsset{}
	}
	if amount == nil {
		c.err = errors.New("empty asset amount")
		return OptionalAsset{}
	}
	return c.optionalAsset(amount.AssetId)
}

func (c *ProtobufConverter) amount(amount *g.Amount) uint64 {
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

func (c *ProtobufConverter) publicKey(pk []byte) crypto.PublicKey {
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

func (c *ProtobufConverter) alias(scheme byte, alias string) Alias {
	if c.err != nil {
		return Alias{}
	}
	a := NewAlias(scheme, alias)
	_, err := a.Valid(scheme)
	if err != nil {
		c.err = err
		return Alias{}
	}
	return *a
}

func (c *ProtobufConverter) Recipient(scheme byte, recipient *g.Recipient) (Recipient, error) {
	if recipient == nil {
		return Recipient{}, errors.New("empty recipient")
	}
	switch r := recipient.Recipient.(type) {
	case *g.Recipient_PublicKeyHash:
		addr, err := c.Address(scheme, r.PublicKeyHash)
		if err != nil {
			return Recipient{}, err
		}
		return NewRecipientFromAddress(addr), nil
	case *g.Recipient_Alias:
		return NewRecipientFromAlias(c.alias(scheme, r.Alias)), nil
	default:
		return Recipient{}, errors.New("invalid recipient")
	}
}

func (c *ProtobufConverter) assetPair(pair *g.AssetPair) AssetPair {
	if c.err != nil {
		return AssetPair{}
	}
	if pair == nil {
		c.err = errors.New("empty asset pair")
		return AssetPair{}
	}
	return AssetPair{
		AmountAsset: c.optionalAsset(pair.AmountAssetId),
		PriceAsset:  c.optionalAsset(pair.PriceAssetId),
	}
}

func (c *ProtobufConverter) orderType(side g.Order_Side) OrderType {
	return OrderType(c.byte(int32(side)))
}

func (c *ProtobufConverter) orderPriceMode(gm g.Order_PriceMode) (OrderPriceMode, error) {
	if c.err != nil {
		return 0, c.err
	}
	var m OrderPriceMode
	if err := m.FromProtobuf(gm); err != nil {
		return 0, err
	}
	return m, nil
}

func (c *ProtobufConverter) proofs(proofs [][]byte) *ProofsV1 {
	if c.err != nil {
		return nil
	}
	r := NewProofs()
	r.Proofs = make([]B58Bytes, 0, len(proofs))
	for _, proof := range proofs {
		r.Proofs = append(r.Proofs, proof)
	}
	if err := r.Valid(); err != nil {
		c.err = err
		return nil
	}
	return r
}

func (c *ProtobufConverter) proof(proofs [][]byte) *crypto.Signature {
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

func (c *ProtobufConverter) blockID(data []byte) BlockID {
	if c.err != nil {
		return BlockID{}
	}
	id, err := NewBlockIDFromBytes(data)
	if err != nil {
		c.err = err
		return BlockID{}
	}
	return id
}

func (c *ProtobufConverter) signature(data []byte) crypto.Signature {
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

func (c *ProtobufConverter) ethSignature(data []byte) EthereumSignature {
	if c.err != nil {
		return EthereumSignature{}
	}
	sig, err := NewEthereumSignatureFromBytes(data)
	if err != nil {
		c.err = err
		return EthereumSignature{}
	}
	return sig
}

func (c *ProtobufConverter) extractOrder(o *g.Order) Order {
	if c.err != nil {
		return nil
	}
	if o == nil {
		c.err = errors.New("empty order")
		return nil
	}
	orderVersion := c.byte(o.Version)
	priceMode, err := c.orderPriceMode(o.PriceMode)
	if err != nil {
		c.err = err
		return nil
	}
	if _, err := priceMode.Valid(orderVersion); err != nil {
		c.err = err
		return nil
	}
	var order Order
	body := OrderBody{
		MatcherPK:  c.publicKey(o.MatcherPublicKey),
		AssetPair:  c.assetPair(o.AssetPair),
		OrderType:  c.orderType(o.OrderSide),
		Price:      c.uint64(o.Price),
		Amount:     c.uint64(o.Amount),
		Timestamp:  c.uint64(o.Timestamp),
		Expiration: c.uint64(o.Expiration),
		MatcherFee: c.amount(o.MatcherFee),
	}
	switch sender := o.Sender.(type) {
	case *g.Order_SenderPublicKey:
		body.SenderPK = c.publicKey(sender.SenderPublicKey)
	case *g.Order_Eip712Signature:
		if o.Version < 4 {
			// nickeskov: see isValid method in com/wavesplatform/transaction/assets/exchange/Order.scala
			c.err = errors.New("eip712Signature available only in OrderV4")
			return nil
		}
	default:
		c.err = errors.Errorf("unknown order.Sender field=%v", sender)
		return nil
	}
	scheme := c.byte(o.ChainId)
	if scheme == 0 {
		scheme = c.FallbackChainID
	}
	switch uncheckedOrderVersion := o.Version; uncheckedOrderVersion {
	case 1:
		order = &OrderV1{
			Signature: c.proof(o.Proofs),
			OrderBody: body,
		}
	case 2:
		order = &OrderV2{
			Version:   orderVersion,
			Proofs:    c.proofs(o.Proofs),
			OrderBody: body,
		}
	case 3:
		order = &OrderV3{
			Version:         orderVersion,
			Proofs:          c.proofs(o.Proofs),
			OrderBody:       body,
			MatcherFeeAsset: c.extractOptionalAsset(o.MatcherFee),
		}
	case 4:
		orderV4 := OrderV4{
			Version:         orderVersion,
			Proofs:          c.proofs(o.Proofs),
			OrderBody:       body,
			MatcherFeeAsset: c.extractOptionalAsset(o.MatcherFee),
			PriceMode:       priceMode,
		}
		if sig, ok := o.Sender.(*g.Order_Eip712Signature); ok {
			ethOrder := EthereumOrderV4{
				Eip712Signature: c.ethSignature(sig.Eip712Signature),
				OrderV4:         orderV4,
			}
			if err := ethOrder.GenerateSenderPK(scheme); err != nil {
				c.err = err
				return nil
			}
			order = &ethOrder
		} else {
			order = &orderV4
		}
	default:
		c.err = errors.Errorf("invalid order version %d", uncheckedOrderVersion)
		return nil
	}
	if err := order.GenerateID(scheme); err != nil {
		c.err = err
		return nil
	}
	return order
}

func (c *ProtobufConverter) transfers(scheme byte, transfers []*g.MassTransferTransactionData_Transfer) []MassTransferEntry {
	if c.err != nil {
		return nil
	}
	r := make([]MassTransferEntry, len(transfers))
	for i, tr := range transfers {
		if tr == nil {
			c.err = errors.New("empty transfer")
			return nil
		}
		rcp, err := c.Recipient(scheme, tr.Recipient)
		if err != nil {
			c.err = err
			return nil
		}
		e := MassTransferEntry{
			Recipient: rcp,
			Amount: c.uint64v(tr.Amount, func(i int64) error {
				return errors.WithStack(ValidateNonNegativeAmount(i, "One of the transfers has negative amount"))
			}),
		}
		if c.err != nil {
			return nil
		}
		r[i] = e
	}
	return r
}

func (c *ProtobufConverter) attachment(att []byte) Attachment {
	// this cast is required, tests fill fall if removed!
	if len(att) == 0 {
		return Attachment{}
	}
	return att
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
	default: // No value means DeleteDataEntry
		e = &DeleteDataEntry{Key: entry.Key}
	}
	return e
}

func (c *ProtobufConverter) Entry(entry *g.DataTransactionData_DataEntry) (DataEntry, error) {
	e := c.entry(entry)
	if c.err != nil {
		err := c.err
		c.reset()
		return nil, err
	}
	return e, nil
}

func (c *ProtobufConverter) script(script []byte) Script {
	if c.err != nil {
		return Script{}
	}
	res := Script{}
	if len(script) != 0 {
		res = script
	}
	return res
}

func (c *ProtobufConverter) entries(entries []*g.DataTransactionData_DataEntry) DataEntries {
	if c.err != nil {
		return nil
	}
	r := make([]DataEntry, len(entries))
	for i, e := range entries {
		r[i] = c.entry(e)
	}
	return r
}

func (c *ProtobufConverter) functionCall(data []byte) FunctionCall {
	if c.err != nil {
		return FunctionCall{}
	}
	// FIXME: The following block fixes the bug introduced in Scala implementation of gRPC
	// It should be removed after the release of fix.
	// TODO: consider removing the block below
	var d []byte
	if len(data) > 3 && data[0] == 1 && data[3] == 9 {
		d = make([]byte, len(data)-2)
		d[0] = data[0]
		copy(d[1:], data[3:])
	} else {
		d = data
	}
	// FIXME: remove the block above after updating to fixed version.
	fc := FunctionCall{}
	err := fc.UnmarshalBinary(d)
	if err != nil {
		c.err = err
		return FunctionCall{}
	}
	return fc
}

func (c *ProtobufConverter) payments(payments []*g.Amount) ScriptPayments {
	if payments == nil {
		return ScriptPayments(nil)
	}
	result := make([]ScriptPayment, len(payments))
	for i, p := range payments {
		asset, amount := c.convertAmount(p)
		result[i] = ScriptPayment{Asset: asset, Amount: amount}
	}
	return result
}

func (c *ProtobufConverter) TransferScriptActions(scheme byte, payments []*g.InvokeScriptResult_Payment) ([]*TransferScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*TransferScriptAction, len(payments))
	for i, p := range payments {
		asset, amount := c.convertAmount(p.Amount)
		if c.err != nil {
			return nil, c.err
		}
		addr, err := c.Address(scheme, p.Address)
		if err != nil {
			return nil, err
		}
		res[i] = &TransferScriptAction{
			Recipient: NewRecipientFromAddress(addr),
			Amount:    int64(amount),
			Asset:     asset,
		}
	}
	return res, nil
}

func (c *ProtobufConverter) IssueScriptActions(issues []*g.InvokeScriptResult_Issue) ([]*IssueScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*IssueScriptAction, len(issues))
	for i, x := range issues {
		res[i] = &IssueScriptAction{
			ID:          c.digest(x.AssetId),
			Name:        x.Name,
			Description: x.Description,
			Quantity:    x.Amount,
			Decimals:    x.Decimals,
			Reissuable:  x.Reissuable,
			Script:      c.script(x.Script),
			Nonce:       x.Nonce,
		}
		if c.err != nil {
			return nil, c.err
		}
	}
	return res, nil
}

func (c *ProtobufConverter) ReissueScriptActions(reissues []*g.InvokeScriptResult_Reissue) ([]*ReissueScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*ReissueScriptAction, len(reissues))
	for i, x := range reissues {
		res[i] = &ReissueScriptAction{
			AssetID:    c.digest(x.AssetId),
			Quantity:   x.Amount,
			Reissuable: x.IsReissuable,
		}
		if c.err != nil {
			return nil, c.err
		}
	}
	return res, nil
}

func (c *ProtobufConverter) BurnScriptActions(burns []*g.InvokeScriptResult_Burn) ([]*BurnScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*BurnScriptAction, len(burns))
	for i, x := range burns {
		res[i] = &BurnScriptAction{
			AssetID:  c.digest(x.AssetId),
			Quantity: x.Amount,
		}
		if c.err != nil {
			return nil, c.err
		}
	}
	return res, nil
}

func (c *ProtobufConverter) SponsorshipScriptActions(sponsorships []*g.InvokeScriptResult_SponsorFee) ([]*SponsorshipScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*SponsorshipScriptAction, len(sponsorships))
	for i, x := range sponsorships {
		assetID, amount := c.convertAssetAmount(x.MinFee)
		if c.err != nil {
			return nil, c.err
		}
		res[i] = &SponsorshipScriptAction{
			AssetID: assetID,
			MinFee:  int64(amount),
		}
	}
	return res, nil
}

func (c *ProtobufConverter) LeaseScriptActions(scheme byte, leases []*g.InvokeScriptResult_Lease) ([]*LeaseScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*LeaseScriptAction, len(leases))
	for i, x := range leases {
		rcp, err := c.Recipient(scheme, x.Recipient)
		if err != nil {
			c.err = err
			return nil, err
		}
		res[i] = &LeaseScriptAction{
			ID:        c.digest(x.LeaseId),
			Recipient: rcp,
			Amount:    x.Amount,
			Nonce:     x.Nonce,
		}
		if c.err != nil {
			return nil, c.err
		}
	}
	return res, nil
}

func (c *ProtobufConverter) LeaseCancelScriptActions(cancels []*g.InvokeScriptResult_LeaseCancel) ([]*LeaseCancelScriptAction, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make([]*LeaseCancelScriptAction, len(cancels))
	for i, x := range cancels {
		res[i] = &LeaseCancelScriptAction{
			LeaseID: c.digest(x.LeaseId),
		}
		if c.err != nil {
			return nil, c.err
		}
	}
	return res, nil
}

func (c *ProtobufConverter) ErrorMessage(msg *g.InvokeScriptResult_ErrorMessage) (ScriptErrorMessage, error) {
	if c.err != nil {
		return ScriptErrorMessage{}, c.err
	}
	if msg == nil {
		return ScriptErrorMessage{}, errors.New("empty invoke script result error message")
	}
	return ScriptErrorMessage{
		Code: TxFailureReason(msg.Code),
		Text: msg.Text,
	}, nil
}

func (c *ProtobufConverter) reset() {
	c.err = nil
}

func (c *ProtobufConverter) Transaction(tx *g.Transaction) (Transaction, error) {
	if tx == nil {
		return nil, errors.New("emtpy transaction")
	}
	ts := c.uint64(tx.Timestamp)
	scheme := c.byte(tx.ChainId)
	if scheme == 0 {
		scheme = c.FallbackChainID
	}
	v := c.byte(tx.Version)
	var rtx Transaction
	switch d := tx.Data.(type) {
	case *g.Transaction_Genesis:
		rcpAddr, err := c.Address(scheme, d.Genesis.RecipientAddress)
		if err != nil {
			c.reset()
			return nil, err
		}
		rtx = &Genesis{
			Type:      GenesisTransaction,
			Version:   v,
			Timestamp: ts,
			Recipient: rcpAddr,
			Amount:    uint64(d.Genesis.Amount),
		}

	case *g.Transaction_Payment:
		rcpAddr, err := c.Address(scheme, d.Payment.RecipientAddress)
		if err != nil {
			c.reset()
			return nil, err
		}
		rtx = &Payment{
			Type:      PaymentTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Recipient: rcpAddr,
			Amount:    c.uint64(d.Payment.Amount),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_Issue:
		pi := Issue{
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			Name:        d.Issue.Name,
			Description: d.Issue.Description,
			Quantity: c.uint64v(d.Issue.Amount, func(v int64) error {
				return ValidatePositiveAmount(v, "quantity")
			}),
			Decimals: c.byteValidated(d.Issue.Decimals, func(value int32) error {
				return nil
			}),
			Reissuable: d.Issue.Reissuable,
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
		}
		if tx.Version >= 2 {
			rtx = &IssueWithProofs{
				Type:    IssueTransaction,
				Version: v,
				Script:  c.script(d.Issue.Script),
				Issue:   pi,
			}
		} else {
			rtx = &IssueWithSig{
				Type:    IssueTransaction,
				Version: v,
				Issue:   pi,
			}
		}

	case *g.Transaction_Transfer:
		aa, amount := c.convertAmount(d.Transfer.Amount)
		fa, fee := c.convertAmount(tx.Fee)
		rcp, err := c.Recipient(scheme, d.Transfer.Recipient)
		if err != nil {
			c.reset()
			return nil, err
		}
		pt := Transfer{
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AmountAsset: aa,
			FeeAsset:    fa,
			Timestamp:   ts,
			Amount:      amount,
			Fee:         fee,
			Recipient:   rcp,
			Attachment:  c.attachment(d.Transfer.Attachment),
		}
		if tx.Version >= 2 {
			rtx = &TransferWithProofs{
				Type:     TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		} else {
			rtx = &TransferWithSig{
				Type:     TransferTransaction,
				Version:  v,
				Transfer: pt,
			}
		}

	case *g.Transaction_Reissue:
		id, quantity := c.convertAssetAmount(d.Reissue.AssetAmount)
		pr := Reissue{
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			AssetID:    id,
			Quantity:   quantity,
			Reissuable: d.Reissue.Reissuable,
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
		}
		if tx.Version >= 2 {
			rtx = &ReissueWithProofs{
				Type:    ReissueTransaction,
				Version: v,
				Reissue: pr,
			}
		} else {
			rtx = &ReissueWithSig{
				Type:    ReissueTransaction,
				Version: v,
				Reissue: pr,
			}
		}

	case *g.Transaction_Burn:
		id, amount := c.convertAssetAmount(d.Burn.AssetAmount)
		pb := Burn{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			AssetID:   id,
			Amount:    amount,
			Timestamp: ts,
			Fee:       c.amount(tx.Fee),
		}
		if tx.Version >= 2 {
			rtx = &BurnWithProofs{
				Type:    BurnTransaction,
				Version: v,
				Burn:    pb,
			}
		} else {
			rtx = &BurnWithSig{
				Type:    BurnTransaction,
				Version: v,
				Burn:    pb,
			}
		}

	case *g.Transaction_Exchange:
		fee := c.amount(tx.Fee)
		if n := len(d.Exchange.Orders); n != 2 {
			c.reset()
			return nil, errors.Errorf("invalid number (%d) of orders in exchange transaction", n)
		}
		o1 := c.extractOrder(d.Exchange.Orders[0])
		o2 := c.extractOrder(d.Exchange.Orders[1])
		if tx.Version >= 2 {
			rtx = &ExchangeWithProofs{
				Type:           ExchangeTransaction,
				Version:        v,
				SenderPK:       c.publicKey(tx.SenderPublicKey),
				Order1:         o1,
				Order2:         o2,
				Price:          c.uint64(d.Exchange.Price),
				Amount:         c.uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}
		} else {
			if o1 != nil && o2 != nil && (o1.GetVersion() != 1 || o2.GetVersion() != 1) {
				c.reset()
				return nil, errors.New("unsupported order version")
			}
			o1v1, ok := o1.(*OrderV1)
			if !ok {
				c.reset()
				return nil, errors.New("invalid pointer to OrderV1")
			}
			o2v1, ok := o2.(*OrderV1)
			if !ok {
				c.reset()
				return nil, errors.New("invalid pointer to OrderV1")
			}
			rtx = &ExchangeWithSig{
				Type:           ExchangeTransaction,
				Version:        v,
				SenderPK:       c.publicKey(tx.SenderPublicKey),
				Order1:         o1v1,
				Order2:         o2v1,
				Price:          c.uint64(d.Exchange.Price),
				Amount:         c.uint64(d.Exchange.Amount),
				BuyMatcherFee:  c.uint64(d.Exchange.BuyMatcherFee),
				SellMatcherFee: c.uint64(d.Exchange.SellMatcherFee),
				Fee:            fee,
				Timestamp:      ts,
			}
		}

	case *g.Transaction_Lease:
		rcp, err := c.Recipient(scheme, d.Lease.Recipient)
		if err != nil {
			c.reset()
			return nil, err
		}
		pl := Lease{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Recipient: rcp,
			Amount:    c.uint64(d.Lease.Amount),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		if tx.Version >= 2 {
			rtx = &LeaseWithProofs{
				Type:    LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		} else {
			rtx = &LeaseWithSig{
				Type:    LeaseTransaction,
				Version: v,
				Lease:   pl,
			}
		}

	case *g.Transaction_LeaseCancel:
		plc := LeaseCancel{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			LeaseID:   c.digest(d.LeaseCancel.LeaseId),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		if tx.Version >= 2 {
			rtx = &LeaseCancelWithProofs{
				Type:        LeaseCancelTransaction,
				Version:     v,
				LeaseCancel: plc,
			}
		} else {
			rtx = &LeaseCancelWithSig{
				Type:        LeaseCancelTransaction,
				Version:     v,
				LeaseCancel: plc,
			}
		}

	case *g.Transaction_CreateAlias:
		pca := CreateAlias{
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Alias:     c.alias(scheme, d.CreateAlias.Alias),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
		if tx.Version >= 2 {
			rtx = &CreateAliasWithProofs{
				Type:        CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		} else {
			rtx = &CreateAliasWithSig{
				Type:        CreateAliasTransaction,
				Version:     v,
				CreateAlias: pca,
			}
		}

	case *g.Transaction_MassTransfer:
		rtx = &MassTransferWithProofs{
			Type:       MassTransferTransaction,
			Version:    v,
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			Asset:      c.optionalAsset(d.MassTransfer.AssetId),
			Transfers:  c.transfers(scheme, d.MassTransfer.Transfers),
			Timestamp:  ts,
			Fee:        c.amount(tx.Fee),
			Attachment: c.attachment(d.MassTransfer.Attachment),
		}

	case *g.Transaction_DataTransaction:
		rtx = &DataWithProofs{
			Type:      DataTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Entries:   c.entries(d.DataTransaction.Data),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SetScript:
		rtx = &SetScriptWithProofs{
			Type:      SetScriptTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			Script:    c.script(d.SetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}

	case *g.Transaction_SponsorFee:
		asset, amount := c.convertAssetAmount(d.SponsorFee.MinFee)
		rtx = &SponsorshipWithProofs{
			Type:        SponsorshipTransaction,
			Version:     v,
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AssetID:     asset,
			MinAssetFee: amount,
			Fee:         c.amount(tx.Fee),
			Timestamp:   ts,
		}

	case *g.Transaction_SetAssetScript:
		rtx = &SetAssetScriptWithProofs{
			Type:      SetAssetScriptTransaction,
			Version:   v,
			SenderPK:  c.publicKey(tx.SenderPublicKey),
			AssetID:   c.digest(d.SetAssetScript.AssetId),
			Script:    c.script(d.SetAssetScript.Script),
			Fee:       c.amount(tx.Fee),
			Timestamp: ts,
		}
	case *g.Transaction_InvokeExpression:
		feeAsset, feeAmount := c.convertAmount(tx.Fee)
		rtx = &InvokeExpressionTransactionWithProofs{
			Type:       InvokeExpressionTransaction,
			Version:    v,
			SenderPK:   c.publicKey(tx.SenderPublicKey),
			FeeAsset:   feeAsset,
			Fee:        feeAmount,
			Timestamp:  ts,
			Expression: d.InvokeExpression.Expression,
		}

	case *g.Transaction_InvokeScript:
		rcp, err := c.Recipient(scheme, d.InvokeScript.DApp)
		if err != nil {
			c.reset()
			return nil, err
		}
		feeAsset, feeAmount := c.convertAmount(tx.Fee)
		rtx = &InvokeScriptWithProofs{
			Type:            InvokeScriptTransaction,
			Version:         v,
			SenderPK:        c.publicKey(tx.SenderPublicKey),
			ScriptRecipient: rcp,
			FunctionCall:    c.functionCall(d.InvokeScript.FunctionCall),
			Payments:        c.payments(d.InvokeScript.Payments),
			FeeAsset:        feeAsset,
			Fee:             feeAmount,
			Timestamp:       ts,
		}
	case *g.Transaction_UpdateAssetInfo:
		feeAsset, feeAmount := c.convertAmount(tx.Fee)
		rtx = &UpdateAssetInfoWithProofs{
			Type:        UpdateAssetInfoTransaction,
			Version:     v,
			SenderPK:    c.publicKey(tx.SenderPublicKey),
			AssetID:     c.digest(d.UpdateAssetInfo.AssetId),
			Name:        d.UpdateAssetInfo.Name,
			Description: d.UpdateAssetInfo.Description,
			FeeAsset:    feeAsset,
			Fee:         feeAmount,
			Timestamp:   ts,
		}
	default:
		c.reset()
		return nil, errors.New("unsupported transaction")
	}
	if c.err != nil {
		err := c.err
		c.reset()
		return nil, err
	}
	if err := rtx.GenerateID(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to generate ID")
	}
	return rtx, nil
}

func (c *ProtobufConverter) extractFirstSignature(proofs *ProofsV1) *crypto.Signature {
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
	return c.signedTransaction(stx)
}

func (c *ProtobufConverter) signedTransaction(stx *g.SignedTransaction) (Transaction, error) {
	if stx == nil {
		return nil, errors.New("empty signed transaction")
	}
	switch wrappedTx := stx.Transaction.(type) {
	case *g.SignedTransaction_WavesTransaction:
		tx, err := c.Transaction(wrappedTx.WavesTransaction)
		if err != nil {
			return nil, err
		}
		proofs := c.proofs(stx.Proofs)
		if c.err != nil {
			err := c.err
			c.reset()
			return nil, err
		}
		switch t := tx.(type) {
		case *Genesis:
			sig := c.extractFirstSignature(proofs)
			t.Signature = sig
			t.ID = sig
			err := c.err
			c.reset()
			return t, err
		case *Payment:
			sig := c.extractFirstSignature(proofs)
			t.Signature = sig
			t.ID = sig
			err := c.err
			c.reset()
			return t, err
		case *IssueWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *IssueWithProofs:
			t.Proofs = proofs
			return t, nil
		case *TransferWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *TransferWithProofs:
			t.Proofs = proofs
			return t, nil
		case *ReissueWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *ReissueWithProofs:
			t.Proofs = proofs
			return t, nil
		case *BurnWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *BurnWithProofs:
			t.Proofs = proofs
			return t, nil
		case *ExchangeWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *ExchangeWithProofs:
			t.Proofs = proofs
			return t, nil
		case *LeaseWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *LeaseWithProofs:
			t.Proofs = proofs
			return t, nil
		case *LeaseCancelWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *LeaseCancelWithProofs:
			t.Proofs = proofs
			return t, nil
		case *CreateAliasWithSig:
			t.Signature = c.extractFirstSignature(proofs)
			err := c.err
			c.reset()
			return t, err
		case *CreateAliasWithProofs:
			t.Proofs = proofs
			return t, nil
		case *MassTransferWithProofs:
			t.Proofs = proofs
			return t, nil
		case *DataWithProofs:
			t.Proofs = proofs
			return t, nil
		case *SetScriptWithProofs:
			t.Proofs = proofs
			return t, nil
		case *SponsorshipWithProofs:
			t.Proofs = proofs
			return t, nil
		case *SetAssetScriptWithProofs:
			t.Proofs = proofs
			return t, nil
		case *InvokeScriptWithProofs:
			t.Proofs = proofs
			return t, nil
		case *InvokeExpressionTransactionWithProofs:
			t.Proofs = proofs
			return t, nil
		case *UpdateAssetInfoWithProofs:
			t.Proofs = proofs
			return t, nil
		default:
			panic("unsupported transaction")
		}
	case *g.SignedTransaction_EthereumTransaction:
		tx, err := c.ethereumTransaction(wrappedTx.EthereumTransaction)
		if err != nil {
			return nil, err
		}
		return tx, nil
	default:
		panic(errors.Errorf(
			"BUG, CREATE REPORT: unsupported protobuf signed transaction variant type %T.",
			stx.Transaction,
		))
	}
}

func (c *ProtobufConverter) ethereumTransaction(canonicalEthTx []byte) (Transaction, error) {
	var tx EthereumTransaction
	if err := tx.DecodeCanonical(canonicalEthTx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal ethereum transaction")
	}
	return &tx, nil
}

func (c *ProtobufConverter) MicroBlock(mb *g.SignedMicroBlock) (MicroBlock, error) {
	if mb == nil {
		return MicroBlock{}, errors.New("empty signed microblock")
	}
	if mb.MicroBlock == nil {
		return MicroBlock{}, errors.New("empty microblock")
	}
	txs, err := c.SignedTransactions(mb.MicroBlock.Transactions)
	if err != nil {
		return MicroBlock{}, err
	}
	v := c.byte(mb.MicroBlock.Version)
	res := MicroBlock{
		VersionField:          v,
		Reference:             c.blockID(mb.MicroBlock.Reference),
		TotalResBlockSigField: c.signature(mb.MicroBlock.UpdatedBlockSignature),
		TotalBlockID:          c.blockID(mb.TotalBlockId),
		TransactionCount:      uint32(len(mb.MicroBlock.Transactions)),
		Transactions:          txs,
		SenderPK:              c.publicKey(mb.MicroBlock.SenderPublicKey),
		Signature:             c.signature(mb.Signature),
	}
	if c.err != nil {
		err := c.err
		c.reset()
		return MicroBlock{}, err
	}
	return res, nil
}

func (c *ProtobufConverter) Block(block *g.Block) (Block, error) {
	if block == nil {
		return Block{}, errors.New("empty block")
	}
	header, err := c.BlockHeader(block)
	if err != nil {
		return Block{}, err
	}
	txs, err := c.SignedTransactions(block.Transactions)
	if err != nil {
		return Block{}, err
	}
	if header.Version < NgBlockVersion {
		header.TransactionBlockLength = uint32(Transactions(txs).BinarySize() + 1)
	} else if header.Version <= RewardBlockVersion {
		header.TransactionBlockLength = uint32(Transactions(txs).BinarySize() + 4)
	}
	return Block{
		BlockHeader:  header,
		Transactions: txs,
	}, nil
}

func (c *ProtobufConverter) SignedTransactions(txs []*g.SignedTransaction) ([]Transaction, error) {
	res := make([]Transaction, len(txs))
	for i, stx := range txs {
		tx, err := c.SignedTransaction(stx)
		if err != nil {
			return nil, err
		}
		res[i] = tx
	}
	return res, nil
}

func (c *ProtobufConverter) features(features []uint32) []int16 {
	r := make([]int16, len(features))
	for i, f := range features {
		r[i] = int16(f)
	}
	return r
}

func (c *ProtobufConverter) consensus(header *g.Block_Header) NxtConsensus {
	if c.err != nil {
		return NxtConsensus{}
	}
	return NxtConsensus{
		GenSignature: header.GenerationSignature,
		BaseTarget:   c.uint64(header.BaseTarget),
	}
}

func (c *ProtobufConverter) BlockHeader(block *g.Block) (BlockHeader, error) {
	if block == nil {
		return BlockHeader{}, errors.New("empty block")
	}
	if block.Header == nil {
		return BlockHeader{}, errors.New("empty block header")
	}
	features := c.features(block.Header.FeatureVotes)
	consensus := c.consensus(block.Header)
	v := BlockVersion(c.byte(block.Header.Version))
	header := BlockHeader{
		Version:              v,
		Timestamp:            c.uint64(block.Header.Timestamp),
		Parent:               c.blockID(block.Header.Reference),
		FeaturesCount:        len(features),
		Features:             features,
		RewardVote:           block.Header.RewardVote,
		ConsensusBlockLength: uint32(consensus.BinarySize()),
		NxtConsensus:         consensus,
		TransactionCount:     len(block.Transactions),
		GeneratorPublicKey:   c.publicKey(block.Header.Generator),
		BlockSignature:       c.signature(block.Signature),
		TransactionsRoot:     block.Header.TransactionsRoot,
	}
	if c.err != nil {
		err := c.err
		c.reset()
		return BlockHeader{}, err
	}
	scheme := c.byte(block.Header.ChainId)
	if scheme == 0 {
		scheme = c.FallbackChainID
	}
	if err := header.GenerateBlockID(scheme); err != nil {
		return BlockHeader{}, err
	}
	return header, nil
}
