package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

// Old implementations kept as oracles for the single-buffer block, microblock, snapshot and message encoders.

func oldTransactionsWriteToBinary(t *testing.T, a Transactions, scheme Scheme) []byte {
	buf := new(bytes.Buffer)
	s := serializer.New(buf)
	for _, tx := range a {
		bts, err := tx.MarshalBinary(scheme)
		require.NoError(t, err)
		require.NoError(t, s.Uint32(uint32(len(bts)))) // #nosec G115 // Test data is small.
		require.NoError(t, s.Bytes(bts))
	}
	return buf.Bytes()
}

func oldMicroBlockWriteWithoutSignature(t *testing.T, a *MicroBlock, scheme Scheme) []byte {
	buf := new(bytes.Buffer)
	s := serializer.NewNonFallable(buf)
	s.Byte(byte(a.VersionField))
	s.Bytes(a.Reference.Bytes())
	s.Bytes(a.TotalResBlockSigField.Bytes())
	txsBuf := new(bytes.Buffer)
	txsSerializer := serializer.NewNonFallable(txsBuf)
	isProto := a.VersionField >= ProtobufBlockVersion
	_, err := a.Transactions.WriteTo(isProto, scheme, txsSerializer)
	require.NoError(t, err)
	s.Uint32(uint32(txsBuf.Len() + 4)) // #nosec G115 // Test data is small.
	s.Uint32(a.TransactionCount)
	s.Bytes(txsBuf.Bytes())
	s.Bytes(a.SenderPK.Bytes())
	var stateHash []byte
	if sh, present := a.GetStateHash(); present {
		stateHash = sh.Bytes()
	}
	s.Bytes(stateHash)
	if isProto && a.PartialFinalization != nil {
		fp, fErr := a.PartialFinalization.ToProtobuf()
		require.NoError(t, fErr)
		fb, mErr := fp.MarshalVTStrict()
		require.NoError(t, mErr)
		s.Bytes(fb)
	}
	return buf.Bytes()
}

func oldBlockSnapshotMarshallBinary(t *testing.T, bs BlockSnapshot) []byte {
	result := binary.BigEndian.AppendUint32([]byte{}, uint32(len(bs.TxSnapshots))) // #nosec G115
	for _, ts := range bs.TxSnapshots {
		var res g.TransactionStateSnapshot
		for _, atomicSnapshot := range ts {
			require.NoError(t, atomicSnapshot.AppendToProtobuf(&res))
		}
		tsBytes, err := res.MarshalVTStrict()
		require.NoError(t, err)
		result = binary.BigEndian.AppendUint32(result, uint32(len(tsBytes))) // #nosec G115
		result = append(result, tsBytes...)
	}
	return result
}

func oldMarshalMessage(t *testing.T, contentID PeerMessageID, body []byte) []byte {
	h, err := NewHeader(contentID, body)
	require.NoError(t, err)
	hdr, err := h.MarshalBinary()
	require.NoError(t, err)
	return append(hdr, body...)
}

func oldWriteMessage(t *testing.T, contentID PeerMessageID, payload io.WriterTo) []byte {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := payload.WriteTo(buf)
	require.NoError(t, err)
	h, err := NewHeader(contentID, buf.Bytes())
	require.NoError(t, err)
	out := new(bytes.Buffer)
	_, err = h.WriteTo(out)
	require.NoError(t, err)
	_, err = buf.WriteTo(out)
	require.NoError(t, err)
	return out.Bytes()
}

// legacyTxCases returns all binary (non-protobuf) transaction cases, plus Genesis and Payment.
func legacyTxCases(t *testing.T) []simpleTxCase {
	var res []simpleTxCase
	for _, c := range append(simpleTxCases(t), complexTxCases(t)...) {
		if IsProtobufTx(c.tx) {
			continue
		}
		// Keep only the transactions that a block can carry, i.e. those that BytesToTransaction accepts back.
		b, err := c.tx.MarshalBinary(benchScheme)
		require.NoError(t, err)
		if _, err = BytesToTransaction(b, benchScheme); err != nil {
			continue
		}
		res = append(res, c)
	}
	kp := benchKeyPair(t, "legacy sender")
	rcp := benchRecipient(t, "legacy recipient")
	gen := NewUnsignedGenesis(*rcp.Address(), 1000, benchTimestamp)
	require.NoError(t, gen.Sign(benchScheme, crypto.SecretKey{}))
	pay := NewUnsignedPayment(kp.Public, *rcp.Address(), 1000, 100_000, benchTimestamp)
	require.NoError(t, pay.Sign(benchScheme, kp.Secret))
	return append(res, simpleTxCase{"Genesis", gen}, simpleTxCase{"Payment", pay})
}

