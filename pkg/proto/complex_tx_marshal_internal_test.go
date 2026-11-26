package proto

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// The functions below are verbatim copies (adapted to free functions) of the multi-buffer marshalling code of data
// entries, invoke arguments, payments, orders, Data, InvokeScript, MassTransfer and Exchange transactions that was
// replaced by the single-buffer append helpers. They are kept as oracles: the new code must produce the same bytes.

func oldDataEntryValue(e DataEntry) ([]byte, error) {
	switch te := e.(type) {
	case *IntegerDataEntry:
		buf := make([]byte, te.MarshaledValueSize())
		buf[0] = byte(DataInteger)
		binary.BigEndian.PutUint64(buf[1:], uint64(te.Value))
		return buf, nil
	case *BooleanDataEntry:
		buf := make([]byte, te.MarshaledValueSize())
		buf[0] = byte(DataBoolean)
		PutBool(buf[1:], te.Value)
		return buf, nil
	case *BinaryDataEntry:
		buf := make([]byte, te.MarshaledValueSize())
		buf[0] = byte(DataBinary)
		if err := PutBytesWithUInt16Len(buf[1:], te.Value); err != nil {
			return nil, errors.Wrap(err, "failed to marshal BinaryDataEntry value")
		}
		return buf, nil
	case *StringDataEntry:
		buf := make([]byte, te.MarshaledValueSize())
		buf[0] = byte(DataString)
		PutStringWithUInt16Len(buf[1:], te.Value)
		return buf, nil
	case *DeleteDataEntry:
		return []byte{byte(DataDelete)}, nil
	default:
		return nil, errors.Errorf("unsupported entry %T", e)
	}
}

func oldDataEntryMarshalBinary(e DataEntry) ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	PutStringWithUInt16Len(buf, e.GetKey())
	valueBytes, err := oldDataEntryValue(e)
	if err != nil {
		return nil, err
	}
	copy(buf[2+len(e.GetKey()):], valueBytes)
	return buf, nil
}

func oldDataWithProofsBody(tx *DataWithProofs) ([]byte, error) {
	var p int
	n := len(tx.Entries)
	el := tx.Entries.BinarySize()
	buf := make([]byte, dataWithProofsFixedBodyLen+el)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint16(buf[p:], uint16(n))
	p += 2
	for _, e := range tx.Entries {
		eb, err := oldDataEntryMarshalBinary(e)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal DataWithProofs transaction body to bytes")
		}
		copy(buf[p:], eb)
		p += e.BinarySize()
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	return buf, nil
}

func oldArgumentMarshalBinary(arg Argument) ([]byte, error) {
	switch a := arg.(type) {
	case *IntegerArgument:
		buf := make([]byte, a.BinarySize())
		buf[0] = byte(ArgumentInteger)
		binary.BigEndian.PutUint64(buf[1:], uint64(a.Value))
		return buf, nil
	case *BooleanArgument:
		buf := make([]byte, a.BinarySize())
		if a.Value {
			buf[0] = byte(ArgumentValueTrue)
		} else {
			buf[0] = byte(ArgumentValueFalse)
		}
		return buf, nil
	case *BinaryArgument:
		buf := make([]byte, a.BinarySize())
		buf[0] = byte(ArgumentBinary)
		if err := PutBytesWithUInt32Len(buf[1:], a.Value); err != nil {
			return nil, errors.Wrap(err, "failed to marshal BinaryArgument")
		}
		return buf, nil
	case *StringArgument:
		buf := make([]byte, a.BinarySize())
		buf[0] = byte(ArgumentString)
		PutStringWithUInt32Len(buf[1:], a.Value)
		return buf, nil
	case *ListArgument:
		buf := make([]byte, a.BinarySize())
		buf[0] = byte(ArgumentList)
		b, err := oldArgumentsMarshalBinary(a.Items)
		if err != nil {
			return nil, err
		}
		copy(buf[1:], b)
		return buf, nil
	default:
		return nil, errors.Errorf("unsupported argument %T", arg)
	}
}

func oldArgumentsMarshalBinary(a Arguments) ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	p := 0
	binary.BigEndian.PutUint32(buf, uint32(len(a)))
	p += 4
	for _, arg := range a {
		b, err := oldArgumentMarshalBinary(arg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal Arguments to bytes")
		}
		copy(buf[p:], b)
		p += len(b)
	}
	return buf, nil
}

func oldFunctionCallMarshalBinary(c FunctionCall) ([]byte, error) {
	if c.Default() {
		return []byte{0}, nil
	}
	buf := make([]byte, c.BinarySize())
	buf[0] = 1
	buf[1] = tokenFunctionCall
	buf[2] = tokenUserFunction
	PutStringWithUInt32Len(buf[3:], c.name)
	ab, err := oldArgumentsMarshalBinary(c.arguments)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal FunctionCall to bytes")
	}
	copy(buf[3+4+len(c.name):], ab)
	return buf, nil
}

