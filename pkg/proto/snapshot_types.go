package proto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type AtomicSnapshot interface {
	Apply(SnapshotApplier) error
	Equal(otherSnapshot AtomicSnapshot) (bool, error)
	String() string
	AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error
}

type WavesBalanceSnapshot struct {
	Address WavesAddress `json:"address"`
	Balance uint64       `json:"balance"`
}

func (s WavesBalanceSnapshot) MarshalJSON() ([]byte, error) {
	out := struct {
		WavesBalanceSnapshot
		Asset OptionalAsset `json:"asset"`
	}{s, NewOptionalAssetWaves()}
	return json.Marshal(out)
}

func (s WavesBalanceSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*WavesBalanceSnapshot)
	if !ok {
		return false, errors.Errorf("expected WavesBalanceSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.Address == other.Address && s.Balance == other.Balance, nil
}

func (s WavesBalanceSnapshot) String() string {
	return fmt.Sprintf("WavesBalanceSnapshot{Address: %s, Balance: %d}", s.Address.String(), s.Balance)
}

func (s WavesBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyWavesBalance(s) }

func (s WavesBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Balance, error) {
	return &g.TransactionStateSnapshot_Balance{
		Address: s.Address.Bytes(),
		Amount: &g.Amount{
			AssetId: nil,
			Amount:  int64(s.Balance),
		},
	}, nil
}

func (s WavesBalanceSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.Balances = append(txSnapshots.Balances, snapshotInProto)
	return nil
}

func (s *WavesBalanceSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_Balance) error {
	var c ProtobufConverter
	addr, err := NewAddressFromBytesChecked(scheme, p.Address)
	if err != nil {
		return err
	}
	asset, amount := c.convertAmount(p.Amount)
	if c.err != nil {
		return c.err
	}
	if asset.Present {
		return errors.New("failed to unmarshal waves balance snapshot: asset is present")
	}
	s.Address = addr
	s.Balance = amount
	return nil
}

type AssetBalanceSnapshot struct {
	Address WavesAddress  `json:"address"`
	AssetID crypto.Digest `json:"asset"`
	Balance uint64        `json:"balance"`
}

func (s AssetBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetBalance(s) }

func (s AssetBalanceSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AssetBalanceSnapshot)
	if !ok {
		return false, errors.Errorf("expected AssetBalanceSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.Address == other.Address && s.AssetID == other.AssetID && s.Balance == other.Balance, nil
}

func (s AssetBalanceSnapshot) String() string {
	return fmt.Sprintf("AssetBalanceSnapshot{Address: %s, AssetID: %s, Balance: %d}",
		s.Address.String(), s.AssetID.String(), s.Balance)
}

func (s AssetBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Balance, error) {
	return &g.TransactionStateSnapshot_Balance{
		Address: s.Address.Bytes(),
		Amount: &g.Amount{
			AssetId: s.AssetID.Bytes(),
			Amount:  int64(s.Balance),
		},
	}, nil
}

func (s *AssetBalanceSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_Balance) error {
	var c ProtobufConverter
	addr, err := NewAddressFromBytesChecked(scheme, p.Address)
	if err != nil {
		return err
	}
	asset, amount := c.convertAmount(p.Amount)
	if c.err != nil {
		return c.err
	}
	if !asset.Present {
		return errors.New("failed to unmarshal asset balance snapshot: asset is not present")
	}
	s.Address = addr
	s.Balance = amount
	s.AssetID = asset.ID
	return nil
}

func (s AssetBalanceSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.Balances = append(txSnapshots.Balances, snapshotInProto)
	return nil
}

type DataEntriesSnapshot struct { // AccountData in pb
	Address     WavesAddress `json:"address"`
	DataEntries DataEntries  `json:"data"`
}

func (s DataEntriesSnapshot) Apply(a SnapshotApplier) error { return a.ApplyDataEntries(s) }

func (s DataEntriesSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*DataEntriesSnapshot)
	if !ok {
		return false, errors.Errorf("expected DataEntriesSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	if s.Address != other.Address || len(s.DataEntries) != len(other.DataEntries) {
		return false, nil
	}
	equalDE, err := s.DataEntries.Equal(other.DataEntries)
	if err != nil {
		return false, err
	}
	if !equalDE {
		return false, nil
	}
	return true, nil
}

func (s DataEntriesSnapshot) String() string {
	return fmt.Sprintf("DataEntriesSnapshot{Address: %s, DataEntries: %s}", s.Address.String(), s.DataEntries.String())
}

func (s DataEntriesSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AccountData, error) {
	entries := make([]*g.DataEntry, 0, len(s.DataEntries))
	for _, e := range s.DataEntries {
		entries = append(entries, e.ToProtobuf())
	}
	return &g.TransactionStateSnapshot_AccountData{
		Address: s.Address.Bytes(),
		Entries: entries,
	}, nil
}

func (s DataEntriesSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AccountData = append(txSnapshots.AccountData, snapshotInProto)
	return nil
}

func (s *DataEntriesSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_AccountData) error {
	var c ProtobufConverter
	addr, err := NewAddressFromBytesChecked(scheme, p.Address)
	if err != nil {
		return err
	}
	dataEntries := make([]DataEntry, 0, len(p.Entries))
	for _, e := range p.Entries {
		dataEntries = append(dataEntries, c.entry(e))
		if c.err != nil {
			return c.err
		}
	}
	s.Address = addr
	s.DataEntries = dataEntries
	return nil
}

type AccountScriptSnapshot struct {
	SenderPublicKey    crypto.PublicKey `json:"publicKey"`
	Script             Script           `json:"script"`
	VerifierComplexity uint64           `json:"verifierComplexity"`
}

func (s AccountScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAccountScript(s) }

func (s AccountScriptSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AccountScriptSnapshot)
	if !ok {
		return false, errors.Errorf("expected AccountScriptSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}

	return s.SenderPublicKey == other.SenderPublicKey &&
		bytes.Equal(s.Script, other.Script) && s.VerifierComplexity == other.VerifierComplexity, nil
}

func (s AccountScriptSnapshot) String() string {
	return fmt.Sprintf("AccountScriptSnapshot{SenderPublicKey: %s, Script: %x, VerifierComplexity: %d}",
		s.SenderPublicKey.String(), s.Script.String(), s.VerifierComplexity)
}

func (s AccountScriptSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AccountScript, error) {
	return &g.TransactionStateSnapshot_AccountScript{
		SenderPublicKey:    s.SenderPublicKey.Bytes(),
		Script:             s.Script,
		VerifierComplexity: int64(s.VerifierComplexity),
	}, nil
}

func (s AccountScriptSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	if txSnapshots.AccountScripts != nil { // sanity check
		return errors.New("protobuf account script field is already set")
	}
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AccountScripts = snapshotInProto
	return nil
}

func (s *AccountScriptSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AccountScript) error {
	var c ProtobufConverter
	publicKey := c.publicKey(p.SenderPublicKey)
	if c.err != nil {
		return c.err
	}
	script := c.script(p.Script)
	if c.err != nil {
		return c.err
	}
	verifierComplexity := c.uint64(p.VerifierComplexity)
	if c.err != nil {
		return c.err
	}
	s.SenderPublicKey = publicKey
	s.Script = script
	s.VerifierComplexity = verifierComplexity
	return nil
}

type AssetScriptSnapshot struct {
	AssetID crypto.Digest `json:"id"`
	Script  Script        `json:"script"`
	// json representation in scala node also has 'complexity' field, but it's always equal to 0, so it's omitted here
}

func (s AssetScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetScript(s) }

func (s AssetScriptSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AssetScriptSnapshot)
	if !ok {
		return false, errors.Errorf("expected AssetScriptSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.AssetID == other.AssetID && bytes.Equal(s.Script, other.Script), nil
}

func (s AssetScriptSnapshot) String() string {
	return fmt.Sprintf("AssetScriptSnapshot{AssetID: %s, Script: %x}", s.AssetID.String(), s.Script.String())
}

func (s AssetScriptSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetScript, error) {
	return &g.TransactionStateSnapshot_AssetScript{
		AssetId: s.AssetID.Bytes(),
		Script:  s.Script,
	}, nil
}

func (s AssetScriptSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	if txSnapshots.AssetScripts != nil { // sanity check
		return errors.New("protobuf asset script field is already set")
	}
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetScripts = snapshotInProto
	return nil
}

func (s *AssetScriptSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AssetScript) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}
	script := c.script(p.Script)
	if c.err != nil {
		return c.err
	}
	s.AssetID = assetID
	s.Script = script
	return nil
}

type LeaseBalanceSnapshot struct {
	Address  WavesAddress `json:"address"`
	LeaseIn  uint64       `json:"in"`
	LeaseOut uint64       `json:"out"`
}

func (s LeaseBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseBalance(s) }

func (s LeaseBalanceSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*LeaseBalanceSnapshot)
	if !ok {
		return false, errors.Errorf("expected LeaseBalanceSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.Address == other.Address && s.LeaseIn == other.LeaseIn && s.LeaseOut == other.LeaseOut, nil
}

func (s LeaseBalanceSnapshot) String() string {
	return fmt.Sprintf("LeaseBalanceSnapshot{Address: %s, LeaseIn: %d, LeaseOut: %d}",
		s.Address.String(), s.LeaseIn, s.LeaseOut)
}

func (s LeaseBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_LeaseBalance, error) {
	return &g.TransactionStateSnapshot_LeaseBalance{
		Address: s.Address.Bytes(),
		In:      int64(s.LeaseIn),
		Out:     int64(s.LeaseOut),
	}, nil
}

func (s LeaseBalanceSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.LeaseBalances = append(txSnapshots.LeaseBalances, snapshotInProto)
	return nil
}

func (s *LeaseBalanceSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_LeaseBalance) error {
	addr, err := NewAddressFromBytesChecked(scheme, p.Address)
	if err != nil {
		return err
	}
	in := uint64(p.In)
	out := uint64(p.Out)
	s.Address = addr
	s.LeaseIn = in
	s.LeaseOut = out
	return nil
}

type NewLeaseSnapshot struct {
	LeaseID       crypto.Digest    `json:"id"`
	Amount        uint64           `json:"amount"`
	SenderPK      crypto.PublicKey `json:"sender"`
	RecipientAddr WavesAddress     `json:"recipient"`
	// json representation in scala node also has 'height' and 'txId' fields,
	// but they aren't important, so omitted
}

func (s NewLeaseSnapshot) Apply(a SnapshotApplier) error { return a.ApplyNewLease(s) }

func (s NewLeaseSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*NewLeaseSnapshot)
	if !ok {
		return false, errors.Errorf("expected NewLeaseSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.LeaseID == other.LeaseID &&
		s.Amount == other.Amount &&
		s.SenderPK == other.SenderPK &&
		s.RecipientAddr == other.RecipientAddr, nil
}

func (s NewLeaseSnapshot) String() string {
	return fmt.Sprintf("NewLeaseSnapshot{LeaseID: %s, Amount: %d, SenderPK: %s, RecipientAddr: %s}",
		s.LeaseID.String(), s.Amount, s.SenderPK.String(), s.RecipientAddr.String())
}

func (s NewLeaseSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_NewLease, error) {
	return &g.TransactionStateSnapshot_NewLease{
		LeaseId:          s.LeaseID.Bytes(),
		SenderPublicKey:  s.SenderPK.Bytes(),
		RecipientAddress: s.RecipientAddr.Bytes(),
		Amount:           int64(s.Amount),
	}, nil
}

func (s NewLeaseSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.NewLeases = append(txSnapshots.NewLeases, snapshotInProto)
	return nil
}

func (s *NewLeaseSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_NewLease) error {
	var c ProtobufConverter
	leaseID := c.digest(p.LeaseId)
	if c.err != nil {
		return c.err
	}
	senderPK := c.publicKey(p.SenderPublicKey)
	if c.err != nil {
		return c.err
	}
	amount := c.uint64(p.Amount)
	if c.err != nil {
		return c.err
	}
	recipientAddr, err := NewAddressFromBytesChecked(scheme, p.RecipientAddress)
	if err != nil {
		return err
	}
	s.LeaseID = leaseID
	s.Amount = amount
	s.SenderPK = senderPK
	s.RecipientAddr = recipientAddr
	return nil
}

type CancelledLeaseSnapshot struct {
	LeaseID crypto.Digest `json:"id"`
	// json representation in scala node also has 'height' and 'txId' fields,
	// but they aren't important, so omitted
}

func (s CancelledLeaseSnapshot) Apply(a SnapshotApplier) error { return a.ApplyCancelledLease(s) }

func (s CancelledLeaseSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*CancelledLeaseSnapshot)
	if !ok {
		return false, errors.Errorf("expected CancelledLeaseSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.LeaseID == other.LeaseID, nil
}

func (s CancelledLeaseSnapshot) String() string {
	return fmt.Sprintf("CancelledLeaseSnapshot{LeaseID: %s}", s.LeaseID.String())
}

func (s CancelledLeaseSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_CancelledLease, error) {
	return &g.TransactionStateSnapshot_CancelledLease{
		LeaseId: s.LeaseID.Bytes(),
	}, nil
}

func (s CancelledLeaseSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.CancelledLeases = append(txSnapshots.CancelledLeases, snapshotInProto)
	return nil
}

func (s *CancelledLeaseSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_CancelledLease) error {
	var c ProtobufConverter
	leaseID := c.digest(p.LeaseId)
	if c.err != nil {
		return c.err
	}
	s.LeaseID = leaseID
	return nil
}

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest `json:"id"`
	MinSponsoredFee uint64        `json:"minSponsoredAssetFee"`
}

func (s SponsorshipSnapshot) Apply(a SnapshotApplier) error { return a.ApplySponsorship(s) }

func (s SponsorshipSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*SponsorshipSnapshot)
	if !ok {
		return false, errors.Errorf("expected SponsorshipSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.AssetID == other.AssetID && s.MinSponsoredFee == other.MinSponsoredFee, nil
}

func (s SponsorshipSnapshot) String() string {
	return fmt.Sprintf("SponsorshipSnapshot{AssetID: %s, MinSponsoredFee: %d}", s.AssetID.String(), s.MinSponsoredFee)
}

func (s SponsorshipSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Sponsorship, error) {
	return &g.TransactionStateSnapshot_Sponsorship{
		AssetId: s.AssetID.Bytes(),
		MinFee:  int64(s.MinSponsoredFee),
	}, nil
}

func (s SponsorshipSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.Sponsorships = append(txSnapshots.Sponsorships, snapshotInProto)
	return nil
}

func (s *SponsorshipSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_Sponsorship) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}
	minFee := c.uint64(p.MinFee)
	if c.err != nil {
		return c.err
	}
	s.AssetID = assetID
	s.MinSponsoredFee = minFee
	return nil
}

type AliasSnapshot struct {
	Address WavesAddress `json:"address"`
	Alias   string       `json:"alias"`
}

func (s *AliasSnapshot) UnmarshalJSON(bytes []byte) error {
	type shadowed AliasSnapshot
	if err := json.Unmarshal(bytes, (*shadowed)(s)); err != nil {
		return err
	}
	_, err := IsValidAliasString(s.Alias)
	return err
}

func (s AliasSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAlias(s) }

func (s AliasSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AliasSnapshot)
	if !ok {
		return false, errors.Errorf("expected AliasSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.Address == other.Address && s.Alias == other.Alias, nil
}

func (s AliasSnapshot) String() string {
	return fmt.Sprintf("AliasSnapshot{Address: %s, Alias: %s}", s.Address.String(), s.Alias)
}

func (s AliasSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Alias, error) {
	return &g.TransactionStateSnapshot_Alias{
		Address: s.Address.Bytes(),
		Alias:   s.Alias,
	}, nil
}

func (s AliasSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	if txSnapshots.Aliases != nil { // sanity check
		return errors.New("protobuf alias field is already set")
	}
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.Aliases = snapshotInProto
	return nil
}

func (s *AliasSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_Alias) error {
	addr, err := NewAddressFromBytesChecked(scheme, p.Address)
	if err != nil {
		return err
	}
	if _, aErr := IsValidAliasString(p.Alias); aErr != nil {
		return aErr
	}
	s.Address = addr
	s.Alias = p.Alias
	return nil
}

// FilledVolumeFeeSnapshot Filled Volume and Fee.
type FilledVolumeFeeSnapshot struct { // OrderFill
	OrderID      crypto.Digest `json:"id"`
	FilledVolume uint64        `json:"volume"`
	FilledFee    uint64        `json:"fee"`
}

func (s FilledVolumeFeeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyFilledVolumeAndFee(s) }

func (s FilledVolumeFeeSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*FilledVolumeFeeSnapshot)
	if !ok {
		return false, errors.Errorf("expected FilledVolumeFeeSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.OrderID == other.OrderID && s.FilledVolume == other.FilledVolume && s.FilledFee == other.FilledFee, nil
}

func (s FilledVolumeFeeSnapshot) String() string {
	return fmt.Sprintf("FilledVolumeFeeSnapshot{OrderID: %s, FilledVolume: %d, FilledFee: %d}",
		s.OrderID.String(), s.FilledVolume, s.FilledFee)
}

func (s FilledVolumeFeeSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_OrderFill, error) {
	return &g.TransactionStateSnapshot_OrderFill{
		OrderId: s.OrderID.Bytes(),
		Volume:  int64(s.FilledVolume),
		Fee:     int64(s.FilledFee),
	}, nil
}

func (s FilledVolumeFeeSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.OrderFills = append(txSnapshots.OrderFills, snapshotInProto)
	return nil
}

func (s *FilledVolumeFeeSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_OrderFill) error {
	var c ProtobufConverter
	orderID := c.digest(p.OrderId)
	if c.err != nil {
		return c.err
	}
	volume := c.uint64(p.Volume)
	if c.err != nil {
		return c.err
	}
	fee := c.uint64(p.Fee)
	if c.err != nil {
		return c.err
	}
	s.OrderID = orderID
	s.FilledVolume = volume
	s.FilledFee = fee
	return nil
}

type NewAssetSnapshot struct {
	AssetID         crypto.Digest    `json:"id"`
	IssuerPublicKey crypto.PublicKey `json:"issuer"`
	Decimals        uint8            `json:"decimals"`
	IsNFT           bool             `json:"nft"`
}

func (s NewAssetSnapshot) Apply(a SnapshotApplier) error { return a.ApplyNewAsset(s) }

func (s NewAssetSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*NewAssetSnapshot)
	if !ok {
		return false, errors.Errorf("expected NewAssetSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.AssetID == other.AssetID && s.IssuerPublicKey == other.IssuerPublicKey &&
		s.Decimals == other.Decimals && s.IsNFT == other.IsNFT, nil
}

func (s NewAssetSnapshot) String() string {
	return fmt.Sprintf("NewAssetSnapshot{AssetID: %s, IssuerPublicKey: %s, Decimals: %d, IsNFT: %t}",
		s.AssetID.String(), s.IssuerPublicKey.String(), s.Decimals, s.IsNFT)
}

func (s NewAssetSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_NewAsset, error) {
	return &g.TransactionStateSnapshot_NewAsset{
		AssetId:         s.AssetID.Bytes(),
		IssuerPublicKey: s.IssuerPublicKey.Bytes(),
		Decimals:        int32(s.Decimals),
		Nft:             s.IsNFT,
	}, nil
}

func (s NewAssetSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetStatics = append(txSnapshots.AssetStatics, snapshotInProto)
	return nil
}

func (s *NewAssetSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_NewAsset) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}
	publicKey := c.publicKey(p.IssuerPublicKey)
	if c.err != nil {
		return c.err
	}
	decimals := c.byte(p.Decimals)
	if c.err != nil {
		return c.err
	}
	s.AssetID = assetID
	s.IssuerPublicKey = publicKey
	s.Decimals = decimals
	s.IsNFT = p.Nft
	return nil
}

type AssetVolumeSnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest `json:"id"`
	TotalQuantity big.Int       `json:"volume"` // volume in protobuf
	IsReissuable  bool          `json:"isReissuable"`
}

func (s AssetVolumeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetVolume(s) }

func (s AssetVolumeSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AssetVolumeSnapshot)
	if !ok {
		return false, errors.Errorf("expected AssetVolumeSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.AssetID == other.AssetID &&
		s.TotalQuantity.Cmp(&other.TotalQuantity) == 0 && s.IsReissuable == other.IsReissuable, nil
}

func (s AssetVolumeSnapshot) String() string {
	return fmt.Sprintf("AssetVolumeSnapshot{AssetID: %s, TotalQuantity: %s, IsReissuable: %t}",
		s.AssetID.String(), s.TotalQuantity.String(), s.IsReissuable)
}

func (s AssetVolumeSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetVolume, error) {
	return &g.TransactionStateSnapshot_AssetVolume{
		AssetId:    s.AssetID.Bytes(),
		Reissuable: s.IsReissuable,
		Volume:     common.Encode2CBigInt(&s.TotalQuantity),
	}, nil
}

func (s AssetVolumeSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetVolumes = append(txSnapshots.AssetVolumes, snapshotInProto)
	return nil
}

func (s *AssetVolumeSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AssetVolume) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}

	s.AssetID = assetID
	s.TotalQuantity = *common.Decode2CBigInt(p.Volume)
	s.IsReissuable = p.Reissuable
	return nil
}

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest `json:"id"`
	AssetName        string        `json:"name"`
	AssetDescription string        `json:"description"`
	// json representation in scala node also has 'lastUpdatedAt' field, but it's not important, so omitted
}

func (s AssetDescriptionSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetDescription(s) }

func (s AssetDescriptionSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*AssetDescriptionSnapshot)
	if !ok {
		return false, errors.Errorf("expected AssetDescriptionSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.AssetID == other.AssetID && s.AssetName == other.AssetName &&
		s.AssetDescription == other.AssetDescription, nil
}

func (s AssetDescriptionSnapshot) String() string {
	return fmt.Sprintf("AssetDescriptionSnapshot{AssetID: %s, AssetName: %s, AssetDescription: %s}",
		s.AssetID.String(), s.AssetName, s.AssetDescription)
}

func (s AssetDescriptionSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetNameAndDescription, error) {
	return &g.TransactionStateSnapshot_AssetNameAndDescription{
		AssetId:     s.AssetID.Bytes(),
		Name:        s.AssetName,
		Description: s.AssetDescription,
	}, nil
}

func (s AssetDescriptionSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetNamesAndDescriptions = append(txSnapshots.AssetNamesAndDescriptions, snapshotInProto)
	return nil
}

func (s *AssetDescriptionSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AssetNameAndDescription) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}

	s.AssetID = assetID
	s.AssetName = p.Name
	s.AssetDescription = p.Description
	return nil
}