func pbTxsWithFallback(t *testing.T) Transactions {
	fx := pbTxFixtures(t)
	txs := make(Transactions, 0, len(fx)+1)
	for _, f := range fx {
		txs = append(txs, f.tx)
	}
	// Genesis has no non-copying protobuf converter, so it exercises the MarshalSignedToProtobuf fallback.
	rcp := benchRecipient(t, "legacy recipient")
	gen := NewUnsignedGenesis(*rcp.Address(), 1000, benchTimestamp)
	require.NoError(t, gen.Sign(benchScheme, crypto.SecretKey{}))
	_, ok := Transaction(gen).(signedTxNoCopyConverter)
	require.False(t, ok)
	return append(txs, gen)
}

func TestAppendTxBinaryGolden(t *testing.T) {
	for _, c := range legacyTxCases(t) {
		t.Run(c.name, func(t *testing.T) {
			expected, err := c.tx.MarshalBinary(benchScheme)
			require.NoError(t, err)
			assert.Len(t, expected, c.tx.BinarySize())
			ap, ok := c.tx.(txBinaryAppender)
			if !ok {
				return
			}
			prefix := []byte{0xde, 0xad}
			dst := make([]byte, len(prefix), len(prefix)+c.tx.BinarySize())
			copy(dst, prefix)
			res, err := ap.appendTxBinary(dst, benchScheme)
			require.NoError(t, err)
			assert.Equal(t, append(prefix, expected...), res)
			assert.Equal(t, cap(dst), cap(res), "BinarySize capacity must be enough")
		})
	}
}

func TestTransactionsBinaryGolden(t *testing.T) {
	cases := legacyTxCases(t)
	txs := make(Transactions, len(cases))
	for i, c := range cases {
		txs[i] = c.tx
	}
	expected := oldTransactionsWriteToBinary(t, txs, benchScheme)
	assert.Len(t, expected, txs.BinarySize())

	mb, err := txs.MarshalBinary(benchScheme)
	require.NoError(t, err)
	assert.Equal(t, expected, mb)
	assert.Equal(t, len(mb), cap(mb), "MarshalBinary must be sized exactly")

	buf := new(bytes.Buffer)
	n, err := txs.WriteToBinary(buf, benchScheme)
	require.NoError(t, err)
	assert.Equal(t, int64(len(expected)), n)
	assert.Equal(t, expected, buf.Bytes())

	ws := new(bytes.Buffer)
	n, err = txs.WriteTo(false, benchScheme, ws)
	require.NoError(t, err)
	assert.Equal(t, int64(len(expected)), n)
	assert.Equal(t, expected, ws.Bytes())

	empty, err := Transactions(nil).MarshalBinary(benchScheme)
	require.NoError(t, err)
	assert.Empty(t, empty)

	restored, err := NewTransactionsFromBytes(mb, len(txs), benchScheme)
	require.NoError(t, err)
	again, err := restored.MarshalBinary(benchScheme)
	require.NoError(t, err)
	assert.Equal(t, expected, again)
}

func TestTransactionsBinaryError(t *testing.T) {
	tx := NewUnsignedTransferWithSig(benchKeyPair(t, "x").Public, NewOptionalAssetWaves(),
		NewOptionalAssetWaves(), benchTimestamp, 1, 1, benchRecipient(t, "y"), nil) // No signature.
	_, err := Transactions{tx}.MarshalBinary(benchScheme)
	require.Error(t, err)
	_, err = Transactions{tx}.WriteToBinary(new(bytes.Buffer), benchScheme)
	require.Error(t, err)
}

func TestSignedProtobufsGolden(t *testing.T) {
	txs := pbTxsWithFallback(t)
	expected := new(bytes.Buffer)
	_, err := txs.WriteTo(true, benchScheme, expected)
	require.NoError(t, err)
	pbs, raw, size, err := txs.signedProtobufs(benchScheme)
	require.NoError(t, err)
	require.Equal(t, expected.Len(), size)
	buf := make([]byte, size)
	require.NoError(t, putSignedProtobufs(buf, pbs, raw))
	assert.Equal(t, expected.Bytes(), buf)
	require.Error(t, putSignedProtobufs(make([]byte, size+1), pbs, raw), "size mismatch must be detected")
}