func oldScriptPaymentMarshalBinary(p ScriptPayment) []byte {
	size := p.BinarySize()
	buf := make([]byte, size)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(size-2))
	pos += 2
	binary.BigEndian.PutUint64(buf[pos:], p.Amount)
	pos += 8
	ab, _ := p.Asset.MarshalBinary()
	copy(buf[pos:], ab)
	return buf
}

func oldScriptPaymentsMarshalBinary(sps ScriptPayments) []byte {
	buf := make([]byte, sps.BinarySize())
	p := 0
	binary.BigEndian.PutUint16(buf[p:], uint16(len(sps)))
	p += 2
	for _, sp := range sps {
		b := oldScriptPaymentMarshalBinary(sp)
		copy(buf[p:], b)
		p += len(b)
	}
	return buf
}

func oldInvokeScriptWithProofsBody(tx *InvokeScriptWithProofs, scheme Scheme) ([]byte, error) {
	p := 0
	buf := make([]byte, invokeScriptWithProofsFixedBodyLen+tx.ScriptRecipient.BinarySize()+
		tx.FunctionCall.BinarySize()+tx.Payments.BinarySize()+tx.FeeAsset.BinarySize())
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = scheme
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	rb, err := tx.ScriptRecipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal InvokeScriptWithProofs body")
	}
	copy(buf[p:], rb)
	p += tx.ScriptRecipient.BinarySize()
	fcb, err := oldFunctionCallMarshalBinary(tx.FunctionCall)
	if err != nil {
		return nil, err
	}
	copy(buf[p:], fcb)
	p += len(fcb)
	psb := oldScriptPaymentsMarshalBinary(tx.Payments)
	copy(buf[p:], psb)
	p += len(psb)
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	fab, _ := tx.FeeAsset.MarshalBinary()
	copy(buf[p:], fab)
	p += len(fab)
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func oldMassTransferEntryMarshalBinary(e *MassTransferEntry) ([]byte, error) {
	rb, err := e.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferEntry")
	}
	rl := e.Recipient.BinarySize()
	buf := make([]byte, massTransferEntryLen+rl)
	copy(buf, rb)
	binary.BigEndian.PutUint64(buf[rl:], e.Amount)
	return buf, nil
}

func oldMassTransferWithProofsBody(tx *MassTransferWithProofs) ([]byte, error) {
	var p int
	n := len(tx.Transfers)
	bl, al := tx.bodyAndAssetLen()
	buf := make([]byte, bl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	ab, _ := tx.Asset.MarshalBinary()
	copy(buf[p:], ab)
	p += 1 + al
	binary.BigEndian.PutUint16(buf[p:], uint16(n))
	p += 2
	for _, t := range tx.Transfers {
		tb, err := oldMassTransferEntryMarshalBinary(&t)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction body to bytes")
		}
		copy(buf[p:], tb)
		p += massTransferEntryLen + t.Recipient.BinarySize()
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	if err := PutBytesWithUInt16Len(buf[p:], tx.Attachment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction body to bytes")
	}
	return buf, nil
}

func oldOrderBodyMarshalBinary(o *OrderBody) []byte {
	var p int
	aal := 0
	if o.AssetPair.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	pal := 0
	if o.AssetPair.PriceAsset.Present {
		pal += crypto.DigestSize
	}
	buf := make([]byte, orderLen+aal+pal)
	copy(buf[0:], o.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], o.MatcherPK[:])
	p += crypto.PublicKeySize
	aa, _ := o.AssetPair.AmountAsset.MarshalBinary()
	copy(buf[p:], aa)
	p += 1 + aal
	pa, _ := o.AssetPair.PriceAsset.MarshalBinary()
	copy(buf[p:], pa)
	p += 1 + pal
	buf[p] = byte(o.OrderType)
	p++
	binary.BigEndian.PutUint64(buf[p:], o.Price)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Expiration)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.MatcherFee)
	return buf
}

func oldOrderV2Body(o *OrderV2) []byte {
	buf := make([]byte, orderV2FixedBodyLen+o.AssetPair.BinarySize()-2)
	buf[0] = byte(o.Version)
	copy(buf[1:], oldOrderBodyMarshalBinary(&o.OrderBody))
	return buf
}

func oldOrderV3Body(o *OrderV3) []byte {
	buf := make([]byte, orderV3FixedBodyLen+o.AssetPair.BinarySize()-2+o.MatcherFeeAsset.BinarySize()-1)
	pos := 0
	buf[pos] = byte(o.Version)
	pos++
	b := oldOrderBodyMarshalBinary(&o.OrderBody)
	copy(buf[pos:], b)
	pos += len(b)
	mfa, _ := o.MatcherFeeAsset.MarshalBinary()
	copy(buf[pos:], mfa)
	return buf
}

func oldConcat(a, b []byte) []byte {
	buf := make([]byte, len(a)+len(b))
	copy(buf, a)
	copy(buf[len(a):], b)
	return buf
}

