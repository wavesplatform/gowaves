package main

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type balanceSnapshotJSON struct {
	Address proto.WavesAddress  `json:"address"`
	Asset   proto.OptionalAsset `json:"asset"`
	Balance uint64              `json:"balance"`
}

type txSnapshotJSON struct {
	ApplicationStatus         proto.TransactionStatus                                `json:"applicationStatus"`
	Balances                  proto.NonNullableSlice[balanceSnapshotJSON]            `json:"balances"`
	LeaseBalances             proto.NonNullableSlice[proto.LeaseBalanceSnapshot]     `json:"leaseBalances"`
	AssetStatics              proto.NonNullableSlice[proto.NewAssetSnapshot]         `json:"assetStatics"`
	AssetVolumes              proto.NonNullableSlice[proto.AssetVolumeSnapshot]      `json:"assetVolumes"`
	AssetNamesAndDescriptions proto.NonNullableSlice[proto.AssetDescriptionSnapshot] `json:"assetNamesAndDescriptions"`
	AssetScripts              proto.NonNullableSlice[proto.AssetScriptSnapshot]      `json:"assetScripts"`
	Sponsorships              proto.NonNullableSlice[proto.SponsorshipSnapshot]      `json:"sponsorships"`
	NewLeases                 proto.NonNullableSlice[proto.NewLeaseSnapshot]         `json:"newLeases"`
	CancelledLeases           proto.NonNullableSlice[proto.CancelledLeaseSnapshot]   `json:"cancelledLeases"`
	Aliases                   proto.NonNullableSlice[proto.AliasSnapshot]            `json:"aliases"`
	OrderFills                proto.NonNullableSlice[proto.FilledVolumeFeeSnapshot]  `json:"orderFills"`
	AccountScripts            proto.NonNullableSlice[proto.AccountScriptSnapshot]    `json:"accountScripts"`
	AccountData               proto.NonNullableSlice[proto.DataEntriesSnapshot]      `json:"accountData"`
}

func (s *txSnapshotJSON) sortFields() {
	sort.Slice(s.Balances, func(i, j int) bool {
		idI := s.Balances[i].Asset.String() + s.Balances[i].Address.String() + strconv.FormatUint(s.Balances[i].Balance, 10)
		idJ := s.Balances[j].Asset.String() + s.Balances[j].Address.String() + strconv.FormatUint(s.Balances[j].Balance, 10)
		return idI < idJ
	})
	sort.Slice(s.LeaseBalances, func(i, j int) bool {
		return s.LeaseBalances[i].Address.String() < s.LeaseBalances[j].Address.String()
	})
	sort.Slice(s.AssetStatics, func(i, j int) bool {
		return s.AssetStatics[i].AssetID.String() < s.AssetStatics[j].AssetID.String()
	})
	sort.Slice(s.AssetVolumes, func(i, j int) bool {
		return s.AssetVolumes[i].AssetID.String() < s.AssetVolumes[j].AssetID.String()
	})
	sort.Slice(s.AssetNamesAndDescriptions, func(i, j int) bool {
		return s.AssetNamesAndDescriptions[i].AssetID.String() < s.AssetNamesAndDescriptions[j].AssetID.String()
	})
	sort.Slice(s.AssetScripts, func(i, j int) bool {
		return s.AssetScripts[i].AssetID.String() < s.AssetScripts[j].AssetID.String()
	})
	sort.Slice(s.Sponsorships, func(i, j int) bool {
		return s.Sponsorships[i].AssetID.String() < s.Sponsorships[j].AssetID.String()
	})
	sort.Slice(s.NewLeases, func(i, j int) bool {
		return s.NewLeases[i].LeaseID.String() < s.NewLeases[j].LeaseID.String()
	})
	sort.Slice(s.CancelledLeases, func(i, j int) bool {
		return s.CancelledLeases[i].LeaseID.String() < s.CancelledLeases[j].LeaseID.String()
	})
	sort.Slice(s.Aliases, func(i, j int) bool {
		idI := s.Aliases[i].Alias + s.Aliases[i].Address.String()
		idJ := s.Aliases[j].Alias + s.Aliases[j].Address.String()
		return idI < idJ
	})
	sort.Slice(s.OrderFills, func(i, j int) bool {
		return s.OrderFills[i].OrderID.String() < s.OrderFills[j].OrderID.String()
	})
	sort.Slice(s.AccountScripts, func(i, j int) bool {
		return s.AccountScripts[i].SenderPublicKey.String() < s.AccountScripts[j].SenderPublicKey.String()
	})
	sortEntries := func(entries proto.DataEntries) {
		sort.Slice(entries, func(i, j int) bool {
			idI := entries[i].GetValueType().String() + entries[i].GetKey()
			idJ := entries[j].GetValueType().String() + entries[j].GetKey()
			return idI < idJ
		})
	}
	for i := range s.AccountData {
		sortEntries(s.AccountData[i].DataEntries)
	}
	sort.Slice(s.AccountData, func(i, j int) bool {
		makeID := func(snap proto.DataEntriesSnapshot) string {
			addr := snap.Address
			entries := snap.DataEntries
			var sb strings.Builder
			sb.WriteString(addr.String())
			for _, e := range entries {
				sb.WriteString(e.GetValueType().String())
				sb.WriteString(e.GetKey())
			}
			return sb.String()
		}
		return makeID(s.AccountData[i]) < makeID(s.AccountData[j])
	})
}

func (s *txSnapshotJSON) diff(
	other txSnapshotJSON,
	firstName, secondNameName string,
	diffContextLines int,
) (string, error) {
	sJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	otherJSON, err := json.MarshalIndent(other, "", "  ")
	if err != nil {
		return "", err
	}
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(sJSON)),
		B:        difflib.SplitLines(string(otherJSON)),
		FromFile: firstName,
		ToFile:   secondNameName,
		Context:  diffContextLines,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return "", err
	}
	return text, nil
}

type blockSnapshotJSON []txSnapshotJSON

func (b blockSnapshotJSON) sortFields() {
	for i := range b {
		b[i].sortFields()
	}
}

type blockIDs struct {
	Transactions []struct {
		ID string `json:"id"`
	} `json:"transactions"`
}

func (b blockIDs) IDs() []string {
	if b.Transactions == nil {
		return nil
	}
	ids := make([]string, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		ids = append(ids, tx.ID)
	}
	return ids
}