func legacyBlocks(t *testing.T) []*Block {
	kp := benchKeyPair(t, "benchmark generator")
	cases := legacyTxCases(t)
	txs := make(Transactions, len(cases))
	for i, c := range cases {
		txs[i] = c.tx
	}
	return []*Block{
		benchLegacyBlock(t, 10),
		benchCreateBlock(t, RewardBlockVersion, txs, kp),
		benchCreateBlock(t, NgBlockVersion, txs, kp),
		benchCreateBlock(t, NgBlockVersion, Transactions{}, kp),
	}
}

func TestLegacyBlockMarshalBinaryGolden(t *testing.T) {
	for _, b := range legacyBlocks(t) {
		data, err := b.MarshalBinary(benchScheme)
		require.NoError(t, err)
		// The old encoding, with the transactions written by the oracle.
		buf := new(bytes.Buffer)
		s := serializer.NewNonFallable(buf)
		s.Byte(byte(b.Version))
		s.Uint64(b.Timestamp)
		s.Bytes(b.Parent.Bytes())
		s.Uint32(b.ConsensusBlockLength)
		s.Uint64(b.BaseTarget)
		s.Bytes(b.GenSignature[:])
		s.Uint32(b.TransactionBlockLength)
		s.Uint32(uint32(b.TransactionCount)) // #nosec G115 // Test data is small.
		s.Bytes(oldTransactionsWriteToBinary(t, b.Transactions, benchScheme))
		s.Uint32(uint32(b.FeaturesCount)) // #nosec G115 // Test data is small.
		fb, err := featuresToBinary(b.Features)
		require.NoError(t, err)
		s.Bytes(fb)
		if b.Version >= RewardBlockVersion {
			s.Int64(b.RewardVote)
		}
		s.Bytes(b.GeneratorPublicKey[:])
		s.Bytes(b.BlockSignature[:])
		assert.Equal(t, buf.Bytes(), data)

		var restored Block
		require.NoError(t, restored.UnmarshalBinary(data, benchScheme))
		again, err := restored.MarshalBinary(benchScheme)
		require.NoError(t, err)
		assert.Equal(t, data, again)
		ok, err := restored.VerifySignature(benchScheme)
		require.NoError(t, err)
		assert.True(t, ok)
	}
}

func TestProtobufBlockRoundTrip(t *testing.T) {
	kp := benchKeyPair(t, "benchmark generator")
	b := benchCreateBlock(t, ProtobufBlockVersion, pbTxsWithFallback(t), kp)
	data, err := b.MarshalToProtobuf(benchScheme)
	require.NoError(t, err)
	var restored Block
	require.NoError(t, restored.UnmarshalFromProtobuf(data))
	again, err := restored.MarshalToProtobuf(benchScheme)
	require.NoError(t, err)
	assert.Equal(t, data, again)
	ok, err := restored.VerifyTransactionsRoot(benchScheme)
	require.NoError(t, err)
	assert.True(t, ok)
}

func testMicroBlocks(t *testing.T) []*MicroBlock {
	kp := benchKeyPair(t, "benchmark generator")
	legacyRef := benchLegacyBlock(t, 1)
	pbRef := benchProtobufBlock(t, 1)
	cases := legacyTxCases(t)
	legacyTxs := make(Transactions, len(cases))
	for i, c := range cases {
		legacyTxs[i] = c.tx
	}
	sh := benchDigest(42)
	withSH := benchMicroBlock(t, pbRef, 3)
	withSH.StateHash = &sh
	withPF := benchMicroBlock(t, pbRef, 2)
	withPF.StateHash = &sh
	withPF.PartialFinalization = &FinalizationVoting{
		EndorserIndexes:                []uint32{1, 2, 5},
		FinalizedBlockHeight:           100,
		AggregatedEndorsementSignature: &bls.Signature{1, 2, 3},
	}
	pbMixed := benchMicroBlock(t, pbRef, 0)
	pbMixed.Transactions = pbTxsWithFallback(t)
	pbMixed.TransactionCount = uint32(len(pbMixed.Transactions)) // #nosec G115 // Test data is small.
	legacy := benchMicroBlock(t, legacyRef, 0)
	legacy.Transactions = legacyTxs
	legacy.TransactionCount = uint32(len(legacyTxs)) // #nosec G115 // Test data is small.
	legacyPF := benchMicroBlock(t, legacyRef, 1)
	legacyPF.PartialFinalization = withPF.PartialFinalization // Ignored for legacy microblocks.
	res := []*MicroBlock{
		benchMicroBlock(t, legacyRef, 10),
		benchMicroBlock(t, pbRef, 10),
		benchMicroBlock(t, pbRef, 0),
		withSH, withPF, pbMixed, legacy, legacyPF,
	}
	for _, mb := range res {
		require.NoError(t, mb.Sign(benchScheme, kp.Secret))
	}
	return res
}

