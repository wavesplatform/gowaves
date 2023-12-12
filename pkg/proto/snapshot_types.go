package proto

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type AtomicSnapshot interface {
	Apply(SnapshotApplier) error
	AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error
}
type WavesBalanceSnapshot struct {
	Address WavesAddress
	Balance uint64
}

func (s WavesBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyWavesBalance(s) }

func (s WavesBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Balance, error) {
	return &g.TransactionStateSnapshot_Balance{
		Address: s.Address.Body(),
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
	addr, err := c.Address(scheme, p.Address)
	if err != nil {
		return err
	}
	amount := c.amount(p.Amount)
	if c.err != nil {
		return err
	}
	s.Address = addr
	s.Balance = amount
	return nil
}

type AssetBalanceSnapshot struct {
	Address WavesAddress
	AssetID crypto.Digest
	Balance uint64
}

func (s AssetBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetBalance(s) }

func (s AssetBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Balance, error) {
	return &g.TransactionStateSnapshot_Balance{
		Address: s.Address.Body(),
		Amount: &g.Amount{
			AssetId: s.AssetID.Bytes(),
			Amount:  int64(s.Balance),
		},
	}, nil
}

func (s *AssetBalanceSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_Balance) error {
	var c ProtobufConverter
	addr, err := c.Address(scheme, p.Address)
	if err != nil {
		return err
	}
	amount := c.amount(p.Amount)
	if c.err != nil {
		return c.err
	}
	asset := c.extractOptionalAsset(p.Amount)
	if c.err != nil {
		return c.err
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
	Address     WavesAddress
	DataEntries []DataEntry
}

func (s DataEntriesSnapshot) Apply(a SnapshotApplier) error { return a.ApplyDataEntries(s) }

func (s DataEntriesSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AccountData, error) {
	entries := make([]*g.DataTransactionData_DataEntry, 0, len(s.DataEntries))
	for _, e := range s.DataEntries {
		entries = append(entries, e.ToProtobuf())
	}
	return &g.TransactionStateSnapshot_AccountData{
		Address: s.Address.Body(),
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
	addr, err := c.Address(scheme, p.Address)
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
	SenderPublicKey    crypto.PublicKey
	Script             Script
	VerifierComplexity uint64
}

func (s AccountScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAccountScript(s) }

func (s AccountScriptSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AccountScript, error) {
	return &g.TransactionStateSnapshot_AccountScript{
		SenderPublicKey:    s.SenderPublicKey.Bytes(),
		Script:             s.Script,
		VerifierComplexity: int64(s.VerifierComplexity),
	}, nil
}

func (s AccountScriptSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AccountScripts = append(txSnapshots.AccountScripts, snapshotInProto)
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
	s.SenderPublicKey = publicKey
	s.Script = script
	s.VerifierComplexity = uint64(p.VerifierComplexity)
	return nil
}

type AssetScriptSnapshot struct {
	AssetID crypto.Digest
	Script  Script
}

func (s AssetScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetScript(s) }

func (s AssetScriptSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetScript, error) {
	return &g.TransactionStateSnapshot_AssetScript{
		AssetId: s.AssetID.Bytes(),
		Script:  s.Script,
	}, nil
}

func (s AssetScriptSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetScripts = append(txSnapshots.AssetScripts, snapshotInProto)
	return nil
}

func (s *AssetScriptSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AssetScript) error {
	var c ProtobufConverter
	asset := c.optionalAsset(p.AssetId)
	if c.err != nil {
		return c.err
	}
	script := c.script(p.Script)
	if c.err != nil {
		return c.err
	}
	s.AssetID = asset.ID
	s.Script = script
	return nil
}

type LeaseBalanceSnapshot struct {
	Address  WavesAddress
	LeaseIn  uint64
	LeaseOut uint64
}

func (s LeaseBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseBalance(s) }

func (s LeaseBalanceSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_LeaseBalance, error) {
	return &g.TransactionStateSnapshot_LeaseBalance{
		Address: s.Address.Body(),
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
	var c ProtobufConverter
	addr, err := c.Address(scheme, p.Address)
	if err != nil {
		return err
	}
	s.Address = addr
	s.LeaseIn = uint64(p.In)
	s.LeaseOut = uint64(p.Out)
	return nil
}

type LeaseStateStatus interface{ leaseStateStatusMarker() }

type LeaseStateStatusActive struct {
	Amount    uint64
	Sender    WavesAddress
	Recipient WavesAddress
}

func (*LeaseStateStatusActive) leaseStateStatusMarker() {}

type LeaseStatusCancelled struct{}

func (*LeaseStatusCancelled) leaseStateStatusMarker() {}

type LeaseStateSnapshot struct {
	LeaseID crypto.Digest
	Status  LeaseStateStatus
}

func (s LeaseStateSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseState(s) }

func (s LeaseStateSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_LeaseState, error) {
	res := &g.TransactionStateSnapshot_LeaseState{
		LeaseId: s.LeaseID.Bytes(),
		Status:  nil,
	}
	switch status := s.Status.(type) {
	case *LeaseStateStatusActive:
		res.Status = &g.TransactionStateSnapshot_LeaseState_Active_{
			Active: &g.TransactionStateSnapshot_LeaseState_Active{
				Amount:    int64(status.Amount),
				Sender:    status.Sender.Body(),
				Recipient: status.Recipient.Body(),
			},
		}
	case *LeaseStatusCancelled:
		res.Status = &g.TransactionStateSnapshot_LeaseState_Cancelled_{
			Cancelled: &g.TransactionStateSnapshot_LeaseState_Cancelled{},
		}
	default:
		return nil, errors.Errorf("failed to serialize LeaseStateSnapshot to Proto: invalid Lease status")
	}
	return res, nil
}

func (s LeaseStateSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.LeaseStates = append(txSnapshots.LeaseStates, snapshotInProto)
	return nil
}

func (s *LeaseStateSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_LeaseState) error {
	var c ProtobufConverter
	leaseID := c.digest(p.LeaseId)
	if c.err != nil {
		return c.err
	}
	var status LeaseStateStatus
	if active := p.GetActive(); active != nil {
		sender, errAddressFromBytes := c.Address(scheme, active.Sender)
		if errAddressFromBytes != nil {
			return errAddressFromBytes
		}
		recipientAddr, errAddressFromBytes := c.Address(scheme, active.Recipient)
		if errAddressFromBytes != nil {
			return errAddressFromBytes
		}
		res := LeaseStateStatusActive{
			Amount:    uint64(active.Amount),
			Sender:    sender,
			Recipient: recipientAddr,
		}
		status = &res
	} else if cancel := p.GetCancelled(); cancel != nil {
		status = &LeaseStatusCancelled{}
	}
	s.LeaseID = leaseID
	s.Status = status
	return nil
}

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest
	MinSponsoredFee uint64
}

func (s SponsorshipSnapshot) Apply(a SnapshotApplier) error { return a.ApplySponsorship(s) }

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
	asset := c.optionalAsset(p.AssetId)
	if c.err != nil {
		return c.err
	}
	s.AssetID = asset.ID
	s.MinSponsoredFee = uint64(p.MinFee)
	return nil
}

type AliasSnapshot struct {
	Address WavesAddress
	Alias   Alias
}

func (s AliasSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAlias(s) }

func (s AliasSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_Alias, error) {
	return &g.TransactionStateSnapshot_Alias{
		Address: s.Address.Body(),
		Alias:   s.Alias.Alias,
	}, nil
}

func (s AliasSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.Aliases = append(txSnapshots.Aliases, snapshotInProto)
	return nil
}

func (s *AliasSnapshot) FromProtobuf(scheme Scheme, p *g.TransactionStateSnapshot_Alias) error {
	var c ProtobufConverter
	addr, err := c.Address(scheme, p.Address)
	if err != nil {
		return err
	}
	s.Address = addr
	s.Alias.Alias = p.Alias
	return nil
}

// FilledVolumeFeeSnapshot Filled Volume and Fee.
type FilledVolumeFeeSnapshot struct { // OrderFill
	OrderID      crypto.Digest
	FilledVolume uint64
	FilledFee    uint64
}

func (s FilledVolumeFeeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyFilledVolumeAndFee(s) }

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
	s.OrderID = orderID
	s.FilledVolume = uint64(p.Volume)
	s.FilledFee = uint64(p.Fee)
	return nil
}

type StaticAssetInfoSnapshot struct {
	AssetID             crypto.Digest
	SourceTransactionID crypto.Digest
	IssuerPublicKey     crypto.PublicKey
	Decimals            uint8
	IsNFT               bool
}

func (s StaticAssetInfoSnapshot) Apply(a SnapshotApplier) error { return a.ApplyStaticAssetInfo(s) }

func (s StaticAssetInfoSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetStatic, error) {
	return &g.TransactionStateSnapshot_AssetStatic{
		AssetId:             s.AssetID.Bytes(),
		SourceTransactionId: s.SourceTransactionID.Bytes(),
		IssuerPublicKey:     s.IssuerPublicKey.Bytes(),
		Decimals:            int32(s.Decimals),
		Nft:                 s.IsNFT,
	}, nil
}

func (s StaticAssetInfoSnapshot) AppendToProtobuf(txSnapshots *g.TransactionStateSnapshot) error {
	snapshotInProto, err := s.ToProtobuf()
	if err != nil {
		return err
	}
	txSnapshots.AssetStatics = append(txSnapshots.AssetStatics, snapshotInProto)
	return nil
}

func (s *StaticAssetInfoSnapshot) FromProtobuf(p *g.TransactionStateSnapshot_AssetStatic) error {
	var c ProtobufConverter
	assetID := c.digest(p.AssetId)
	if c.err != nil {
		return c.err
	}
	txID := c.digest(p.SourceTransactionId)
	if c.err != nil {
		return c.err
	}
	publicKey := c.publicKey(p.IssuerPublicKey)
	if c.err != nil {
		return c.err
	}
	s.AssetID = assetID
	s.SourceTransactionID = txID
	s.IssuerPublicKey = publicKey
	s.Decimals = uint8(p.Decimals)
	s.IsNFT = p.Nft
	return nil
}

type AssetVolumeSnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest
	TotalQuantity big.Int // volume in protobuf
	IsReissuable  bool
}

