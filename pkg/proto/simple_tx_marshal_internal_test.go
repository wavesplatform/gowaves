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

// The functions below are verbatim copies of the multi-buffer marshalling code that was replaced by the single-buffer
// append helpers. They are kept as oracles: the new code must produce exactly the same bytes.

func oldProofsMarshalBinary(p *ProofsV1) []byte {
	buf := make([]byte, p.BinarySize())
	pos := 0
	buf[pos] = proofsVersion
	pos++
	binary.BigEndian.PutUint16(buf[pos:], uint16(len(p.Proofs)))
	pos += 2
	for _, e := range p.Proofs {
		el := len(e)
		binary.BigEndian.PutUint16(buf[pos:], uint16(el))
		pos += 2
		copy(buf[pos:], e)
		pos += el
	}
	return buf
}

func oldIssueMarshalBinary(i *Issue) []byte {
	nl := len(i.Name)
	dl := len(i.Description)
	buf := make([]byte, issueLen+nl+dl)
	p := 0
	copy(buf[p:], i.SenderPK[:])
	p += crypto.PublicKeySize
	PutStringWithUInt16Len(buf[p:], i.Name)
	p += 2 + nl
	PutStringWithUInt16Len(buf[p:], i.Description)
	p += 2 + dl
	binary.BigEndian.PutUint64(buf[p:], i.Quantity)
	p += 8
	buf[p] = i.Decimals
	p++
	PutBool(buf[p:], i.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], i.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], i.Timestamp)
	return buf
}

func oldTransferMarshalBinary(tr *Transfer) ([]byte, error) {
	p := 0
	aal := 0
	if tr.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	fal := 0
	if tr.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	rb, err := tr.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	rl := len(rb)
	att := tr.Attachment
	atl := att.Size()
	buf := make([]byte, transferLen+aal+fal+atl+rl)
	copy(buf[p:], tr.SenderPK[:])
	p += crypto.PublicKeySize
	aab, err := tr.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], aab)
	p += 1 + aal
	fab, err := tr.FeeAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], fab)
	p += 1 + fal
	binary.BigEndian.PutUint64(buf[p:], tr.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tr.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tr.Fee)
	p += 8
	copy(buf[p:], rb)
	p += rl
	attBytes := att.Bytes()
	if err = PutBytesWithUInt16Len(buf[p:], attBytes); err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	return buf, nil
}

func oldReissueMarshalBinary(r *Reissue) []byte {
	p := 0
	buf := make([]byte, reissueLen)
	copy(buf[p:], r.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], r.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], r.Quantity)
	p += 8
	PutBool(buf[p:], r.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], r.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], r.Timestamp)
	return buf
}

func oldBurnMarshalBinary(b *Burn) []byte {
	buf := make([]byte, burnLen)
	p := 0
	copy(buf[p:], b.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], b.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], b.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], b.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], b.Timestamp)
	return buf
}

func oldLeaseMarshalBinary(l *Lease) ([]byte, error) {
	rl := l.Recipient.BinarySize()
	buf := make([]byte, leaseLen+rl)
	p := 0
	copy(buf[p:], l.SenderPK[:])
	p += crypto.PublicKeySize
	rb, err := l.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal lease to bytes")
	}
	copy(buf[p:], rb)
	p += rl
	binary.BigEndian.PutUint64(buf[p:], l.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], l.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], l.Timestamp)
	return buf, nil
}

func oldLeaseCancelMarshalBinary(lc *LeaseCancel) []byte {
	buf := make([]byte, leaseCancelLen)
	p := 0
	copy(buf[p:], lc.SenderPK[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint64(buf[p:], lc.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], lc.Timestamp)
	p += 8
	copy(buf[p:], lc.LeaseID[:])
	return buf
}

func oldCreateAliasMarshalBinary(ca *CreateAlias) []byte {
	p := 0
	buf := make([]byte, createAliasLen+len(ca.Alias.Alias))
	copy(buf[p:], ca.SenderPK[:])
	p += crypto.PublicKeySize
	ab := ca.Alias.Bytes()
	al := len(ab)
	binary.BigEndian.PutUint16(buf[p:], uint16(al))
	p += 2
	copy(buf[p:], ab)
	p += al
	binary.BigEndian.PutUint64(buf[p:], ca.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], ca.Timestamp)
	return buf
}

