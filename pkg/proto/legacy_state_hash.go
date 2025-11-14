package proto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	legacyStateHashFieldsCountV1 = 9
	legacyStateHashFieldsCountV2 = 10
)

// FieldsHashesV1 is set of hashes fields for the legacy StateHashV1.
type FieldsHashesV1 struct {
	WavesBalanceHash  crypto.Digest
	AssetBalanceHash  crypto.Digest
	DataEntryHash     crypto.Digest
	AccountScriptHash crypto.Digest
	AssetScriptHash   crypto.Digest
	LeaseBalanceHash  crypto.Digest
	LeaseStatusHash   crypto.Digest
	SponsorshipHash   crypto.Digest
	AliasesHash       crypto.Digest
}

func (s *FieldsHashesV1) Equal(other FieldsHashesV1) bool {
	return s.WavesBalanceHash == other.WavesBalanceHash && s.AssetBalanceHash == other.AssetBalanceHash &&
		s.DataEntryHash == other.DataEntryHash && s.AccountScriptHash == other.AccountScriptHash &&
		s.AssetScriptHash == other.AssetScriptHash && s.LeaseBalanceHash == other.LeaseBalanceHash &&
		s.LeaseStatusHash == other.LeaseStatusHash && s.SponsorshipHash == other.SponsorshipHash &&
		s.AliasesHash == other.AliasesHash
}

func (s FieldsHashesV1) MarshalJSON() ([]byte, error) {
	return json.Marshal(fieldsHashesJSV1{
		WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
		AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
		DataEntryHash:     DigestWrapped(s.DataEntryHash),
		AccountScriptHash: DigestWrapped(s.AccountScriptHash),
		AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
		LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
		LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
		SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
		AliasesHash:       DigestWrapped(s.AliasesHash),
	})
}

func (s *FieldsHashesV1) UnmarshalJSON(value []byte) error {
	var sh fieldsHashesJSV1
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	return nil
}

//nolint:dupl // in sake of performance we allow code duplication here.
func (s *FieldsHashesV1) WriteTo(w io.Writer) (int64, error) {
	var (
		n   int
		cnt int64
		err error
	)
	if n, err = w.Write(s.WavesBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.AssetBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.DataEntryHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.AccountScriptHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.AssetScriptHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.LeaseBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.LeaseStatusHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = w.Write(s.SponsorshipHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	n, err = w.Write(s.AliasesHash[:])
	return cnt + int64(n), err
}

//nolint:dupl // in sake of performance we allow code duplication here.
func (s *FieldsHashesV1) ReadFrom(r io.Reader) (int64, error) {
	var (
		n   int
		cnt int64
		err error
	)
	if n, err = r.Read(s.WavesBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.AssetBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.DataEntryHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.AccountScriptHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.AssetScriptHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.LeaseBalanceHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.LeaseStatusHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	if n, err = r.Read(s.SponsorshipHash[:]); err != nil {
		return cnt + int64(n), err
	}
	cnt += int64(n)
	n, err = r.Read(s.AliasesHash[:])
	return cnt + int64(n), err
}

// FieldsHashesV2 is set of hashes fields for the legacy StateHashV2.
// It's a FieldsHashesV1 with an additional GeneratorsHash field.
type FieldsHashesV2 struct {
	FieldsHashesV1
	GeneratorsHash crypto.Digest
}

func (s *FieldsHashesV2) Equal(other FieldsHashesV2) bool {
	return s.FieldsHashesV1.Equal(other.FieldsHashesV1) && s.GeneratorsHash == other.GeneratorsHash
}

func (s FieldsHashesV2) MarshalJSON() ([]byte, error) {
	return json.Marshal(fieldsHashesJSV2{
		fieldsHashesJSV1: fieldsHashesJSV1{
			WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
			AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
			DataEntryHash:     DigestWrapped(s.DataEntryHash),
			AccountScriptHash: DigestWrapped(s.AccountScriptHash),
			AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
			LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
			LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
			SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
			AliasesHash:       DigestWrapped(s.AliasesHash),
		},
		GeneratorsHash: DigestWrapped(s.GeneratorsHash),
	})
}

func (s *FieldsHashesV2) UnmarshalJSON(value []byte) error {
	var sh fieldsHashesJSV2
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	s.GeneratorsHash = crypto.Digest(sh.GeneratorsHash)
	return nil
}

func (s *FieldsHashesV2) WriteTo(w io.Writer) (int64, error) {
	n, err := s.FieldsHashesV1.WriteTo(w)
	if err != nil {
		return n, err
	}
	m, err := w.Write(s.GeneratorsHash[:])
	return n + int64(m), err
}

// StateHashV1 is the legacy state hash structure used prior the activation of Deterministic Finality feature.
type StateHashV1 struct {
	BlockID BlockID
	SumHash crypto.Digest
	FieldsHashesV1
}

func (s *StateHashV1) GenerateSumHash(prevSumHash []byte) error {
	h, err := crypto.NewFastHash()
	if err != nil {
		return err
	}
	if _, wErr := h.Write(prevSumHash); wErr != nil {
		return wErr
	}
	if _, wErr := s.WriteTo(h); wErr != nil {
		return wErr
	}
	h.Sum(s.SumHash[:0])
	return nil
}

func (s *StateHashV1) MarshalBinary() []byte {
	res := make([]byte, 0, 1+s.BlockID.Len()+crypto.DigestSize*(legacyStateHashFieldsCountV1+1))
	buf := bytes.NewBuffer(res)
	if _, err := SizedBlockID(s.BlockID).WriteTo(buf); err != nil {
		// TODO: replace panic with error handling.
		panic(err)
	}
	buf.Write(s.SumHash[:])
	if _, err := s.WriteTo(buf); err != nil {
		// TODO: replace panic with error handling.
		panic(err)
	}
	return buf.Bytes()
}

func (s *StateHashV1) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	sid := SizedBlockID{}
	if _, rErr := sid.ReadFrom(r); rErr != nil {
		return rErr
	}
	s.BlockID = BlockID(sid)
	if _, rErr := r.Read(s.SumHash[:]); rErr != nil {
		return rErr
	}
	if _, rErr := s.ReadFrom(r); rErr != nil {
		return rErr
	}
	return nil
}

func (s StateHashV1) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toStateHashJS())
}

