package proto

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
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
	DataEntryHash     crypto.Digest
	AccountScriptHash crypto.Digest
	AssetScriptHash   crypto.Digest
	LeaseStatusHash   crypto.Digest
	SponsorshipHash   crypto.Digest
	AliasesHash       crypto.Digest
	WavesBalanceHash  crypto.Digest
	AssetBalanceHash  crypto.Digest
	LeaseBalanceHash  crypto.Digest
}

func (s *FieldsHashesV1) Equal(other FieldsHashesV1) bool {
	return s.DataEntryHash == other.DataEntryHash && s.AccountScriptHash == other.AccountScriptHash &&
		s.AssetScriptHash == other.AssetScriptHash && s.LeaseStatusHash == other.LeaseStatusHash &&
		s.SponsorshipHash == other.SponsorshipHash && s.AliasesHash == other.AliasesHash &&
		s.WavesBalanceHash == other.WavesBalanceHash && s.AssetBalanceHash == other.AssetBalanceHash &&
		s.LeaseBalanceHash == other.LeaseBalanceHash
}

func (s FieldsHashesV1) MarshalJSON() ([]byte, error) {
	return json.Marshal(fieldsHashesJSV1{
		DataEntryHash:     DigestWrapped(s.DataEntryHash),
		AccountScriptHash: DigestWrapped(s.AccountScriptHash),
		AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
		LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
		SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
		AliasesHash:       DigestWrapped(s.AliasesHash),
		WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
		AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
		LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
	})
}

func (s *FieldsHashesV1) UnmarshalJSON(value []byte) error {
	var sh fieldsHashesJSV1
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
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

func (s *FieldsHashesV1) MarshalBinary() []byte {
	res := make([]byte, crypto.DigestSize*legacyStateHashFieldsCountV1)
	pos := 0
	copy(res[pos:pos+crypto.DigestSize], s.DataEntryHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AccountScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseStatusHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.SponsorshipHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AliasesHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.WavesBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseBalanceHash[:])
	return res
}

func (s *FieldsHashesV1) UnmarshalBinary(data []byte) (int, error) {
	expectedLen := crypto.DigestSize * legacyStateHashFieldsCountV1
	if l := len(data); l < expectedLen {
		return 0, fmt.Errorf("invalid data size %d less than expected %d bytes", l, expectedLen)
	}
	pos := 0
	copy(s.DataEntryHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AccountScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseStatusHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.SponsorshipHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AliasesHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.WavesBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseBalanceHash[:], data[pos:pos+crypto.DigestSize])
	return expectedLen, nil
}

func (s *FieldsHashesV1) HashFields(h hash.Hash) error {
	if _, err := h.Write(s.WavesBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AssetBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.DataEntryHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AccountScriptHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AssetScriptHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.LeaseBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.LeaseStatusHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.SponsorshipHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AliasesHash[:]); err != nil {
		return err
	}
	return nil
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
			DataEntryHash:     DigestWrapped(s.DataEntryHash),
			AccountScriptHash: DigestWrapped(s.AccountScriptHash),
			AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
			LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
			SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
			AliasesHash:       DigestWrapped(s.AliasesHash),
			WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
			AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
			LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
		},
		GeneratorsHash: DigestWrapped(s.GeneratorsHash),
	})
}

func (s *FieldsHashesV2) UnmarshalJSON(value []byte) error {
	var sh fieldsHashesJSV2
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	s.GeneratorsHash = crypto.Digest(sh.GeneratorsHash)
	return nil
}

func (s *FieldsHashesV2) HashFields(h hash.Hash) error {
	if err := s.FieldsHashesV1.HashFields(h); err != nil {
		return err
	}
	if _, wErr := h.Write(s.GeneratorsHash[:]); wErr != nil {
		return wErr
	}
	return nil
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
	if hErr := s.FieldsHashesV1.HashFields(h); hErr != nil {
		return hErr
	}
	h.Sum(s.SumHash[:0])
	return nil
}