func oldOrderMarshalBinary(order Order) ([]byte, error) {
	switch o := order.(type) {
	case *OrderV1:
		return oldConcat(oldOrderBodyMarshalBinary(&o.OrderBody), o.Signature[:]), nil
	case *OrderV2:
		return oldConcat(oldOrderV2Body(o), oldProofsMarshalBinary(o.Proofs)), nil
	case *OrderV3:
		return oldConcat(oldOrderV3Body(o), oldProofsMarshalBinary(o.Proofs)), nil
	default:
		return nil, errors.Errorf("unsupported order %T", order)
	}
}

func oldMarshalAsOrder(order Order) ([]byte, error) {
	b, err := oldOrderMarshalBinary(order)
	if err != nil {
		return nil, err
	}
	l := len(b)
	if order.GetVersion() == OrderVersionV1 {
		buf := make([]byte, 4+1+l)
		binary.BigEndian.PutUint32(buf, uint32(l))
		buf[4] = 1
		copy(buf[5:], b)
		return buf, nil
	}
	buf := make([]byte, 4+l)
	binary.BigEndian.PutUint32(buf, uint32(l))
	copy(buf[4:], b)
	return buf, nil
}

func oldExchangeTail(buf []byte, p int, price, amount, bmf, smf, fee, ts uint64) {
	for _, v := range []uint64{price, amount, bmf, smf, fee, ts} {
		binary.BigEndian.PutUint64(buf[p:], v)
		p += 8
	}
}

func oldExchangeWithProofsBody(tx *ExchangeWithProofs) ([]byte, error) {
	o1b, err := oldMarshalAsOrder(tx.Order1)
	if err != nil {
		return nil, err
	}
	o2b, err := oldMarshalAsOrder(tx.Order2)
	if err != nil {
		return nil, err
	}
	o1l, o2l := len(o1b), len(o2b)
	buf := make([]byte, exchangeWithProofsFixedBodyLen+(o1l-4)+(o2l-4))
	buf[0] = 0
	buf[1] = byte(tx.Type)
	buf[2] = tx.Version
	p := 3
	copy(buf[p:], o1b)
	p += o1l
	copy(buf[p:], o2b)
	p += o2l
	oldExchangeTail(buf, p, tx.Price, tx.Amount, tx.BuyMatcherFee, tx.SellMatcherFee, tx.Fee, tx.Timestamp)
	return buf, nil
}

func oldExchangeWithSigBody(tx *ExchangeWithSig) ([]byte, error) {
	bob, err := oldOrderMarshalBinary(tx.Order1)
	if err != nil {
		return nil, err
	}
	sob, err := oldOrderMarshalBinary(tx.Order2)
	if err != nil {
		return nil, err
	}
	bol, sol := len(bob), len(sob)
	buf := make([]byte, exchangeWithSigFixedBodyLen+bol+sol)
	buf[0] = byte(tx.Type)
	p := 1
	binary.BigEndian.PutUint32(buf[p:], uint32(bol))
	p += 4
	binary.BigEndian.PutUint32(buf[p:], uint32(sol))
	p += 4
	copy(buf[p:], bob)
	p += bol
	copy(buf[p:], sob)
	p += sol
	oldExchangeTail(buf, p, tx.Price, tx.Amount, tx.BuyMatcherFee, tx.SellMatcherFee, tx.Fee, tx.Timestamp)
	return buf, nil
}

// oldComplexTxMarshal returns the body and full binary bytes produced by the replaced implementation.
func oldComplexTxMarshal(tx Transaction, scheme Scheme) ([]byte, []byte, error) {
	switch t := tx.(type) {
	case *DataWithProofs:
		body, err := oldDataWithProofsBody(t)
		if err != nil {
			return nil, nil, err
		}
		full, err := oldWithProofs(body, t.Proofs)
		return body, full, err
	case *InvokeScriptWithProofs:
		body, err := oldInvokeScriptWithProofsBody(t, scheme)
		if err != nil {
			return nil, nil, err
		}
		full, err := oldWithProofs(body, t.Proofs)
		return body, full, err
	case *MassTransferWithProofs:
		body, err := oldMassTransferWithProofsBody(t)
		if err != nil {
			return nil, nil, err
		}
		return body, oldConcat(body, oldProofsMarshalBinary(t.Proofs)), nil
	case *ExchangeWithProofs:
		body, err := oldExchangeWithProofsBody(t)
		if err != nil {
			return nil, nil, err
		}
		return body, oldConcat(body, oldProofsMarshalBinary(t.Proofs)), nil
	case *ExchangeWithSig:
		body, err := oldExchangeWithSigBody(t)
		if err != nil {
			return nil, nil, err
		}
		return body, oldSigLast(t.Signature, body), nil
	default:
		return nil, nil, errors.Errorf("unsupported transaction %T", tx)
	}
}

