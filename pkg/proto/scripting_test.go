package proto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"testing"

	pb "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
)

func TestScriptResultBinaryRoundTrip(t *testing.T) {
	waves, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	asset0, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	require.NoError(t, err)
	asset1, err := NewOptionalAssetFromString("Ft5X1v1LTa1ABafufpaCWyVj7KkaxUWE6xBhW6sNFJck")
	require.NoError(t, err)
	addr0, err := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	require.NoError(t, err)
	rcp := NewRecipientFromAddress(addr0)

	for i, test := range []ScriptResult{
		{
			DataEntries: []DataEntryScriptAction{
				{&IntegerDataEntry{"some key", 12345}},
				{&BooleanDataEntry{"negative value", false}},
				{&StringDataEntry{"some key", "some value string"}},
				{&BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}}},
				{&IntegerDataEntry{"some key2", -12345}},
				{&BooleanDataEntry{"negative value2", true}},
				{&StringDataEntry{"some key143", "some value2 string"}},
				{&BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
				{&DeleteDataEntry{Key: "xxx"}},
			},
			Transfers: []TransferScriptAction{
				{Amount: math.MaxInt64, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
		{
			DataEntries: []DataEntryScriptAction{
				{&IntegerDataEntry{"some key", 12345}},
			},
		},
		{
			Transfers: []TransferScriptAction{
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
		{
			Issues: []IssueScriptAction{
				{ID: asset0.ID, Name: "xxx1", Description: "some asset", Quantity: 10000000, Decimals: 2, Reissuable: false, Script: nil, Timestamp: 0},
				{ID: asset1.ID, Name: strings.Repeat("x", 100), Description: strings.Repeat("s", 1000), Quantity: math.MaxUint32, Decimals: 0, Reissuable: true, Script: nil, Timestamp: math.MaxInt64},
			},
		},
		{
			Issues: []IssueScriptAction{
				{ID: asset1.ID, Name: "xxx1", Description: "some asset", Quantity: 10000000, Decimals: 2, Reissuable: false, Script: nil, Timestamp: 0},
				{ID: asset0.ID, Name: strings.Repeat("x", 100), Description: strings.Repeat("s", 1000), Quantity: math.MaxUint32, Decimals: 0, Reissuable: true, Script: nil, Timestamp: math.MaxInt64},
			},
			Reissues: []ReissueScriptAction{
				{AssetID: asset0.ID, Quantity: 100000, Reissuable: false},
				{AssetID: asset1.ID, Quantity: 1234567890, Reissuable: true},
			},
		},
		{
			Reissues: []ReissueScriptAction{
				{AssetID: asset0.ID, Quantity: 100000, Reissuable: false},
				{AssetID: asset1.ID, Quantity: 1234567890, Reissuable: true},
			},
			Burns: []BurnScriptAction{
				{AssetID: asset1.ID, Quantity: 12345},
				{AssetID: asset0.ID, Quantity: 0},
			},
		},
	} {
		if msg, err := test.ToProtobuf(); assert.NoError(t, err) {
			if b, err := MarshalDeterministic(msg); assert.NoError(t, err) {
				in := &g.InvokeScriptResult{}
				if err := pb.Unmarshal(b, in); assert.NoError(t, err) {
					sr := ScriptResult{}
					if err := sr.FromProtobuf('W', in); assert.NoError(t, err) {
						ok, msg := compare(test, sr)
						assert.True(t, ok, fmt.Sprintf("#%d: %s", i+1, msg))
					}
				}
			}
		}
	}
}

func compare(a, b ScriptResult) (bool, string) {
	if len(a.DataEntries) != len(b.DataEntries) {
		return false, fmt.Sprintf("Different lenght of DataEntries: %d != %d", len(a.DataEntries), len(b.DataEntries))
	}
	for i := range a.DataEntries {
		ok, msg := compareDataEntries(a.DataEntries[i].Entry, b.DataEntries[i].Entry)
		if !ok {
			return false, fmt.Sprintf("DataEntry #%d: %s", i+1, msg)
		}
	}
	if len(a.Transfers) != len(b.Transfers) {
		return false, fmt.Sprintf("Different lenght of Transfers: %d != %d", len(a.Transfers), len(b.Transfers))
	}
	for i := range a.Transfers {
		ok, msg := compareTransfers(a.Transfers[i], b.Transfers[i])
		if !ok {
			return false, fmt.Sprintf("Transfer #%d: %s", i+1, msg)
		}
	}
	if len(a.Issues) != len(b.Issues) {
		return false, fmt.Sprintf("Different lenght of Issues: %d != %d", len(a.Issues), len(b.Issues))
	}
	for i := range a.Issues {
		ok, msg := compareIssues(a.Issues[i], b.Issues[i])
		if !ok {
			return false, fmt.Sprintf("Issue #%d: %s", i+1, msg)
		}
	}
	if len(a.Reissues) != len(b.Reissues) {
		return false, fmt.Sprintf("Different lenght of Reissues: %d != %d", len(a.Reissues), len(b.Reissues))
	}
	for i := range a.Reissues {
		ok, msg := compareReissues(a.Reissues[i], b.Reissues[i])
		if !ok {
			return false, fmt.Sprintf("Reissue #%d: %s", i+1, msg)
		}
	}
	if len(a.Burns) != len(b.Burns) {
		return false, fmt.Sprintf("Different lenght of Burns: %d != %d", len(a.Burns), len(b.Burns))
	}
	for i := range a.Burns {
		ok, msg := compareBurns(a.Burns[i], b.Burns[i])
		if !ok {
			return false, fmt.Sprintf("Burn #%d: %s", i+1, msg)
		}
	}
	return true, ""
}

func compareDataEntries(a, b DataEntry) (bool, string) {
	if a.GetKey() != b.GetKey() {
		return false, fmt.Sprintf("Different DataEntry key: %s != %s", a.GetKey(), b.GetKey())
	}
	switch ta := a.(type) {
	case *IntegerDataEntry:
		tb, ok := b.(*IntegerDataEntry)
		if !ok {
			return false, fmt.Sprintf("Different DataEntry types: %T != %T", ta, tb)
		}
		if ta.Value != tb.Value {
			return false, fmt.Sprintf("Different values: %d != %d", int(ta.Value), int(tb.Value))
		}
	case *BooleanDataEntry:
		tb, ok := b.(*BooleanDataEntry)
		if !ok {
			return false, fmt.Sprintf("Different DataEntry types: %T != %T", ta, tb)
		}
		if ta.Value != tb.Value {
			return false, fmt.Sprintf("Different values: %v != %v", ta.Value, tb.Value)
		}
	case *StringDataEntry:
		tb, ok := b.(*StringDataEntry)
		if !ok {
			return false, fmt.Sprintf("Different DataEntry types: %T != %T", ta, tb)
		}
		if ta.Value != tb.Value {
			return false, fmt.Sprintf("Different values: %s != %s", ta.Value, tb.Value)
		}
	case *BinaryDataEntry:
		tb, ok := b.(*BinaryDataEntry)
		if !ok {
			return false, fmt.Sprintf("Different DataEntry types: %T != %T", ta, tb)
		}
		if !bytes.Equal(ta.Value, tb.Value) {
			return false, fmt.Sprintf("Different values: %s != %s", hex.EncodeToString(ta.Value), hex.EncodeToString(tb.Value))
		}
	}
	return true, ""
}

func compareTransfers(a, b TransferScriptAction) (bool, string) {
	if a.Amount != b.Amount {
		return false, fmt.Sprintf("Different amounts: %d != %d", int(a.Amount), int(b.Amount))
	}
	if !a.Recipient.Eq(b.Recipient) {
		return false, fmt.Sprintf("Different recipients: %s != %s", a.Recipient.String(), b.Recipient.String())
	}
	if !a.Asset.Eq(b.Asset) {
		return false, fmt.Sprintf("Different assets: %s != %s", a.Asset.String(), b.Asset.String())
	}
	return true, ""
}

func compareIssues(a, b IssueScriptAction) (bool, string) {
	if a.ID != b.ID {
		return false, fmt.Sprintf("Different IDs: %s != %s", a.ID.String(), b.ID.String())
	}
	if !bytes.Equal(a.Script, b.Script) {
		return false, fmt.Sprintf("Different scripts: %s != %s", hex.EncodeToString(a.Script), hex.EncodeToString(b.Script))
	}
	if a.Reissuable != b.Reissuable {
		return false, fmt.Sprintf("Different reissuables: %v != %v", a.Reissuable, b.Reissuable)
	}
	if a.Decimals != b.Decimals {
		return false, fmt.Sprintf("Different decimals: %d != %d", int(a.Decimals), int(b.Decimals))
	}
	if a.Description != b.Description {
		return false, fmt.Sprintf("Different descriptions: %s != %s", a.Description, b.Description)
	}
	if a.Name != b.Name {
		return false, fmt.Sprintf("Different namess: %s != %s", a.Name, b.Name)
	}
	if a.Quantity != b.Quantity {
		return false, fmt.Sprintf("Different quantities: %d != %d", int(a.Quantity), int(b.Quantity))
	}
	if a.Timestamp != b.Timestamp {
		return false, fmt.Sprintf("Different timestamps: %d != %d", int(a.Timestamp), int(b.Timestamp))
	}
	return true, ""
}

func compareReissues(a, b ReissueScriptAction) (bool, string) {
	if a.AssetID != b.AssetID {
		return false, fmt.Sprintf("Different assets IDs: %s != %s", a.AssetID.String(), b.AssetID.String())
	}
	if a.Quantity != b.Quantity {
		return false, fmt.Sprintf("Different quantities: %d != %d", int(a.Quantity), int(b.Quantity))
	}
	if a.Reissuable != b.Reissuable {
		return false, fmt.Sprintf("Different reissuables: %v != %v", a.Reissuable, b.Reissuable)
	}
	return true, ""
}

func compareBurns(a, b BurnScriptAction) (bool, string) {
	if a.AssetID != b.AssetID {
		return false, fmt.Sprintf("Different assets IDs: %s != %s", a.AssetID.String(), b.AssetID.String())
	}
	if a.Quantity != b.Quantity {
		return false, fmt.Sprintf("Different quantities: %d != %d", int(a.Quantity), int(b.Quantity))
	}
	return true, ""
}