func oldIssueWithProofsBody(tx *IssueWithProofs, scheme Scheme) ([]byte, error) {
	var p int
	nl := len(tx.Name)
	dl := len(tx.Description)
	sl := len(tx.Script)
	if sl > 0 {
		sl += 2
	}
	buf := make([]byte, issueWithProofsFixedBodyLen+nl+dl+sl)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = scheme
	p = 3
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	PutStringWithUInt16Len(buf[p:], tx.Name)
	p += 2 + nl
	PutStringWithUInt16Len(buf[p:], tx.Description)
	p += 2 + dl
	binary.BigEndian.PutUint64(buf[p:], tx.Quantity)
	p += 8
	buf[p] = tx.Decimals
	p++
	PutBool(buf[p:], tx.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	PutBool(buf[p:], tx.NonEmptyScript())
	p++
	if tx.NonEmptyScript() {
		if err := PutBytesWithUInt16Len(buf[p:], tx.Script); err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func oldPrefixed(prefix []byte, body []byte, size int) []byte {
	buf := make([]byte, size)
	copy(buf, prefix)
	copy(buf[len(prefix):], body)
	return buf
}

func oldWithProofs(body []byte, proofs *ProofsV1) ([]byte, error) {
	if proofs == nil {
		return nil, errors.New("no proofs")
	}
	pb := oldProofsMarshalBinary(proofs)
	buf := make([]byte, 1+len(body)+len(pb))
	copy(buf[1:], body)
	copy(buf[1+len(body):], pb)
	return buf, nil
}

func oldSigFirst(t TransactionType, sig *crypto.Signature, body []byte) []byte {
	buf := make([]byte, 1+crypto.SignatureSize+len(body))
	buf[0] = byte(t)
	copy(buf[1:], sig[:])
	copy(buf[1+crypto.SignatureSize:], body)
	return buf
}

func oldSigLast(sig *crypto.Signature, body []byte) []byte {
	buf := make([]byte, len(body)+crypto.SignatureSize)
	copy(buf, body)
	copy(buf[len(body):], sig[:])
	return buf
}

// oldSimpleTxMarshal returns the body and full binary bytes produced by the replaced implementation.
func oldSimpleTxMarshal(tx Transaction, scheme Scheme) ([]byte, []byte, error) {
	var body []byte
	switch tx := tx.(type) {
	case *IssueWithProofs:
		b, err := oldIssueWithProofsBody(tx, scheme)
		if err != nil {
			return nil, nil, err
		}
		full, err := oldWithProofs(b, tx.Proofs)
		return b, full, err
	case *TransferWithProofs:
		b, err := oldTransferMarshalBinary(&tx.Transfer)
		if err != nil {
			return nil, nil, err
		}
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version}, b, 2+len(b))
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *ReissueWithProofs:
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version, scheme}, oldReissueMarshalBinary(&tx.Reissue),
			reissueWithProofsBodyLen)
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *BurnWithProofs:
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version, scheme}, oldBurnMarshalBinary(&tx.Burn),
			burnWithProofsBodyLen)
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *LeaseWithProofs:
		b, err := oldLeaseMarshalBinary(&tx.Lease)
		if err != nil {
			return nil, nil, err
		}
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version, 0}, b, leaseWithProofsBodyLen+tx.Recipient.BinarySize())
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *LeaseCancelWithProofs:
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version, scheme}, oldLeaseCancelMarshalBinary(&tx.LeaseCancel),
			leaseCancelWithProofsBodyLen)
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *CreateAliasWithProofs:
		body = oldPrefixed([]byte{byte(tx.Type), tx.Version}, oldCreateAliasMarshalBinary(&tx.CreateAlias),
			createAliasWithProofsFixedBodyLen+len(tx.Alias.Alias))
		full, err := oldWithProofs(body, tx.Proofs)
		return body, full, err
	case *IssueWithSig:
		b := oldIssueMarshalBinary(&tx.Issue)
		body = oldPrefixed([]byte{byte(tx.Type)}, b, 1+len(b))
		return body, oldSigFirst(tx.Type, tx.Signature, body), nil
	case *TransferWithSig:
		b, err := oldTransferMarshalBinary(&tx.Transfer)
		if err != nil {
			return nil, nil, err
		}
		body = oldPrefixed([]byte{byte(tx.Type)}, b, 1+len(b))
		return body, oldSigFirst(tx.Type, tx.Signature, body), nil
	case *ReissueWithSig:
		body = oldPrefixed([]byte{byte(tx.Type)}, oldReissueMarshalBinary(&tx.Reissue), reissueWithSigBodyLen)
		return body, oldSigFirst(tx.Type, tx.Signature, body), nil
	case *BurnWithSig:
		body = oldPrefixed([]byte{byte(tx.Type)}, oldBurnMarshalBinary(&tx.Burn), burnWithSigBodyLen)
		return body, oldSigLast(tx.Signature, body), nil
	case *LeaseWithSig:
		b, err := oldLeaseMarshalBinary(&tx.Lease)
		if err != nil {
			return nil, nil, err
		}
		body = oldPrefixed([]byte{byte(tx.Type)}, b, leaseWithSigBodyLen+tx.Recipient.BinarySize())
		return body, oldSigLast(tx.Signature, body), nil
	case *LeaseCancelWithSig:
		body = oldPrefixed([]byte{byte(tx.Type)}, oldLeaseCancelMarshalBinary(&tx.LeaseCancel),
			leaseCancelWithSigBodyLen)
		return body, oldSigLast(tx.Signature, body), nil
	case *CreateAliasWithSig:
		body = oldPrefixed([]byte{byte(tx.Type)}, oldCreateAliasMarshalBinary(&tx.CreateAlias),
			createAliasWithSigFixedBodyLen+len(tx.Alias.Alias))
		return body, oldSigLast(tx.Signature, body), nil
	default:
		return nil, nil, errors.Errorf("unsupported transaction %T", tx)
	}
}