// complexDataEntries returns entries of all types, including edge values, empty values and long values.
// If withDelete is true, a DeleteDataEntry is added (it is allowed only in protobuf transactions).
func complexDataEntries(n int, withDelete bool) DataEntries {
	entries := make(DataEntries, 0, n)
	for i := range n {
		key := fmt.Sprintf("key-%03d", i)
		var e DataEntry
		switch i % 9 {
		case 0:
			e = &IntegerDataEntry{Key: key, Value: int64(i) * 1_000_003}
		case 1:
			e = &IntegerDataEntry{Key: key, Value: -1 << 63}
		case 2:
			e = &BooleanDataEntry{Key: key, Value: true}
		case 3:
			e = &BooleanDataEntry{Key: key, Value: false}
		case 4:
			e = &BinaryDataEntry{Key: key, Value: []byte(fmt.Sprintf("binary-value-%d", i))}
		case 5:
			e = &BinaryDataEntry{Key: key, Value: nil}
		case 6:
			e = &StringDataEntry{Key: key, Value: strings.Repeat("ж", i)}
		case 7:
			e = &StringDataEntry{Key: key, Value: ""}
		default:
			if withDelete {
				e = &DeleteDataEntry{Key: key}
			} else {
				e = &IntegerDataEntry{Key: key, Value: 1<<63 - 1}
			}
		}
		entries = append(entries, e)
	}
	return entries
}

func complexArguments() map[string]Arguments {
	flat := Arguments{
		NewIntegerArgument(12345),
		NewIntegerArgument(-1),
		NewStringArgument("string argument"),
		NewStringArgument(""),
		&BinaryArgument{Value: []byte("binary argument")},
		&BinaryArgument{Value: nil},
		&BooleanArgument{Value: true},
		&BooleanArgument{Value: false},
	}
	nested := Arguments{
		NewIntegerArgument(1),
		&ListArgument{Items: Arguments{
			NewIntegerArgument(1), NewStringArgument("a"), &BooleanArgument{Value: true},
			&ListArgument{Items: Arguments{&BinaryArgument{Value: []byte{1, 2, 3}}, &ListArgument{}}},
		}},
		&ListArgument{Items: Arguments{}},
		NewStringArgument("tail"),
	}
	return map[string]Arguments{"noArgs": {}, "nilArgs": nil, "flat": flat, "nested": nested}
}

func complexPayments() map[string]ScriptPayments {
	return map[string]ScriptPayments{
		"noPayments": nil,
		"payments": {
			{Amount: 100_000, Asset: NewOptionalAssetWaves()},
			{Amount: 200_000, Asset: *NewOptionalAssetFromDigest(benchDigest(2))},
		},
	}
}

func complexOrder(t testing.TB, version byte, orderType OrderType, amountAsset OptionalAsset, sender,
	matcher KeyPair) Order {
	priceAsset := NewOptionalAssetWaves()
	const (
		price      = 100_000
		amount     = 1_000
		expiration = benchTimestamp + 1_000_000
		matcherFee = 300_000
	)
	switch version {
	case 1:
		o := NewUnsignedOrderV1(sender.Public, matcher.Public, amountAsset, priceAsset, orderType, price, amount,
			benchTimestamp, expiration, matcherFee)
		require.NoError(t, o.Sign(benchScheme, sender.Secret))
		return o
	case 2:
		o := NewUnsignedOrderV2(sender.Public, matcher.Public, amountAsset, priceAsset, orderType, price, amount,
			benchTimestamp, expiration, matcherFee)
		require.NoError(t, o.Sign(benchScheme, sender.Secret))
		return o
	case 3:
		o := NewUnsignedOrderV3(sender.Public, matcher.Public, amountAsset, priceAsset, orderType, price, amount,
			benchTimestamp, expiration, matcherFee, *NewOptionalAssetFromDigest(benchDigest(9)))
		require.NoError(t, o.Sign(benchScheme, sender.Secret))
		return o
	default:
		require.FailNow(t, "unsupported order version")
		return nil
	}
}

// binaryOrder is implemented by the orders that have a binary representation.
type binaryOrder interface {
	Order
	MarshalBinary() ([]byte, error)
	BodyMarshalBinary() ([]byte, error)
}

// externalDataEntry and externalArgument hide the concrete types from the append helpers,
// so the MarshalBinary fallback is used for them.
type externalDataEntry struct{ *IntegerDataEntry }

type externalArgument struct{ *IntegerArgument }