type TransactionStatusSnapshot struct {
	Status TransactionStatus `json:"transactionStatus"` // this is not canonical json representation
}

func (s TransactionStatusSnapshot) Apply(a SnapshotApplier) error {
	return a.ApplyTransactionsStatus(s)
}

func (s TransactionStatusSnapshot) Equal(otherSnapshot AtomicSnapshot) (bool, error) {
	other, ok := otherSnapshot.(*TransactionStatusSnapshot)
	if !ok {
		return false, errors.Errorf("expected TransactionStatusSnapshot, received %s", reflect.TypeOf(otherSnapshot))
	}
	return s.Status == other.Status, nil
}

func (s TransactionStatusSnapshot) String() string {
	return fmt.Sprintf("TransactionStatusSnapshot{Status: %s}", s.Status.String())
}

func (s *TransactionStatusSnapshot) FromProtobuf(p g.TransactionStatus) error {
	switch p {
	case g.TransactionStatus_SUCCEEDED:
		s.Status = TransactionSucceeded
	case g.TransactionStatus_FAILED:
		s.Status = TransactionFailed
	case g.TransactionStatus_ELIDED:
		s.Status = TransactionElided
	default:
		return errors.Errorf("undefinded tx status %d", p)
	}
	return nil
}

func (s TransactionStatusSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	switch s.Status {
	case TransactionSucceeded:
		txSnapshots.TransactionStatus = g.TransactionStatus_SUCCEEDED
	case TransactionElided:
		txSnapshots.TransactionStatus = g.TransactionStatus_ELIDED
	case TransactionFailed:
		txSnapshots.TransactionStatus = g.TransactionStatus_FAILED
	default:
		return errors.Errorf("undefined tx status %d", s.Status)
	}
	return nil
}

