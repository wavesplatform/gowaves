package internal

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/mr-tron/base58"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func ExtractDataEntries(res *waves.InvokeScriptResult) []DataEntry {
	r := make([]DataEntry, len(res.GetData()))
	for i, e := range res.GetData() {
		r[i] = DataEntry{
			Key:   e.GetKey(),
			Value: extractValue(e),
		}
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractDataEntries(inv.GetStateChanges())...)
		}
	}
	sort.Sort(dataEntrySorter(r))
	return r
}

func ExtractTransfers(res *waves.InvokeScriptResult, scheme byte) []Transfer {
	r := make([]Transfer, len(res.GetTransfers()))
	for i, e := range res.GetTransfers() {
		r[i] = extractTransfer(e, scheme)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractTransfers(inv.GetStateChanges(), scheme)...)
		}
	}
	sort.Sort(transferSorter(r))
	return r
}

func ExtractIssues(res *waves.InvokeScriptResult) []Issue {
	r := make([]Issue, len(res.GetIssues()))
	for i, e := range res.GetIssues() {
		r[i] = extractIssue(e)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractIssues(inv.GetStateChanges())...)
		}
	}
	sort.Sort(issueSorter(r))
	return r
}

func ExtractReissues(res *waves.InvokeScriptResult) []Reissue {
	r := make([]Reissue, len(res.GetReissues()))
	for i, e := range res.GetReissues() {
		r[i] = extractReissue(e)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractReissues(inv.GetStateChanges())...)
		}
	}
	sort.Sort(reissueSorter(r))
	return r
}

func ExtractBurns(res *waves.InvokeScriptResult) []Burn {
	r := make([]Burn, len(res.GetBurns()))
	for i, e := range res.GetBurns() {
		r[i] = extractBurn(e)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractBurns(inv.GetStateChanges())...)
		}
	}
	sort.Sort(burnSorter(r))
	return r
}

func ExtractSponsorships(res *waves.InvokeScriptResult) []Sponsorship {
	r := make([]Sponsorship, len(res.GetSponsorFees()))
	for i, e := range res.GetSponsorFees() {
		r[i] = extractSponsorship(e)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractSponsorships(inv.GetStateChanges())...)
		}
	}
	sort.Sort(sponsorshipSorter(r))
	return r
}

func ExtractLeases(res *waves.InvokeScriptResult, scheme byte) []Lease {
	r := make([]Lease, len(res.GetLeases()))
	for i, e := range res.GetLeases() {
		r[i] = extractLease(e, scheme)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractLeases(inv.GetStateChanges(), scheme)...)
		}
	}
	sort.Sort(leaseSorter(r))
	return r
}

func ExtractLeaseCancels(res *waves.InvokeScriptResult) []LeaseCancel {
	r := make([]LeaseCancel, len(res.GetLeaseCancels()))
	for i, e := range res.GetLeaseCancels() {
		r[i] = extractLeaseCancel(e)
	}
	if len(res.GetInvokes()) > 0 {
		for _, inv := range res.GetInvokes() {
			r = append(r, ExtractLeaseCancels(inv.GetStateChanges())...)
		}
	}
	sort.Sort(leaseCancelSorter(r))
	return r
}

func extractValue(e *waves.DataTransactionData_DataEntry) string {
	switch v := e.GetValue().(type) {
	case *waves.DataTransactionData_DataEntry_BinaryValue:
		return base58.Encode(v.BinaryValue)
	case *waves.DataTransactionData_DataEntry_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	case *waves.DataTransactionData_DataEntry_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *waves.DataTransactionData_DataEntry_StringValue:
		return v.StringValue
	default:
		return fmt.Sprintf("unsupported value type %T", e.GetValue())
	}
}

func extractAddress(b []byte, scheme byte) string {
	a, err := proto.RebuildAddress(scheme, b)
	if err != nil {
		return fmt.Sprintf("Invalid address '%s'", base58.Encode(b))
	}
	return a.String()
}

func extractAssetID(b []byte) string {
	if len(b) == 0 {
		return "WAVES"
	}
	return base58.Encode(b)
}

func extractTransfer(e *waves.InvokeScriptResult_Payment, scheme byte) Transfer {
	return Transfer{
		Address: extractAddress(e.GetAddress(), scheme),
		AssetID: extractAssetID(e.GetAmount().GetAssetId()),
		Amount:  int(e.GetAmount().GetAmount()),
	}
}

func extractIssue(e *waves.InvokeScriptResult_Issue) Issue {
	return Issue{
		AssetID:     extractAssetID(e.GetAssetId()),
		Name:        e.GetName(),
		Description: e.GetDescription(),
		Amount:      int(e.GetAmount()),
		Decimals:    int(e.GetDecimals()),
		Reissuable:  e.GetReissuable(),
		Script:      base64.StdEncoding.EncodeToString(e.GetScript()),
		Nonce:       int(e.GetNonce()),
	}
}

func extractReissue(e *waves.InvokeScriptResult_Reissue) Reissue {
	return Reissue{
		AssetID:    extractAssetID(e.GetAssetId()),
		Amount:     int(e.GetAmount()),
		Reissuable: e.GetIsReissuable(),
	}
}

func extractBurn(e *waves.InvokeScriptResult_Burn) Burn {
	return Burn{
		AssetID: extractAssetID(e.GetAssetId()),
		Amount:  int(e.GetAmount()),
	}
}

func extractSponsorship(e *waves.InvokeScriptResult_SponsorFee) Sponsorship {
	return Sponsorship{
		AssetID: extractAssetID(e.GetMinFee().GetAssetId()),
		MinFee:  int(e.GetMinFee().GetAmount()),
	}
}

func extractRecipient(e *waves.Recipient, scheme byte) string {
	switch tr := e.GetRecipient().(type) {
	case *waves.Recipient_Alias:
		return tr.Alias
	case *waves.Recipient_PublicKeyHash:
		a, err := proto.RebuildAddress(scheme, tr.PublicKeyHash)
		if err != nil {
			return fmt.Sprintf("invalid address '%s'", base58.Encode(tr.PublicKeyHash))
		}
		return a.String()
	default:
		return fmt.Sprintf("unsupported recipient type %T", e)
	}
}

func extractLease(e *waves.InvokeScriptResult_Lease, scheme byte) Lease {
	return Lease{
		LeaseID:   base58.Encode(e.GetLeaseId()),
		Recipient: extractRecipient(e.GetRecipient(), scheme),
		Amount:    int(e.GetAmount()),
		Nonce:     int(e.GetNonce()),
	}
}

func extractLeaseCancel(e *waves.InvokeScriptResult_LeaseCancel) LeaseCancel {
	return LeaseCancel{LeaseID: base58.Encode(e.GetLeaseId())}
}