//nolint:gocognit // Nested loops over a flat list of test vector parameters.
func complexTxCases(t testing.TB) []simpleTxCase {
	kp := benchKeyPair(t, "complex-sender")
	pk := kp.Public
	ts := uint64(benchTimestamp)
	assets := map[string]OptionalAsset{
		"waves": NewOptionalAssetWaves(),
		"asset": *NewOptionalAssetFromDigest(benchDigest(5)),
	}
	attachments := map[string]Attachment{"nilAtt": nil, "emptyAtt": {}, "att": []byte("some attachment")}
	var cases []simpleTxCase
	add := func(name string, tx Transaction) { cases = append(cases, simpleTxCase{name: name, tx: tx}) }

	for pn, proofs := range simpleTxProofs() {
		for _, v := range []byte{1, 2} {
			for _, n := range []int{0, 1, 9, 100} {
				tx := NewUnsignedDataWithProofs(v, pk, 1_500_000, ts)
				tx.Entries = complexDataEntries(n, v > 1)
				tx.Proofs = proofs
				add(fmt.Sprintf("DataWithProofs/v%d/%d/%s", v, n, pn), tx)
			}
			for rn, rcp := range simpleTxRecipients(t) {
				for an, args := range complexArguments() {
					for payn, payments := range complexPayments() {
						for fn, fa := range assets {
							tx := NewUnsignedInvokeScriptWithProofs(v, pk, rcp, NewFunctionCall("call", args), payments,
								fa, 500_000, ts)
							tx.Proofs = proofs
							add(fmt.Sprintf("InvokeScriptWithProofs/v%d/%s/%s/%s/%s/%s", v, rn, an, payn, fn, pn), tx)
						}
					}
				}
				tx := NewUnsignedInvokeScriptWithProofs(v, pk, rcp, FunctionCall{}, nil, assets["waves"], 500_000, ts)
				tx.Proofs = proofs
				add(fmt.Sprintf("InvokeScriptWithProofs/v%d/%s/default/%s", v, rn, pn), tx)
			}
			for an, asset := range assets {
				for atn, att := range attachments {
					for _, n := range []int{0, 1, 100} {
						transfers := make([]MassTransferEntry, n)
						recipients := simpleTxRecipients(t)
						for i := range transfers {
							rcp := recipients["addr"]
							if i%2 == 1 {
								rcp = recipients["alias"]
							}
							transfers[i] = MassTransferEntry{Recipient: rcp, Amount: uint64(i+1) * 1_000}
						}
						tx := NewUnsignedMassTransferWithProofs(v, pk, asset, transfers, 5_100_000, ts, att)
						tx.Proofs = proofs
						add(fmt.Sprintf("MassTransferWithProofs/v%d/%s/%s/%d/%s", v, an, atn, n, pn), tx)
					}
				}
			}
		}
		for _, v1 := range []byte{1, 2, 3} {
			for _, v2 := range []byte{1, 2, 3} {
				for an, asset := range assets {
					buy := complexOrder(t, v1, Buy, asset, benchKeyPair(t, "buyer"), kp)
					sell := complexOrder(t, v2, Sell, asset, benchKeyPair(t, "seller"), kp)
					tx := NewUnsignedExchangeWithProofs(2, buy, sell, 100_000, 1_000, 300_000, 300_000, 700_000, ts)
					tx.Proofs = proofs
					add(fmt.Sprintf("ExchangeWithProofs/v2/o%d/o%d/%s/%s", v1, v2, an, pn), tx)
				}
			}
		}
	}
	for an, asset := range assets {
		buy := complexOrder(t, 1, Buy, asset, benchKeyPair(t, "buyer"), kp)
		sell := complexOrder(t, 1, Sell, asset, benchKeyPair(t, "seller"), kp)
		buyV1, ok := buy.(*OrderV1)
		require.True(t, ok)
		sellV1, ok := sell.(*OrderV1)
		require.True(t, ok)
		tx := NewUnsignedExchangeWithSig(buyV1, sellV1, 100_000, 1_000, 300_000, 300_000, 700_000, ts)
		require.NoError(t, tx.Sign(benchScheme, kp.Secret))
		add("ExchangeWithSig/"+an, tx)
	}

	// Real signed transactions from the benchmark fixtures.
	signed := []Transaction{
		benchData(t, 1, kp),
		benchInvoke(t, 1, kp),
		benchMassTransfer(t, 1, kp),
	}
	for i, tx := range signed {
		add(fmt.Sprintf("Signed/%d/%T", i, tx), tx)
	}
	return cases
}

func TestComplexTxMarshalBinaryGolden(t *testing.T) {
	for _, c := range complexTxCases(t) {
		t.Run(c.name, func(t *testing.T) {
			expectedBody, expectedFull, err := oldComplexTxMarshal(c.tx, benchScheme)
			require.NoError(t, err)

			body, err := c.tx.BodyMarshalBinary(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, expectedBody, body)
			assert.Equal(t, len(body), cap(body), "body buffer must be sized exactly")

			full, err := c.tx.MarshalBinary(benchScheme)
			require.NoError(t, err)
			assert.Equal(t, expectedFull, full)
			assert.Equal(t, len(full), cap(full), "transaction buffer must be sized exactly")

			sizer, ok := c.tx.(binarySizer)
			require.True(t, ok)
			assert.Equal(t, sizer.BinarySize(), len(full))

			if !IsProtobufTx(c.tx) {
				bodyFromHelper, hErr := MarshalTxBody(benchScheme, c.tx)
				require.NoError(t, hErr)
				assert.Equal(t, expectedBody, bodyFromHelper)
				// Round trip through the binary format.
				parsed, pErr := BytesToTransaction(full, benchScheme)
				require.NoError(t, pErr)
				again, mErr := parsed.MarshalBinary(benchScheme)
				require.NoError(t, mErr)
				assert.Equal(t, full, again)
			}
		})
	}
}

