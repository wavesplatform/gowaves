package proto

import (
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
	emptyDataEntries := make([]*DataEntryScriptAction, 0)
	emptyTransfers := make([]*TransferScriptAction, 0)
	emptyIssues := make([]*IssueScriptAction, 0)
	emptyReissues := make([]*ReissueScriptAction, 0)
	emptyBurns := make([]*BurnScriptAction, 0)
	for i, test := range []ScriptResult{
		{
			DataEntries: []*DataEntryScriptAction{
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
			Transfers: []*TransferScriptAction{
				{Amount: math.MaxInt64, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
			Issues:   emptyIssues,
			Reissues: emptyReissues,
			Burns:    emptyBurns,
		},
		{
			DataEntries: []*DataEntryScriptAction{
				{&IntegerDataEntry{"some key", 12345}},
			},
			Transfers: emptyTransfers,
			Issues:    emptyIssues,
			Reissues:  emptyReissues,
			Burns:     emptyBurns,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers: []*TransferScriptAction{
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
			Issues:   emptyIssues,
			Reissues: emptyReissues,
			Burns:    emptyBurns,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers:   emptyTransfers,
			Issues: []*IssueScriptAction{
				{ID: asset0.ID, Name: "xxx1", Description: "some asset", Quantity: 10000000, Decimals: 2, Reissuable: false, Script: Script{}, Nonce: 0},
				{ID: asset1.ID, Name: strings.Repeat("x", 100), Description: strings.Repeat("s", 1000), Quantity: math.MaxUint32, Decimals: 0, Reissuable: true, Script: Script{}, Nonce: math.MaxInt64},
			},
			Reissues: emptyReissues,
			Burns:    emptyBurns,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers:   emptyTransfers,
			Issues: []*IssueScriptAction{
				{ID: asset1.ID, Name: "xxx1", Description: "some asset", Quantity: 10000000, Decimals: 2, Reissuable: false, Script: Script{}, Nonce: 0},
				{ID: asset0.ID, Name: strings.Repeat("x", 100), Description: strings.Repeat("s", 1000), Quantity: math.MaxUint32, Decimals: 0, Reissuable: true, Script: Script{}, Nonce: math.MaxInt64},
			},
			Reissues: []*ReissueScriptAction{
				{AssetID: asset0.ID, Quantity: 100000, Reissuable: false},
				{AssetID: asset1.ID, Quantity: 1234567890, Reissuable: true},
			},
			Burns: emptyBurns,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers:   emptyTransfers,
			Issues:      emptyIssues,
			Reissues: []*ReissueScriptAction{
				{AssetID: asset0.ID, Quantity: 100000, Reissuable: false},
				{AssetID: asset1.ID, Quantity: 1234567890, Reissuable: true},
			},
			Burns: []*BurnScriptAction{
				{AssetID: asset1.ID, Quantity: 12345},
				{AssetID: asset0.ID, Quantity: 0},
			},
		},
	} {
		if msg, err := test.ToProtobuf(); assert.NoError(t, err) {
			if b, err := MarshalToProtobufDeterministic(msg); assert.NoError(t, err) {
				in := &g.InvokeScriptResult{}
				if err := pb.Unmarshal(b, in); assert.NoError(t, err) {
					sr := ScriptResult{}
					if err := sr.FromProtobuf('W', in); assert.NoError(t, err) {
						assert.EqualValues(t, test, sr, fmt.Sprintf("#%d", i+1))
					}
				}
			}
		}
	}
}

func TestActionsValidation(t *testing.T) {
	addr0, err := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	require.NoError(t, err)
	rcp0 := NewRecipientFromAddress(addr0)
	for i, test := range []struct {
		actions      []ScriptAction
		restrictions ActionsValidationRestrictions
		valid        bool
	}{
		{actions: []ScriptAction{
			&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
			&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
			&DataEntryScriptAction{Entry: &StringDataEntry{"some key143", "some value2 string"}},
			&DataEntryScriptAction{Entry: &BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
			&DataEntryScriptAction{Entry: &DeleteDataEntry{Key: "xxx"}},
			&TransferScriptAction{Recipient: rcp0, Amount: 100, Asset: OptionalAsset{}},
		}, restrictions: ActionsValidationRestrictions{}, valid: true},
		{actions: []ScriptAction{
			&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
			&TransferScriptAction{Recipient: rcp0, Amount: -100, Asset: OptionalAsset{}},
			&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
		}, restrictions: ActionsValidationRestrictions{}, valid: false},
		{actions: []ScriptAction{
			&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
			&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
			&DataEntryScriptAction{Entry: &StringDataEntry{"some key143", "some value2 string"}},
			&DataEntryScriptAction{Entry: &BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
			&DataEntryScriptAction{Entry: &DeleteDataEntry{Key: "xxx"}},
			&TransferScriptAction{Recipient: rcp0, Amount: 100, Asset: OptionalAsset{}},
		}, restrictions: ActionsValidationRestrictions{DisableSelfTransfers: true, ScriptAddress: addr0}, valid: false},
	} {
		err := ValidateActions(test.actions, test.restrictions)
		if test.valid {
			require.NoError(t, err, fmt.Sprintf("#%d", i))
		} else {
			require.Error(t, err, fmt.Sprintf("#%d", i))
		}
	}
}