func TestMicroBlockMarshalGolden(t *testing.T) {
	for i, mb := range testMicroBlocks(t) {
		expected := oldMicroBlockWriteWithoutSignature(t, mb, benchScheme)

		buf := new(bytes.Buffer)
		n, err := mb.WriteWithoutSignature(benchScheme, buf)
		require.NoError(t, err)
		assert.Equal(t, int64(len(expected)), n, "case %d", i)
		assert.Equal(t, expected, buf.Bytes(), "case %d", i)

		data, err := mb.MarshalBinary(benchScheme)
		require.NoError(t, err)
		full := append(bytes.Clone(expected), mb.Signature[:]...)
		assert.Equal(t, full, data, "case %d", i)
		assert.Equal(t, len(data), cap(data), "case %d: MarshalBinary must be sized exactly", i)

		wt := new(bytes.Buffer)
		n, err = mb.WriteTo(benchScheme, wt)
		require.NoError(t, err)
		assert.Equal(t, int64(len(full)), n)
		assert.Equal(t, full, wt.Bytes())

		ok, err := mb.VerifySignature(benchScheme)
		require.NoError(t, err)
		assert.True(t, ok, "case %d", i)
		assert.True(t, crypto.Verify(mb.SenderPK, mb.Signature, expected), "case %d", i)

		if mb.VersionField < ProtobufBlockVersion {
			var restored MicroBlock
			require.NoError(t, restored.UnmarshalBinary(data, benchScheme), "case %d", i)
			again, mErr := restored.MarshalBinary(benchScheme)
			require.NoError(t, mErr)
			assert.Equal(t, data, again, "case %d", i)
		}
	}
}

func TestMicroBlockMarshalError(t *testing.T) {
	mb := benchMicroBlock(t, benchLegacyBlock(t, 1), 0)
	tx := NewUnsignedTransferWithSig(benchKeyPair(t, "x").Public, NewOptionalAssetWaves(),
		NewOptionalAssetWaves(), benchTimestamp, 1, 1, benchRecipient(t, "y"), nil) // No signature.
	mb.Transactions = Transactions{tx}
	_, err := mb.MarshalBinary(benchScheme)
	require.Error(t, err)
	_, err = mb.WriteTo(benchScheme, new(bytes.Buffer))
	require.Error(t, err)
	_, err = mb.VerifySignature(benchScheme)
	require.Error(t, err)
}

func TestBlockSnapshotMarshalGolden(t *testing.T) {
	for _, n := range []int{0, 1, 17} {
		bs := benchBlockSnapshot(t, n)
		if n == 1 {
			bs.AppendTxSnapshot(nil) // An empty transaction snapshot.
		}
		expected := oldBlockSnapshotMarshallBinary(t, bs)
		data, err := bs.MarshallBinary()
		require.NoError(t, err)
		assert.Equal(t, expected, data)
		assert.Equal(t, len(data), cap(data))

		var restored BlockSnapshot
		require.NoError(t, restored.UnmarshalBinary(data, benchScheme))
		if n == 0 {
			assert.Nil(t, restored.TxSnapshots)
		} else {
			assert.Len(t, restored.TxSnapshots, len(bs.TxSnapshots))
		}
		again, err := restored.MarshallBinary()
		require.NoError(t, err)
		assert.Equal(t, data, again)

		var imported BlockSnapshot
		require.NoError(t, imported.UnmarshalBinaryImport(data[uint32Size:], benchScheme))
		assert.Equal(t, restored, imported)
	}
}