func TestDataEntriesAppendBinaryGolden(t *testing.T) {
	for _, e := range complexDataEntries(18, true) {
		expected, err := oldDataEntryMarshalBinary(e)
		require.NoError(t, err)
		got, err := e.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, expected, got, "%T", e)
		assert.Equal(t, e.BinarySize(), len(got), "%T", e)
		assert.Equal(t, len(got), cap(got), "%T", e)

		expectedValue, err := oldDataEntryValue(e)
		require.NoError(t, err)
		gotValue, err := e.MarshalValue()
		require.NoError(t, err)
		assert.Equal(t, expectedValue, gotValue, "%T", e)
		assert.Equal(t, e.MarshaledValueSize(), len(gotValue), "%T", e)

		prefix := []byte{0xAA}
		appended, err := appendDataEntryBinary(prefix, e)
		require.NoError(t, err)
		assert.Equal(t, append([]byte{0xAA}, expected...), appended, "%T", e)
	}
	// Implementations from outside the package go through the MarshalBinary fallback.
	v := &IntegerDataEntry{Key: "value", Value: 7}
	expected, err := oldDataEntryMarshalBinary(v)
	require.NoError(t, err)
	got, err := appendDataEntryBinary([]byte{0xAA}, externalDataEntry{v})
	require.NoError(t, err)
	assert.Equal(t, append([]byte{0xAA}, expected...), got)
}

func TestArgumentsAppendBinaryGolden(t *testing.T) {
	for name, args := range complexArguments() {
		expected, err := oldArgumentsMarshalBinary(args)
		require.NoError(t, err)
		got, err := args.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, expected, got, name)
		assert.Equal(t, args.BinarySize(), len(got), name)
		assert.Equal(t, len(got), cap(got), name)
		for i, arg := range args {
			expectedArg, aErr := oldArgumentMarshalBinary(arg)
			require.NoError(t, aErr)
			gotArg, aErr := arg.MarshalBinary()
			require.NoError(t, aErr)
			assert.Equal(t, expectedArg, gotArg, "%s/%d", name, i)
			assert.Equal(t, arg.BinarySize(), len(gotArg), "%s/%d", name, i)
		}

		fc := NewFunctionCall("function", args)
		expectedFC, err := oldFunctionCallMarshalBinary(fc)
		require.NoError(t, err)
		gotFC, err := fc.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, expectedFC, gotFC, name)
		assert.Equal(t, fc.BinarySize(), len(gotFC), name)
	}
	gotDefault, err := FunctionCall{}.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, []byte{0}, gotDefault)

	// Implementations from outside the package go through the MarshalBinary fallback.
	ia := NewIntegerArgument(-7)
	expected, err := oldArgumentsMarshalBinary(Arguments{ia})
	require.NoError(t, err)
	got, err := Arguments{externalArgument{ia}}.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestScriptPaymentsAppendBinaryGolden(t *testing.T) {
	for name, sps := range complexPayments() {
		got, err := sps.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, oldScriptPaymentsMarshalBinary(sps), got, name)
		assert.Equal(t, sps.BinarySize(), len(got), name)
		for _, sp := range sps {
			gotSP, spErr := sp.MarshalBinary()
			require.NoError(t, spErr)
			assert.Equal(t, oldScriptPaymentMarshalBinary(sp), gotSP, name)
		}
	}
}

func TestOrdersAppendBinaryGolden(t *testing.T) {
	sender := benchKeyPair(t, "order-sender")
	matcher := benchKeyPair(t, "order-matcher")
	for _, v := range []byte{1, 2, 3} {
		for _, asset := range []OptionalAsset{NewOptionalAssetWaves(), *NewOptionalAssetFromDigest(benchDigest(4))} {
			o, ok := complexOrder(t, v, Sell, asset, sender, matcher).(binaryOrder)
			require.True(t, ok)
			expected, err := oldOrderMarshalBinary(o)
			require.NoError(t, err)
			got, err := o.MarshalBinary()
			require.NoError(t, err)
			assert.Equal(t, expected, got, "v%d", v)
			assert.Equal(t, o.BinarySize(), len(got), "v%d", v)
			assert.Equal(t, len(got), cap(got), "v%d", v)

			body, err := o.BodyMarshalBinary()
			require.NoError(t, err)
			switch to := o.(type) {
			case *OrderV1:
				assert.Equal(t, oldOrderBodyMarshalBinary(&to.OrderBody), body)
			case *OrderV2:
				assert.Equal(t, oldOrderV2Body(to), body)
			case *OrderV3:
				assert.Equal(t, oldOrderV3Body(to), body)
			}
			verified, err := o.Verify(benchScheme)
			require.NoError(t, err)
			assert.True(t, verified)
		}
	}
}