func (s *StateHashV1) UnmarshalJSON(value []byte) error {
	var sh stateHashJSV1
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.BlockID = sh.BlockID
	s.SumHash = crypto.Digest(sh.SumHash)
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	return nil
}

func (s *StateHashV1) toStateHashJS() stateHashJSV1 {
	return stateHashJSV1{
		BlockID: s.BlockID,
		SumHash: DigestWrapped(s.SumHash),
		fieldsHashesJSV1: fieldsHashesJSV1{
			WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
			AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
			DataEntryHash:     DigestWrapped(s.DataEntryHash),
			AccountScriptHash: DigestWrapped(s.AccountScriptHash),
			AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
			LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
			LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
			SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
			AliasesHash:       DigestWrapped(s.AliasesHash),
		},
	}
}

// StateHashV2 is the legacy state hash structure used after the activation of Deterministic Finality feature.
type StateHashV2 struct {
	BlockID BlockID
	SumHash crypto.Digest
	FieldsHashesV2
}

func (s *StateHashV2) GenerateSumHash(prevSumHash []byte) error {
	h, err := crypto.NewFastHash()
	if err != nil {
		return err
	}
	if _, wErr := h.Write(prevSumHash); wErr != nil {
		return wErr
	}
	if _, wErr := s.WriteTo(h); wErr != nil {
		return wErr
	}
	h.Sum(s.SumHash[:0])
	return nil
}

func (s *StateHashV2) MarshalJSON() ([]byte, error) {
	return json.Marshal(stateHashJSV2{
		BlockID: s.BlockID,
		SumHash: DigestWrapped(s.SumHash),
		fieldsHashesJSV2: fieldsHashesJSV2{
			fieldsHashesJSV1: fieldsHashesJSV1{
				WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
				AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
				DataEntryHash:     DigestWrapped(s.DataEntryHash),
				AccountScriptHash: DigestWrapped(s.AccountScriptHash),
				AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
				LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
				LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
				SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
				AliasesHash:       DigestWrapped(s.AliasesHash),
			},
			GeneratorsHash: DigestWrapped(s.GeneratorsHash),
		},
	})
}

type StateHashDebugV1 struct {
	stateHashJSV1
	Height       uint64        `json:"height,omitempty"`
	Version      string        `json:"version,omitempty"`
	SnapshotHash crypto.Digest `json:"snapshotHash"`
}