type simpleTxCase struct {
	name string
	tx   Transaction
}

type binarySizer interface {
	BinarySize() int
}

func simpleTxRecipients(t testing.TB) map[string]Recipient {
	addr := benchRecipient(t, "simple-recipient")
	alias := NewAlias(benchScheme, "some-alias")
	return map[string]Recipient{"addr": addr, "alias": NewRecipientFromAlias(*alias)}
}

func simpleTxProofs() map[string]*ProofsV1 {
	sig := make([]byte, crypto.SignatureSize)
	for i := range sig {
		sig[i] = byte(i + 1)
	}
	return map[string]*ProofsV1{
		"noProofs":   NewProofs(),
		"oneProof":   {Version: proofsVersion, Proofs: []B58Bytes{sig}},
		"manyProofs": {Version: proofsVersion, Proofs: []B58Bytes{sig, {}, {1, 2, 3}, sig[:proofMaxSize]}},
	}
}

//nolint:gocognit // Nested loops over a flat list of test vector parameters.
func simpleTxCases(t testing.TB) []simpleTxCase {
	kp := benchKeyPair(t, "simple-sender")
	pk := kp.Public
	asset := benchDigest(7)
	alias := NewAlias(benchScheme, "new-alias")
	assets := map[string]OptionalAsset{"waves": NewOptionalAssetWaves(), "asset": *NewOptionalAssetFromDigest(asset)}
	attachments := map[string]Attachment{"nilAtt": nil, "emptyAtt": {}, "att": []byte("some attachment")}
	scripts := map[string]Script{"noScript": nil, "emptyScript": {}, "script": {1, 2, 3, 4, 5}}
	ts := uint64(benchTimestamp)
	sig := crypto.Signature{}
	for i := range sig {
		sig[i] = byte(0xff - i)
	}
	var cases []simpleTxCase
	add := func(name string, tx Transaction) { cases = append(cases, simpleTxCase{name: name, tx: tx}) }

	for pn, proofs := range simpleTxProofs() {
		for _, v := range []byte{1, 2, 3} {
			for sn, script := range scripts {
				tx := NewUnsignedIssueWithProofs(v, pk, "name", "description", 1000, 8, true, script, ts, 100)
				tx.Proofs = proofs
				add(fmt.Sprintf("IssueWithProofs/v%d/%s/%s", v, sn, pn), tx)
			}
			for rn, rcp := range simpleTxRecipients(t) {
				for an, aa := range assets {
					for fn, fa := range assets {
						for atn, att := range attachments {
							tx := NewUnsignedTransferWithProofs(v, pk, aa, fa, ts, 12345, 100, rcp, att)
							tx.Proofs = proofs
							add(fmt.Sprintf("TransferWithProofs/v%d/%s/%s/%s/%s/%s", v, rn, an, fn, atn, pn), tx)
						}
					}
				}
				tx := NewUnsignedLeaseWithProofs(v, pk, rcp, 1000, 100, ts)
				tx.Proofs = proofs
				add(fmt.Sprintf("LeaseWithProofs/v%d/%s/%s", v, rn, pn), tx)
			}
			reissue := NewUnsignedReissueWithProofs(v, pk, asset, 1000, false, ts, 100)
			reissue.Proofs = proofs
			add(fmt.Sprintf("ReissueWithProofs/v%d/%s", v, pn), reissue)
			burn := NewUnsignedBurnWithProofs(v, pk, asset, 1000, ts, 100)
			burn.Proofs = proofs
			add(fmt.Sprintf("BurnWithProofs/v%d/%s", v, pn), burn)
			lc := NewUnsignedLeaseCancelWithProofs(v, pk, asset, 100, ts)
			lc.Proofs = proofs
			add(fmt.Sprintf("LeaseCancelWithProofs/v%d/%s", v, pn), lc)
			ca := NewUnsignedCreateAliasWithProofs(v, pk, *alias, 100, ts)
			ca.Proofs = proofs
			add(fmt.Sprintf("CreateAliasWithProofs/v%d/%s", v, pn), ca)
		}
	}

	for _, reissuable := range []bool{false, true} {
		issue := NewUnsignedIssueWithSig(pk, "name", "", 1000, 2, reissuable, ts, 100)
		issue.Signature = &sig
		add(fmt.Sprintf("IssueWithSig/%t", reissuable), issue)
		reissue := NewUnsignedReissueWithSig(pk, asset, 1000, reissuable, ts, 100)
		reissue.Signature = &sig
		add(fmt.Sprintf("ReissueWithSig/%t", reissuable), reissue)
	}
	for rn, rcp := range simpleTxRecipients(t) {
		for an, aa := range assets {
			for atn, att := range attachments {
				tx := NewUnsignedTransferWithSig(pk, aa, aa, ts, 12345, 100, rcp, att)
				tx.Signature = &sig
				add(fmt.Sprintf("TransferWithSig/%s/%s/%s", rn, an, atn), tx)
			}
		}
		lease := NewUnsignedLeaseWithSig(pk, rcp, 1000, 100, ts)
		lease.Signature = &sig
		add("LeaseWithSig/"+rn, lease)
	}
	burn := NewUnsignedBurnWithSig(pk, asset, 1000, ts, 100)
	burn.Signature = &sig
	add("BurnWithSig", burn)
	lc := NewUnsignedLeaseCancelWithSig(pk, asset, 100, ts)
	lc.Signature = &sig
	add("LeaseCancelWithSig", lc)
	ca := NewUnsignedCreateAliasWithSig(pk, *alias, 100, ts)
	ca.Signature = &sig
	add("CreateAliasWithSig", ca)

	// Real signed transactions.
	signed := []Transaction{
		NewUnsignedTransferWithProofs(2, pk, assets["asset"], assets["waves"], ts, 1, 1, simpleTxRecipients(t)["addr"],
			attachments["att"]),
		NewUnsignedTransferWithSig(pk, assets["waves"], assets["waves"], ts, 1, 1, simpleTxRecipients(t)["alias"], nil),
		NewUnsignedIssueWithProofs(2, pk, "signed", "", 1, 0, false, scripts["script"], ts, 1),
		NewUnsignedCreateAliasWithSig(pk, *alias, 1, ts),
	}
	for i, tx := range signed {
		require.NoError(t, tx.Sign(benchScheme, kp.Secret))
		add(fmt.Sprintf("Signed/%d/%T", i, tx), tx)
	}
	return cases
}