func TestComplexTxMarshalBinaryErrors(t *testing.T) {
	kp := benchKeyPair(t, "complex-sender")
	ts := uint64(benchTimestamp)
	waves := NewOptionalAssetWaves()
	rcp := benchRecipient(t, "complex-recipient")
	buy := complexOrder(t, 2, Buy, waves, benchKeyPair(t, "buyer"), kp)
	sell := complexOrder(t, 2, Sell, waves, benchKeyPair(t, "seller"), kp)

	t.Run("no proofs", func(t *testing.T) {
		txs := []Transaction{
			NewUnsignedDataWithProofs(1, kp.Public, 1, ts),
			NewUnsignedInvokeScriptWithProofs(1, kp.Public, rcp, FunctionCall{}, nil, waves, 1, ts),
			NewUnsignedMassTransferWithProofs(1, kp.Public, waves, nil, 1, ts, nil),
			NewUnsignedExchangeWithProofs(2, buy, sell, 1, 1, 1, 1, 1, ts),
		}
		for _, tx := range txs {
			_, err := tx.MarshalBinary(benchScheme)
			require.ErrorContains(t, err, "no proofs", "%T", tx)
		}
	})
	t.Run("no signature", func(t *testing.T) {
		b1, ok := complexOrder(t, 1, Buy, waves, benchKeyPair(t, "buyer"), kp).(*OrderV1)
		require.True(t, ok)
		s1, ok := complexOrder(t, 1, Sell, waves, benchKeyPair(t, "seller"), kp).(*OrderV1)
		require.True(t, ok)
		tx := NewUnsignedExchangeWithSig(b1, s1, 1, 1, 1, 1, 1, ts)
		_, err := tx.MarshalBinary(benchScheme)
		require.ErrorContains(t, err, "no signature")
		unsigned := NewUnsignedOrderV1(kp.Public, kp.Public, waves, waves, Buy, 1, 1, ts, ts, 1)
		tx = NewUnsignedExchangeWithSig(unsigned, s1, 1, 1, 1, 1, 1, ts)
		_, err = tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "no signature")
	})
	t.Run("order without proofs", func(t *testing.T) {
		for _, o := range []binaryOrder{
			NewUnsignedOrderV2(kp.Public, kp.Public, waves, waves, Buy, 1, 1, ts, ts, 1),
			NewUnsignedOrderV3(kp.Public, kp.Public, waves, waves, Buy, 1, 1, ts, ts, 1, waves),
		} {
			_, err := o.MarshalBinary()
			require.ErrorContains(t, err, "no proofs", "%T", o)
			tx := NewUnsignedExchangeWithProofs(2, o, sell, 1, 1, 1, 1, 1, ts)
			_, err = tx.BodyMarshalBinary(benchScheme)
			require.ErrorContains(t, err, "no proofs", "%T", o)
		}
	})
	t.Run("unsupported order version", func(t *testing.T) {
		o4 := benchOrder(t, 4, Buy, benchKeyPair(t, "buyer"), kp)
		tx := NewUnsignedExchangeWithProofs(3, o4, o4, 1, 1, 1, 1, 1, ts)
		_, err := tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "order of version 4 is not supported")
		tx = NewUnsignedExchangeWithProofs(2, buy, nil, 1, 1, 1, 1, 1, ts)
		_, err = tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "empty order")
		_, err = tx.MarshalBinary(benchScheme)
		require.ErrorContains(t, err, "empty order")
		sigTx := &ExchangeWithSig{Type: ExchangeTransaction, Signature: &crypto.Signature{}}
		_, err = sigTx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "empty order")
		_, err = sigTx.MarshalBinary(benchScheme)
		require.ErrorContains(t, err, "empty order")
	})
	t.Run("mismatched order type", func(t *testing.T) {
		o := &OrderV2{Version: 1, Proofs: NewProofs()}
		tx := NewUnsignedExchangeWithProofs(2, o, sell, 1, 1, 1, 1, 1, ts)
		_, err := tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "failed to cast an order with version 1 to OrderV1")
	})
	t.Run("nil recipient", func(t *testing.T) {
		mt := NewUnsignedMassTransferWithProofs(1, kp.Public, waves, []MassTransferEntry{{Amount: 1}}, 1, ts, nil)
		_, err := mt.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "recipient is <nil>")
		inv := NewUnsignedInvokeScriptWithProofs(1, kp.Public, Recipient{}, FunctionCall{}, nil, waves, 1, ts)
		_, err = inv.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "recipient is <nil>")
	})
	t.Run("too long values", func(t *testing.T) {
		data := NewUnsignedDataWithProofs(1, kp.Public, 1, ts)
		data.Entries = DataEntries{&BinaryDataEntry{Key: "k", Value: make([]byte, 1<<15)}}
		_, err := data.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "invalid data length")
		_, oldErr := oldDataWithProofsBody(data)
		require.Error(t, oldErr)

		mt := NewUnsignedMassTransferWithProofs(1, kp.Public, waves, nil, 1, ts, make([]byte, 1<<15))
		_, err = mt.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "invalid data length")
		_, oldErr = oldMassTransferWithProofsBody(mt)
		require.Error(t, oldErr)
	})
}

