package proto

import (
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type BlockSnapshot struct {
	TxSnapshots [][]AtomicSnapshot
}

func (bs *BlockSnapshot) AppendTxSnapshot(txSnapshot []AtomicSnapshot) {
	bs.TxSnapshots = append(bs.TxSnapshots, txSnapshot)
}

func (bs BlockSnapshot) MarshallBinary() ([]byte, error) {
	result := binary.BigEndian.AppendUint32([]byte{}, uint32(len(bs.TxSnapshots)))
	for _, ts := range bs.TxSnapshots {
		var res g.TransactionStateSnapshot
		for _, atomicSnapshot := range ts {
			if err := atomicSnapshot.AppendToProtobuf(&res); err != nil {
				return nil, errors.Wrap(err, "failed to marshall TransactionSnapshot to proto")
			}
		}
		tsBytes, err := res.MarshalVTStrict()
		if err != nil {
			return nil, err
		}
		result = binary.BigEndian.AppendUint32(result, uint32(len(tsBytes)))
		result = append(result, tsBytes...)
	}
	return result, nil
}

func (bs *BlockSnapshot) UnmarshalBinary(data []byte, scheme Scheme) error {
	if len(data) < uint32Size {
		return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid data size")
	}
	txSnCnt := binary.BigEndian.Uint32(data[0:uint32Size])
	data = data[uint32Size:]
	var txSnapshots [][]AtomicSnapshot
	for i := uint32(0); i < txSnCnt; i++ {
		if len(data) < uint32Size {
			return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid data size")
		}
		tsBytesLen := binary.BigEndian.Uint32(data[0:uint32Size])
		var tsProto g.TransactionStateSnapshot
		data = data[uint32Size:]
		if uint32(len(data)) < tsBytesLen {
			return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid snapshot size")
		}
		err := tsProto.UnmarshalVT(data[0:tsBytesLen])
		if err != nil {
			return err
		}
		atomicTS, err := TxSnapshotsFromProtobuf(scheme, &tsProto)
		if err != nil {
			return err
		}
		txSnapshots = append(txSnapshots, atomicTS)
		data = data[tsBytesLen:]
	}
	bs.TxSnapshots = txSnapshots
	return nil
}

func (bs BlockSnapshot) MarshalJSON() ([]byte, error) {
	if len(bs.TxSnapshots) == 0 {
		return []byte("[]"), nil
	}
	res := make([]txSnapshotJSON, 0, len(bs.TxSnapshots))
	for _, txSnapshot := range bs.TxSnapshots {
		var js txSnapshotJSON
		for _, snapshot := range txSnapshot {
			if err := snapshot.Apply(&js); err != nil {
				return nil, err
			}
		}
		res = append(res, js)
	}
	return json.Marshal(res)
}

func (bs *BlockSnapshot) UnmarshalJSON(bytes []byte) error {
	var blockSnapshotJSON []txSnapshotJSON
	if err := json.Unmarshal(bytes, &blockSnapshotJSON); err != nil {
		return err
	}
	if len(blockSnapshotJSON) == 0 {
		bs.TxSnapshots = nil
		return nil
	}
	res := make([][]AtomicSnapshot, 0, len(blockSnapshotJSON))
	for _, js := range blockSnapshotJSON {
		txSnapshot, err := js.toTransactionSnapshot()
		if err != nil {
			return err
		}
		res = append(res, txSnapshot)
	}
	bs.TxSnapshots = res
	return nil
}

type balanceSnapshotJSON struct {
	Address WavesAddress  `json:"address"`
	Asset   OptionalAsset `json:"asset"`
	Balance uint64        `json:"balance"`
}

type txSnapshotJSON struct {
	ApplicationStatus         TransactionStatus                          `json:"applicationStatus"`
	Balances                  NonNullableSlice[balanceSnapshotJSON]      `json:"balances"`
	LeaseBalances             NonNullableSlice[LeaseBalanceSnapshot]     `json:"leaseBalances"`
	AssetStatics              NonNullableSlice[NewAssetSnapshot]         `json:"assetStatics"`
	AssetVolumes              NonNullableSlice[AssetVolumeSnapshot]      `json:"assetVolumes"`
	AssetNamesAndDescriptions NonNullableSlice[AssetDescriptionSnapshot] `json:"assetNamesAndDescriptions"`
	AssetScripts              NonNullableSlice[AssetScriptSnapshot]      `json:"assetScripts"`
	Sponsorships              NonNullableSlice[SponsorshipSnapshot]      `json:"sponsorships"`
	NewLeases                 NonNullableSlice[NewLeaseSnapshot]         `json:"newLeases"`
	CancelledLeases           NonNullableSlice[CancelledLeaseSnapshot]   `json:"cancelledLeases"`
	Aliases                   NonNullableSlice[AliasSnapshot]            `json:"aliases"`
	OrderFills                NonNullableSlice[FilledVolumeFeeSnapshot]  `json:"orderFills"`
	AccountScripts            NonNullableSlice[AccountScriptSnapshot]    `json:"accountScripts"`
	AccountData               NonNullableSlice[DataEntriesSnapshot]      `json:"accountData"`
}

func (s *txSnapshotJSON) MarshalJSON() ([]byte, error) {
	if s.ApplicationStatus == unknownTransactionStatus {
		return nil, errors.New("empty transaction status")
	}
	type shadowed txSnapshotJSON
	return json.Marshal((*shadowed)(s))
}

func (s *txSnapshotJSON) UnmarshalJSON(bytes []byte) error {
	type shadowed txSnapshotJSON
	if err := json.Unmarshal(bytes, (*shadowed)(s)); err != nil {
		return err
	}
	if s.ApplicationStatus == unknownTransactionStatus {
		return errors.New("empty transaction status")
	}
	return nil
}

func (s *txSnapshotJSON) snapshotsCount() int {
	return 1 + // ApplicationStatus == TransactionStatusSnapshot
		len(s.Balances) +
		len(s.LeaseBalances) +
		len(s.AssetStatics) +
		len(s.AssetVolumes) +
		len(s.AssetNamesAndDescriptions) +
		len(s.AssetScripts) +
		len(s.Sponsorships) +
		len(s.NewLeases) +
		len(s.CancelledLeases) +
		len(s.Aliases) +
		len(s.OrderFills) +
		len(s.AccountScripts) +
		len(s.AccountData)
}

func (s *txSnapshotJSON) toTransactionSnapshot() ([]AtomicSnapshot, error) {
	if s.ApplicationStatus == unknownTransactionStatus {
		return nil, errors.New("empty transaction status")
	}
	res := make([]AtomicSnapshot, 0, s.snapshotsCount())
	res = append(res, &TransactionStatusSnapshot{Status: s.ApplicationStatus})
	for _, bs := range s.Balances {
		if bs.Asset.Present {
			res = append(res, &AssetBalanceSnapshot{
				Address: bs.Address,
				AssetID: bs.Asset.ID,
				Balance: bs.Balance,
			})
		} else {
			res = append(res, &WavesBalanceSnapshot{
				Address: bs.Address,
				Balance: bs.Balance,
			})
		}
	}
	for i := range s.LeaseBalances {
		res = append(res, &s.LeaseBalances[i])
	}
	for i := range s.AssetStatics {
		res = append(res, &s.AssetStatics[i])
	}
	for i := range s.AssetVolumes {
		res = append(res, &s.AssetVolumes[i])
	}
	for i := range s.AssetNamesAndDescriptions {
		res = append(res, &s.AssetNamesAndDescriptions[i])
	}
	for i := range s.AssetScripts {
		res = append(res, &s.AssetScripts[i])
	}
	for i := range s.Sponsorships {
		res = append(res, &s.Sponsorships[i])
	}
	for i := range s.NewLeases {
		res = append(res, &s.NewLeases[i])
	}
	for i := range s.CancelledLeases {
		res = append(res, &s.CancelledLeases[i])
	}
	for i := range s.Aliases {
		res = append(res, &s.Aliases[i])
	}
	for i := range s.OrderFills {
		res = append(res, &s.OrderFills[i])
	}
	for i := range s.AccountScripts {
		res = append(res, &s.AccountScripts[i])
	}
	for i := range s.AccountData {
		res = append(res, &s.AccountData[i])
	}
	return res, nil
}

func (s *txSnapshotJSON) ApplyWavesBalance(snapshot WavesBalanceSnapshot) error {
	bs := balanceSnapshotJSON{
		Address: snapshot.Address,
		Asset:   NewOptionalAssetWaves(),
		Balance: snapshot.Balance,
	}
	s.Balances = append(s.Balances, bs)
	return nil
}

func (s *txSnapshotJSON) ApplyLeaseBalance(snapshot LeaseBalanceSnapshot) error {
	s.LeaseBalances = append(s.LeaseBalances, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyAssetBalance(snapshot AssetBalanceSnapshot) error {
	bs := balanceSnapshotJSON{
		Address: snapshot.Address,
		Asset:   *NewOptionalAssetFromDigest(snapshot.AssetID),
		Balance: snapshot.Balance,
	}
	s.Balances = append(s.Balances, bs)
	return nil
}

func (s *txSnapshotJSON) ApplyAlias(snapshot AliasSnapshot) error {
	s.Aliases = append(s.Aliases, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyNewAsset(snapshot NewAssetSnapshot) error {
	s.AssetStatics = append(s.AssetStatics, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyAssetDescription(snapshot AssetDescriptionSnapshot) error {
	s.AssetNamesAndDescriptions = append(s.AssetNamesAndDescriptions, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyAssetVolume(snapshot AssetVolumeSnapshot) error {
	s.AssetVolumes = append(s.AssetVolumes, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyAssetScript(snapshot AssetScriptSnapshot) error {
	s.AssetScripts = append(s.AssetScripts, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplySponsorship(snapshot SponsorshipSnapshot) error {
	s.Sponsorships = append(s.Sponsorships, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyAccountScript(snapshot AccountScriptSnapshot) error {
	s.AccountScripts = append(s.AccountScripts, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyFilledVolumeAndFee(snapshot FilledVolumeFeeSnapshot) error {
	s.OrderFills = append(s.OrderFills, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyDataEntries(snapshot DataEntriesSnapshot) error {
	s.AccountData = append(s.AccountData, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyNewLease(snapshot NewLeaseSnapshot) error {
	s.NewLeases = append(s.NewLeases, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyCancelledLease(snapshot CancelledLeaseSnapshot) error {
	s.CancelledLeases = append(s.CancelledLeases, snapshot)
	return nil
}

func (s *txSnapshotJSON) ApplyTransactionsStatus(snapshot TransactionStatusSnapshot) error {
	if s.ApplicationStatus != unknownTransactionStatus {
		return errors.New("transaction status already set")
	}
	s.ApplicationStatus = snapshot.Status
	return nil
}