type TxSnapshot []AtomicSnapshot

func (a TxSnapshot) Len() int      { return len(a) }
func (a TxSnapshot) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TxSnapshot) Less(i, j int) bool {
	return a[i].String() < a[j].String()
}

func (a TxSnapshot) Equal(other TxSnapshot) (bool, error) {
	if len(a) != len(other) {
		return false, nil
	}
	SortAtomicSnapshotsByType(a)
	SortAtomicSnapshotsByType(other)
	for i := range a {
		equal, err := a[i].Equal(other[i])
		if err != nil {
			return false, err
		}
		if !equal {
			return false, nil
		}
	}
	return true, nil
}

func SortAtomicSnapshotsByType(snapshots TxSnapshot) {
	sort.Sort(snapshots)
}

type SnapshotApplier interface {
	ApplyWavesBalance(snapshot WavesBalanceSnapshot) error
	ApplyLeaseBalance(snapshot LeaseBalanceSnapshot) error
	ApplyAssetBalance(snapshot AssetBalanceSnapshot) error
	ApplyAlias(snapshot AliasSnapshot) error
	ApplyNewAsset(snapshot NewAssetSnapshot) error
	ApplyAssetDescription(snapshot AssetDescriptionSnapshot) error
	ApplyAssetVolume(snapshot AssetVolumeSnapshot) error
	ApplyAssetScript(snapshot AssetScriptSnapshot) error
	ApplySponsorship(snapshot SponsorshipSnapshot) error
	ApplyAccountScript(snapshot AccountScriptSnapshot) error
	ApplyFilledVolumeAndFee(snapshot FilledVolumeFeeSnapshot) error
	ApplyDataEntries(snapshot DataEntriesSnapshot) error
	ApplyNewLease(snapshot NewLeaseSnapshot) error
	ApplyCancelledLease(snapshot CancelledLeaseSnapshot) error
	ApplyTransactionsStatus(snapshot TransactionStatusSnapshot) error
}