// TestDataWithProofsUnmarshalEntries checks that the pre-sized entries and the map-based duplicate key check behave
// exactly as the replaced linear scan through AppendEntry.
func TestDataWithProofsUnmarshalEntries(t *testing.T) {
	kp := benchKeyPair(t, "complex-sender")
	ts := uint64(benchTimestamp)
	for _, n := range []int{0, 1, dataEntriesLinearScanMax, dataEntriesLinearScanMax + 1, 100} {
		tx := NewUnsignedDataWithProofs(1, kp.Public, 1_500_000, ts)
		tx.Entries = complexDataEntries(n, false)
		require.NoError(t, tx.Sign(benchScheme, kp.Secret))
		b, err := tx.MarshalBinary(benchScheme)
		require.NoError(t, err)

		var parsed DataWithProofs
		require.NoError(t, parsed.UnmarshalBinary(b, benchScheme), n)
		require.Len(t, parsed.Entries, n)
		assert.Equal(t, n, cap(parsed.Entries), "entries must be pre-sized, n=%d", n)
		again, err := parsed.MarshalBinary(benchScheme)
		require.NoError(t, err)
		assert.Equal(t, b, again, n)

		// Unmarshal into a used transaction replaces the entries.
		require.NoError(t, parsed.UnmarshalBinary(b, benchScheme), n)
		require.Len(t, parsed.Entries, n)
		again, err = parsed.MarshalBinary(benchScheme)
		require.NoError(t, err)
		assert.Equal(t, b, again, n)
	}

	for _, n := range []int{2, dataEntriesLinearScanMax, dataEntriesLinearScanMax + 1, 100} {
		for _, dupAt := range []int{1, n - 1} {
			tx := NewUnsignedDataWithProofs(1, kp.Public, 1_500_000, ts)
			tx.Entries = complexDataEntries(n, false)
			dup, ok := tx.Entries[dupAt].(interface{ SetKey(string) })
			require.True(t, ok)
			dup.SetKey(tx.Entries[0].GetKey())
			require.NoError(t, tx.Sign(benchScheme, kp.Secret))
			b, err := tx.MarshalBinary(benchScheme)
			require.NoError(t, err)

			var parsed DataWithProofs
			err = parsed.UnmarshalBinary(b, benchScheme)
			require.EqualError(t, err, fmt.Sprintf("failed to unmarshal DataWithProofs transaction from bytes: "+
				"failed to unmarshal DataWithProofs transaction body from bytes: key '%s' already exist",
				tx.Entries[0].GetKey()), "n=%d, dupAt=%d", n, dupAt)
		}
	}

	for _, n := range []int{1, dataEntriesLinearScanMax + 1} {
		tx := NewUnsignedDataWithProofs(1, kp.Public, 1_500_000, ts)
		tx.Entries = complexDataEntries(n, false)
		tx.Entries[n-1].SetKey("")
		require.NoError(t, tx.Sign(benchScheme, kp.Secret))
		b, err := tx.MarshalBinary(benchScheme)
		require.NoError(t, err)
		var parsed DataWithProofs
		err = parsed.UnmarshalBinary(b, benchScheme)
		require.ErrorContains(t, err, "empty keys are not allowed", n)
	}

	t.Run("forged count", func(t *testing.T) {
		tx := NewUnsignedDataWithProofs(1, kp.Public, 1_500_000, ts)
		tx.Entries = complexDataEntries(1, false)
		require.NoError(t, tx.Sign(benchScheme, kp.Secret))
		b, err := tx.MarshalBinary(benchScheme)
		require.NoError(t, err)
		binary.BigEndian.PutUint16(b[1+2+crypto.PublicKeySize:], 0xffff)
		var parsed DataWithProofs
		require.Error(t, parsed.UnmarshalBinary(b, benchScheme))
		assert.LessOrEqual(t, cap(parsed.Entries), len(b)/dataEntryMinBinarySize)
	})
}

func BenchmarkComplexTxMarshalBinary(b *testing.B) {
	kp := benchKeyPair(b, "complex-sender")
	txs := []Transaction{
		benchData(b, 1, kp),
		benchInvoke(b, 1, kp),
		benchMassTransfer(b, 1, kp),
	}
	buy := complexOrder(b, 2, Buy, NewOptionalAssetWaves(), benchKeyPair(b, "buyer"), kp)
	sell := complexOrder(b, 3, Sell, NewOptionalAssetWaves(), benchKeyPair(b, "seller"), kp)
	exchange := NewUnsignedExchangeWithProofs(2, buy, sell, 100_000, 1_000, 300_000, 300_000, 700_000, benchTimestamp)
	require.NoError(b, exchange.Sign(benchScheme, kp.Secret))
	txs = append(txs, exchange)
	for _, tx := range txs {
		name := fmt.Sprintf("%T", tx)[len("*proto."):]
		b.Run(name+"/old", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := oldComplexTxMarshal(tx, benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/new", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := tx.BodyMarshalBinary(benchScheme); err != nil {
					b.Fatal(err)
				}
				if _, err := tx.MarshalBinary(benchScheme); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDataWithProofsUnmarshalBinary(b *testing.B) {
	kp := benchKeyPair(b, "complex-sender")
	tx := benchData(b, 1, kp)
	data, err := tx.MarshalBinary(benchScheme)
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		var parsed DataWithProofs
		if uErr := parsed.UnmarshalBinary(data, benchScheme); uErr != nil {
			b.Fatal(uErr)
		}
	}
}