func NewStateHashJSDebugV1(s StateHashV1, h uint64, v string, snapshotStateHash crypto.Digest) StateHashDebugV1 {
	return StateHashDebugV1{stateHashJSV1: s.toStateHashJS(), Height: h, Version: v, SnapshotHash: snapshotStateHash}
}

func (s StateHashDebugV1) GetStateHash() *StateHashV1 {
	sh := &StateHashV1{
		BlockID: s.BlockID,
		SumHash: crypto.Digest(s.SumHash),
		FieldsHashesV1: FieldsHashesV1{
			WavesBalanceHash:  crypto.Digest(s.WavesBalanceHash),
			AssetBalanceHash:  crypto.Digest(s.AssetBalanceHash),
			DataEntryHash:     crypto.Digest(s.DataEntryHash),
			AccountScriptHash: crypto.Digest(s.AccountScriptHash),
			AssetScriptHash:   crypto.Digest(s.AssetBalanceHash),
			LeaseBalanceHash:  crypto.Digest(s.LeaseBalanceHash),
			LeaseStatusHash:   crypto.Digest(s.LeaseStatusHash),
			SponsorshipHash:   crypto.Digest(s.SponsorshipHash),
			AliasesHash:       crypto.Digest(s.AliasesHash),
		},
	}
	return sh
}

// DigestWrapped is required for state hashes API.
// The quickest way to use Hex for hashes in JSON in this particular case.
type DigestWrapped crypto.Digest

func (d DigestWrapped) MarshalJSON() ([]byte, error) {
	s := hex.EncodeToString(d[:])
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (d *DigestWrapped) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(b) != crypto.DigestSize {
		return errors.New("bad size")
	}
	copy(d[:], b[:crypto.DigestSize])
	return nil
}

type fieldsHashesJSV1 struct {
	WavesBalanceHash  DigestWrapped `json:"wavesBalanceHash"`
	AssetBalanceHash  DigestWrapped `json:"assetBalanceHash"`
	DataEntryHash     DigestWrapped `json:"dataEntryHash"`
	AccountScriptHash DigestWrapped `json:"accountScriptHash"`
	AssetScriptHash   DigestWrapped `json:"assetScriptHash"`
	LeaseBalanceHash  DigestWrapped `json:"leaseBalanceHash"`
	LeaseStatusHash   DigestWrapped `json:"leaseStatusHash"`
	SponsorshipHash   DigestWrapped `json:"sponsorshipHash"`
	AliasesHash       DigestWrapped `json:"aliasHash"`
}

type fieldsHashesJSV2 struct {
	fieldsHashesJSV1
	GeneratorsHash DigestWrapped `json:"nextCommittedGeneratorsHash"`
}

type stateHashJSV1 struct {
	BlockID BlockID       `json:"blockId"`
	SumHash DigestWrapped `json:"stateHash"`
	fieldsHashesJSV1
}

type stateHashJSV2 struct {
	BlockID BlockID       `json:"blockId"`
	SumHash DigestWrapped `json:"stateHash"`
	fieldsHashesJSV2
}

type SizedBlockID BlockID

func (id SizedBlockID) WriteTo(w io.Writer) (int64, error) {
	oid := BlockID(id)
	l := oid.Len()
	if l == 0 {
		return 0, errors.New("invalid BlockID")
	}
	n, err := w.Write([]byte{byte(l)})
	if err != nil {
		return int64(n), err
	}
	m, err := oid.WriteTo(w)
	return int64(n) + m, err
}

func (id *SizedBlockID) ReadFrom(r io.Reader) (int64, error) {
	l := make([]byte, 1)
	n, err := r.Read(l)
	if err != nil {
		return int64(n), err
	}
	var oid BlockID
	switch l[0] {
	case crypto.DigestSize:
		oid = NewBlockIDFromDigest(crypto.Digest{})
	case crypto.SignatureSize:
		oid = NewBlockIDFromSignature(crypto.Signature{})
	default:
		return int64(n), errors.New("invalid BlockID size")
	}
	m, err := oid.ReadFrom(r)
	if err != nil {
		return int64(n) + m, err
	}
	*id = SizedBlockID(oid)
	return int64(n) + m, nil
}