func (s AssetVolumeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetVolume(s) }

func (s AssetVolumeSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetVolume, error) {
	return &g.TransactionStateSnapshot_AssetVolume{
		AssetId:    s.AssetID.Bytes(),
		Reissuable: s.IsReissuable,
		Volume:     s.TotalQuantity.Bytes(),
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
	s.TotalQuantity.SetBytes(p.Volume)
	s.IsReissuable = p.Reissuable
	return nil
}

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest
	AssetName        string
	AssetDescription string
	ChangeHeight     Height // last_updated in pb
}

func (s AssetDescriptionSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetDescription(s) }

func (s AssetDescriptionSnapshot) ToProtobuf() (*g.TransactionStateSnapshot_AssetNameAndDescription, error) {
	return &g.TransactionStateSnapshot_AssetNameAndDescription{
		AssetId:     s.AssetID.Bytes(),
		Name:        s.AssetName,
		Description: s.AssetDescription,
		LastUpdated: int32(s.ChangeHeight),
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
	s.ChangeHeight = uint64(p.LastUpdated)
	return nil
}

type TransactionStatusSnapshot struct {
	TransactionID crypto.Digest
	Status        TransactionStatus
}

func (s TransactionStatusSnapshot) Apply(a SnapshotApplier) error {
	return a.ApplyTransactionsStatus(s)
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

type SnapshotApplier interface {
	ApplyWavesBalance(snapshot WavesBalanceSnapshot) error
	ApplyLeaseBalance(snapshot LeaseBalanceSnapshot) error
	ApplyAssetBalance(snapshot AssetBalanceSnapshot) error
	ApplyAlias(snapshot AliasSnapshot) error
	ApplyStaticAssetInfo(snapshot StaticAssetInfoSnapshot) error
	ApplyAssetDescription(snapshot AssetDescriptionSnapshot) error
	ApplyAssetVolume(snapshot AssetVolumeSnapshot) error
	ApplyAssetScript(snapshot AssetScriptSnapshot) error
	ApplySponsorship(snapshot SponsorshipSnapshot) error
	ApplyAccountScript(snapshot AccountScriptSnapshot) error
	ApplyFilledVolumeAndFee(snapshot FilledVolumeFeeSnapshot) error
	ApplyDataEntries(snapshot DataEntriesSnapshot) error
	ApplyLeaseState(snapshot LeaseStateSnapshot) error
	ApplyTransactionsStatus(snapshot TransactionStatusSnapshot) error
}