func TestSimpleTxMarshalBinaryGolden(t *testing.T) {
	for _, c := range simpleTxCases(t) {
		t.Run(c.name, func(t *testing.T) {
			expectedBody, expectedFull, err := oldSimpleTxMarshal(c.tx, benchScheme)
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

			bodyFromHelper, err := MarshalTxBody(benchScheme, c.tx)
			require.NoError(t, err)
			if !IsProtobufTx(c.tx) {
				assert.Equal(t, expectedBody, bodyFromHelper)
			}
			// Version 1 of transactions with proofs exists only as a test vector for marshalling.
			if !IsProtobufTx(c.tx) && !strings.Contains(c.name, "WithProofs/v1/") {
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

func TestSimpleTxBodyHelpersGolden(t *testing.T) {
	for _, c := range simpleTxCases(t) {
		var got, expected []byte
		var err error
		switch tx := c.tx.(type) {
		case *TransferWithProofs:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected, err = oldTransferMarshalBinary(&tx.Transfer)
			require.NoError(t, err)
			assert.Equal(t, tx.Transfer.BinarySize(), len(got), c.name)
		case *LeaseWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected, err = oldLeaseMarshalBinary(&tx.Lease)
			require.NoError(t, err)
			assert.Equal(t, tx.Lease.BinarySize(), len(got), c.name)
		case *IssueWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected = oldIssueMarshalBinary(&tx.Issue)
			assert.Equal(t, tx.Issue.BinarySize(), len(got), c.name)
		case *ReissueWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected = oldReissueMarshalBinary(&tx.Reissue)
			assert.Equal(t, tx.Reissue.BinarySize(), len(got), c.name)
		case *BurnWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected = oldBurnMarshalBinary(&tx.Burn)
			assert.Equal(t, tx.Burn.BinarySize(), len(got), c.name)
		case *LeaseCancelWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected = oldLeaseCancelMarshalBinary(&tx.LeaseCancel)
			assert.Equal(t, tx.LeaseCancel.BinarySize(), len(got), c.name)
		case *CreateAliasWithSig:
			got, err = tx.marshalBinary()
			require.NoError(t, err)
			expected = oldCreateAliasMarshalBinary(&tx.CreateAlias)
			assert.Equal(t, tx.CreateAlias.BinarySize(), len(got), c.name)
		default:
			continue
		}
		assert.Equal(t, expected, got, c.name)
	}
}

func TestProofsAppendBinaryGolden(t *testing.T) {
	for name, p := range simpleTxProofs() {
		got, err := p.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, oldProofsMarshalBinary(p), got, name)
		assert.Equal(t, p.BinarySize(), len(got), name)
		prefix := []byte{0xAA}
		assert.Equal(t, append([]byte{0xAA}, got...), p.appendBinary(prefix), name)
	}
}

func TestSimpleTxMarshalBinaryErrors(t *testing.T) {
	kp := benchKeyPair(t, "simple-sender")
	ts := uint64(benchTimestamp)
	waves := NewOptionalAssetWaves()
	rcp := benchRecipient(t, "simple-recipient")

	t.Run("no proofs", func(t *testing.T) {
		txs := []Transaction{
			NewUnsignedIssueWithProofs(2, kp.Public, "name", "", 1, 0, false, nil, ts, 1),
			NewUnsignedTransferWithProofs(2, kp.Public, waves, waves, ts, 1, 1, rcp, nil),
			NewUnsignedReissueWithProofs(2, kp.Public, benchDigest(1), 1, false, ts, 1),
			NewUnsignedBurnWithProofs(2, kp.Public, benchDigest(1), 1, ts, 1),
			NewUnsignedLeaseWithProofs(2, kp.Public, rcp, 1, 1, ts),
			NewUnsignedLeaseCancelWithProofs(2, kp.Public, benchDigest(1), 1, ts),
			NewUnsignedCreateAliasWithProofs(2, kp.Public, *NewAlias(benchScheme, "alias"), 1, ts),
		}
		for _, tx := range txs {
			_, err := tx.MarshalBinary(benchScheme)
			require.ErrorContains(t, err, "no proofs", "%T", tx)
		}
	})
	t.Run("no signature", func(t *testing.T) {
		txs := []Transaction{
			NewUnsignedIssueWithSig(kp.Public, "name", "", 1, 0, false, ts, 1),
			NewUnsignedTransferWithSig(kp.Public, waves, waves, ts, 1, 1, rcp, nil),
			NewUnsignedReissueWithSig(kp.Public, benchDigest(1), 1, false, ts, 1),
			NewUnsignedBurnWithSig(kp.Public, benchDigest(1), 1, ts, 1),
			NewUnsignedLeaseWithSig(kp.Public, rcp, 1, 1, ts),
			NewUnsignedLeaseCancelWithSig(kp.Public, benchDigest(1), 1, ts),
			NewUnsignedCreateAliasWithSig(kp.Public, *NewAlias(benchScheme, "alias"), 1, ts),
		}
		for _, tx := range txs {
			_, err := tx.MarshalBinary(benchScheme)
			require.ErrorContains(t, err, "no signature", "%T", tx)
		}
	})
	t.Run("nil recipient", func(t *testing.T) {
		txs := []Transaction{
			NewUnsignedTransferWithProofs(2, kp.Public, waves, waves, ts, 1, 1, Recipient{}, nil),
			NewUnsignedLeaseWithProofs(2, kp.Public, Recipient{}, 1, 1, ts),
		}
		for _, tx := range txs {
			_, err := tx.BodyMarshalBinary(benchScheme)
			require.ErrorContains(t, err, "recipient is <nil>", "%T", tx)
		}
	})
	t.Run("attachment too long", func(t *testing.T) {
		tx := NewUnsignedTransferWithProofs(2, kp.Public, waves, waves, ts, 1, 1, rcp, make([]byte, 1<<15))
		_, err := tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "invalid data length")
		_, oldErr := oldTransferMarshalBinary(&tx.Transfer)
		require.Error(t, oldErr)
	})
	t.Run("script too long", func(t *testing.T) {
		tx := NewUnsignedIssueWithProofs(2, kp.Public, "name", "", 1, 0, false, make([]byte, 1<<15), ts, 1)
		_, err := tx.BodyMarshalBinary(benchScheme)
		require.ErrorContains(t, err, "invalid data length")
		_, oldErr := oldIssueWithProofsBody(tx, benchScheme)
		require.Error(t, oldErr)
	})
}

func BenchmarkSimpleTxMarshalBinary(b *testing.B) {
	kp := benchKeyPair(b, "simple-sender")
	ts := uint64(benchTimestamp)
	waves := NewOptionalAssetWaves()
	rcp := benchRecipient(b, "simple-recipient")
	txs := []Transaction{
		NewUnsignedTransferWithProofs(2, kp.Public, waves, waves, ts, 1, 1, rcp, []byte("attachment")),
		NewUnsignedTransferWithSig(kp.Public, waves, waves, ts, 1, 1, rcp, []byte("attachment")),
		NewUnsignedIssueWithProofs(2, kp.Public, "name", "description", 1, 0, false, nil, ts, 1),
		NewUnsignedLeaseWithProofs(2, kp.Public, rcp, 1, 1, ts),
		NewUnsignedBurnWithSig(kp.Public, benchDigest(1), 1, ts, 1),
	}
	for _, tx := range txs {
		require.NoError(b, tx.Sign(benchScheme, kp.Secret))
		name := fmt.Sprintf("%T", tx)[len("*proto."):]
		b.Run(name+"/old", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := oldSimpleTxMarshal(tx, benchScheme); err != nil {
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