func (s *StateHashV1) MarshalBinary() []byte {
	idBytes := s.BlockID.Bytes()
	res := make([]byte, 1+len(idBytes)+crypto.DigestSize*legacyStateHashFieldsCount)
	res[0] = byte(len(idBytes))
	pos := 1
	copy(res[pos:pos+len(idBytes)], idBytes)
	pos += len(idBytes)
	copy(res[pos:pos+crypto.DigestSize], s.SumHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.DataEntryHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AccountScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseStatusHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.SponsorshipHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AliasesHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.WavesBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.GeneratorsHash[:])
	return res
}

func (s *StateHashV1) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return errors.New("invalid data size")
	}
	idBytesLen := int(data[0])
	correctSize := 1 + idBytesLen + crypto.DigestSize*legacyStateHashFieldsCount
	if len(data) != correctSize {
		return errors.New("invalid data size")
	}
	var err error
	pos := 1
	s.BlockID, err = NewBlockIDFromBytes(data[pos : pos+idBytesLen])
	if err != nil {
		return err
	}
	pos += idBytesLen
	copy(s.SumHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.DataEntryHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AccountScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseStatusHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.SponsorshipHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AliasesHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.WavesBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseBalanceHash[:], data[pos:pos+crypto.DigestSize])
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
			DataEntryHash:     DigestWrapped(s.DataEntryHash),
			AccountScriptHash: DigestWrapped(s.AccountScriptHash),
			AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
			LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
			SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
			AliasesHash:       DigestWrapped(s.AliasesHash),
			WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
			AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
			LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
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
	s.FieldsHashesV2.HashFields(h)
	if _, wErr := h.Write(s.WavesBalanceHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.AssetBalanceHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.DataEntryHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.AccountScriptHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.AssetScriptHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.LeaseBalanceHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.LeaseStatusHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.SponsorshipHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.AliasesHash[:]); wErr != nil {
		return wErr
	}
	if _, wErr := h.Write(s.GeneratorsHash[:]); wErr != nil {
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
				DataEntryHash:     DigestWrapped(s.DataEntryHash),
				AccountScriptHash: DigestWrapped(s.AccountScriptHash),
				AssetScriptHash:   DigestWrapped(s.AssetScriptHash),
				LeaseStatusHash:   DigestWrapped(s.LeaseStatusHash),
				SponsorshipHash:   DigestWrapped(s.SponsorshipHash),
				AliasesHash:       DigestWrapped(s.AliasesHash),
				WavesBalanceHash:  DigestWrapped(s.WavesBalanceHash),
				AssetBalanceHash:  DigestWrapped(s.AssetBalanceHash),
				LeaseBalanceHash:  DigestWrapped(s.LeaseBalanceHash),
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

func NewStateHashJSDebug(s StateHash, h uint64, v string, snapshotStateHash crypto.Digest) StateHashDebug {
	return StateHashDebug{s.toStateHashJS(), h, v, snapshotStateHash}
}

func (s StateHashDebug) GetStateHash() *StateHash {
	sh := &StateHash{
		BlockID: s.BlockID,
		SumHash: crypto.Digest(s.SumHash),
		FieldsHashes: FieldsHashes{
			crypto.Digest(s.DataEntryHash),
			crypto.Digest(s.AccountScriptHash),
			crypto.Digest(s.AssetScriptHash),
			crypto.Digest(s.LeaseStatusHash),
			crypto.Digest(s.SponsorshipHash),
			crypto.Digest(s.AliasesHash),
			crypto.Digest(s.WavesBalanceHash),
			crypto.Digest(s.AssetBalanceHash),
			crypto.Digest(s.LeaseBalanceHash),
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
	if s == "null" {
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
	DataEntryHash     DigestWrapped `json:"dataEntryHash"`
	AccountScriptHash DigestWrapped `json:"accountScriptHash"`
	AssetScriptHash   DigestWrapped `json:"assetScriptHash"`
	LeaseStatusHash   DigestWrapped `json:"leaseStatusHash"`
	SponsorshipHash   DigestWrapped `json:"sponsorshipHash"`
	AliasesHash       DigestWrapped `json:"aliasHash"`
	WavesBalanceHash  DigestWrapped `json:"wavesBalanceHash"`
	AssetBalanceHash  DigestWrapped `json:"assetBalanceHash"`
	LeaseBalanceHash  DigestWrapped `json:"leaseBalanceHash"`
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