func TestBlockSnapshotUnmarshalBogusCount(t *testing.T) {
	data := binary.BigEndian.AppendUint32(nil, 0xffffffff)
	var bs BlockSnapshot
	require.Error(t, bs.UnmarshalBinary(data, benchScheme))
	require.Error(t, bs.UnmarshalBinary(append(data, 0, 0, 0, 5, 1), benchScheme))
}

func testMessagePayloads() map[string][]byte {
	return map[string][]byte{
		"empty": nil,
		"small": {1, 2, 3},
		"large": bytes.Repeat([]byte{0xab}, 100_000),
	}
}

func TestMarshalMessageGolden(t *testing.T) {
	type msgCase struct {
		id  PeerMessageID
		msg func(p []byte) Message
	}
	cases := map[string]msgCase{
		"Transaction":   {ContentIDTransaction, func(p []byte) Message { return &TransactionMessage{Transaction: p} }},
		"PBTransaction": {ContentIDPBTransaction, func(p []byte) Message { return &PBTransactionMessage{Transaction: p} }},
		"Block":         {ContentIDBlock, func(p []byte) Message { return &BlockMessage{BlockBytes: p} }},
		"PBBlock":       {ContentIDPBBlock, func(p []byte) Message { return &PBBlockMessage{PBBlockBytes: p} }},
		"Score":         {ContentIDScore, func(p []byte) Message { return &ScoreMessage{Score: p} }},
		"MicroBlock":    {ContentIDMicroblock, func(p []byte) Message { return &MicroBlockMessage{Body: p} }},
		"PBMicroBlock":  {ContentIDPBMicroBlock, func(p []byte) Message { return &PBMicroBlockMessage{MicroBlockBytes: p} }},
		"BlockSnapshot": {ContentIDBlockSnapshot, func(p []byte) Message { return &BlockSnapshotMessage{Bytes: p} }},
		"MicroBlockSnapshot": {ContentIDMicroBlockSnapshot,
			func(p []byte) Message { return &MicroBlockSnapshotMessage{Bytes: p} }},
		"EndorseBlock": {ContentIDEndorseBlock, func(p []byte) Message { return &EndorseBlockMessage{Bytes: p} }},
	}
	for name, c := range cases {
		for pn, p := range testMessagePayloads() {
			t.Run(name+"/"+pn, func(t *testing.T) {
				m := c.msg(p)
				expected := oldMarshalMessage(t, c.id, p)
				data, err := m.MarshalBinary()
				require.NoError(t, err)
				assert.Equal(t, expected, data)
				assert.Equal(t, len(data), cap(data))

				buf := new(bytes.Buffer)
				n, err := m.WriteTo(buf)
				require.NoError(t, err)
				assert.Equal(t, int64(len(expected)), n)
				assert.Equal(t, expected, buf.Bytes())
			})
		}
	}
}

func TestMarshalMessageGoldenStructured(t *testing.T) {
	id := NewBlockIDFromDigest(benchDigest(3))
	var sig crypto.Signature
	copy(sig[:], "signature")
	sigID := NewBlockIDFromSignature(sig)
	msgs := []struct {
		id   PeerMessageID
		m    Message
		body []byte
	}{
		{ContentIDGetBlock, &GetBlockMessage{BlockID: id}, id.Bytes()},
		{ContentIDGetBlockSnapshot, &GetBlockSnapshotMessage{BlockID: id}, id.Bytes()},
		{ContentIDMicroBlockSnapshotRequest, &MicroBlockSnapshotRequestMessage{BlockID: id}, id.Bytes()},
		{ContentIDGetSignatures, &GetSignaturesMessage{Signatures: []crypto.Signature{sig, sig}},
			append(binary.BigEndian.AppendUint32(nil, 2), append(sig[:], sig[:]...)...)},
		{ContentIDSignatures, &SignaturesMessage{Signatures: []crypto.Signature{sig}},
			append(binary.BigEndian.AppendUint32(nil, 1), sig[:]...)},
		{ContentIDGetBlockIDs, &GetBlockIDsMessage{Blocks: []BlockID{id, sigID}},
			append(append(append(binary.BigEndian.AppendUint32(nil, 2), byte(len(id.Bytes()))), id.Bytes()...),
				append([]byte{byte(len(sigID.Bytes()))}, sigID.Bytes()...)...)},
		{ContentIDBlockIDs, &BlockIDsMessage{Blocks: []BlockID{id}},
			append(append(binary.BigEndian.AppendUint32(nil, 1), byte(len(id.Bytes()))), id.Bytes()...)},
	}
	for _, c := range msgs {
		data, err := c.m.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, oldMarshalMessage(t, c.id, c.body), data, "%T", c.m)
		buf := new(bytes.Buffer)
		_, err = c.m.WriteTo(buf)
		require.NoError(t, err)
		assert.Equal(t, data, buf.Bytes(), "%T", c.m)
	}
}

func TestWriteMessageGolden(t *testing.T) {
	for pn, p := range testMessagePayloads() {
		bp := BytesPayload(p)
		expected := oldWriteMessage(t, ContentIDBlock, &bp)
		buf := new(bytes.Buffer)
		n, err := WriteMessage(buf, ContentIDBlock, "test", &bp)
		require.NoError(t, err, pn)
		assert.Equal(t, int64(len(expected)), n, pn)
		assert.Equal(t, expected, buf.Bytes(), pn)

		// Non-BytesPayload payloads still go through the pooled buffer.
		u := U64(123456789)
		expected = oldWriteMessage(t, ContentIDScore, u)
		buf.Reset()
		_, err = WriteMessage(buf, ContentIDScore, "test", u)
		require.NoError(t, err)
		assert.Equal(t, expected, buf.Bytes())
	}
}

type failingWriter struct{ after int }

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.after <= 0 {
		return 0, io.ErrClosedPipe
	}
	w.after--
	return len(p), nil
}

func TestWriteMessageErrors(t *testing.T) {
	bp := BytesPayload{1, 2, 3}
	_, err := WriteMessage(&failingWriter{after: 0}, ContentIDBlock, "test", &bp)
	require.ErrorContains(t, err, "failed to write header")
	_, err = WriteMessage(&failingWriter{after: 1}, ContentIDBlock, "test", &bp)
	require.ErrorContains(t, err, "failed to write payload")
	m := &MicroBlockMessage{Body: bp}
	_, err = m.WriteTo(&failingWriter{after: 0})
	require.ErrorIs(t, err, io.ErrClosedPipe)
	_, err = m.WriteTo(&failingWriter{after: 1})
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestBytesPayloadReadFrom(t *testing.T) {
	for pn, p := range testMessagePayloads() {
		t.Run(pn, func(t *testing.T) {
			// Through ReadMessage the payload is pre-sized from the header.
			data := oldMarshalMessage(t, ContentIDBlock, p)
			var m BlockMessage
			n, err := m.ReadFrom(bytes.NewReader(data))
			require.NoError(t, err)
			assert.Equal(t, int64(len(data)), n)
			if len(p) == 0 {
				assert.Empty(t, m.BlockBytes)
			} else {
				assert.Equal(t, p, []byte(m.BlockBytes))
				assert.Equal(t, len(p), cap(m.BlockBytes), "pre-sized payload")
			}
			// Without a size hint the payload is read until EOF.
			var bp BytesPayload
			n, err = bp.ReadFrom(bytes.NewReader(p))
			require.NoError(t, err)
			assert.Equal(t, int64(len(p)), n)
			assert.Equal(t, len(p), len(bp))
		})
	}
	// A truncated payload is an error.
	data := oldMarshalMessage(t, ContentIDBlock, []byte{1, 2, 3, 4})
	var m BlockMessage
	_, err := m.ReadFrom(bytes.NewReader(data[:len(data)-1]))
	require.Error(t, err)
	// A corrupted payload fails the checksum.
	bad := bytes.Clone(data)
	bad[len(bad)-1] ^= 0xff
	_, err = m.ReadFrom(bytes.NewReader(bad))
	require.ErrorContains(t, err, "checksum")
}

func TestBytesPayloadReadFromHugeDeclaredLength(t *testing.T) {
	cr, err := NewChecksumReader(io.LimitReader(bytes.NewReader([]byte{1, 2, 3}), maxPresizedPayload+1))
	require.NoError(t, err)
	var bp BytesPayload
	n, err := bp.ReadFrom(cr)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
	assert.Equal(t, BytesPayload{1, 2, 3}, bp)
	assert.Less(t, cap(bp), maxPresizedPayload, "no allocation from the declared length")
}
