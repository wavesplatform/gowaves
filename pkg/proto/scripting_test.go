package proto

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	pb "google.golang.org/protobuf/proto"
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
	emptySponsorships := make([]*SponsorshipScriptAction, 0)
	emptyLeases := make([]*LeaseScriptAction, 0)
	emptyLeaseCancels := make([]*LeaseCancelScriptAction, 0)
	for i, test := range []ScriptResult{
		{
			DataEntries: []*DataEntryScriptAction{
				{Entry: &IntegerDataEntry{"some key", 12345}},
				{Entry: &BooleanDataEntry{"negative value", false}},
				{Entry: &StringDataEntry{"some key", "some value string"}},
				{Entry: &BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}}},
				{Entry: &IntegerDataEntry{"some key2", -12345}},
				{Entry: &BooleanDataEntry{"negative value2", true}},
				{Entry: &StringDataEntry{"some key143", "some value2 string"}},
				{Entry: &BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
				{Entry: &DeleteDataEntry{Key: "xxx"}},
			},
			Transfers: []*TransferScriptAction{
				{Amount: math.MaxInt64, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
			Issues:       emptyIssues,
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries: []*DataEntryScriptAction{
				{Entry: &IntegerDataEntry{"some key", 12345}},
			},
			Transfers:    emptyTransfers,
			Issues:       emptyIssues,
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers: []*TransferScriptAction{
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: 10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
			Issues:       emptyIssues,
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers:   emptyTransfers,
			Issues: []*IssueScriptAction{
				{ID: asset0.ID, Name: "xxx1", Description: "some asset", Quantity: 10000000, Decimals: 2, Reissuable: false, Script: Script{}, Nonce: 0},
				{ID: asset1.ID, Name: strings.Repeat("x", 100), Description: strings.Repeat("s", 1000), Quantity: math.MaxUint32, Decimals: 0, Reissuable: true, Script: Script{}, Nonce: math.MaxInt64},
			},
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
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
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
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
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries: emptyDataEntries,
			Transfers:   emptyTransfers,
			Issues:      emptyIssues,
			Reissues:    emptyReissues,
			Burns:       emptyBurns,
			Sponsorships: []*SponsorshipScriptAction{
				{AssetID: asset0.ID, MinFee: 12345},
				{AssetID: asset1.ID, MinFee: 0},
			},
			Leases:       emptyLeases,
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries:  emptyDataEntries,
			Transfers:    emptyTransfers,
			Issues:       emptyIssues,
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases: []*LeaseScriptAction{
				{
					ID:        asset0.ID,
					Recipient: rcp,
					Amount:    12345,
					Nonce:     67890,
				},
				{
					ID:        asset1.ID,
					Recipient: rcp,
					Amount:    0,
					Nonce:     0,
				},
			},
			LeaseCancels: emptyLeaseCancels,
		},
		{
			DataEntries:  emptyDataEntries,
			Transfers:    emptyTransfers,
			Issues:       emptyIssues,
			Reissues:     emptyReissues,
			Burns:        emptyBurns,
			Sponsorships: emptySponsorships,
			Leases:       emptyLeases,
			LeaseCancels: []*LeaseCancelScriptAction{
				{LeaseID: asset0.ID},
				{LeaseID: asset1.ID},
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
	pk0, err := crypto.NewPublicKeyFromBase58("FPqvNYPoqbkwvsyoNSiYU4xeU2tFCRe6AjsHGRNT2VWn")
	require.NoError(t, err)
	addr0, err := NewAddressFromPublicKey(TestNetScheme, pk0)
	require.NoError(t, err)
	rcp0 := NewRecipientFromAddress(addr0)
	addr1, err := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	require.NoError(t, err)
	generateActions := func(dataEntries, payments, transferGroup, issueGroup byte) []ScriptAction {
		actions := make([]ScriptAction, 0, dataEntries+payments+transferGroup+issueGroup)
		for i := byte(0); i < dataEntries; i++ {
			action := &DataEntryScriptAction{
				Entry: &IntegerDataEntry{Key: fmt.Sprintf("data entry #%d", i), Value: int64(i) + 1},
			}
			actions = append(actions, action)
		}
		for i := byte(0); i < payments; i++ {
			action := &AttachedPaymentScriptAction{
				Sender:    &pk0,
				Recipient: NewRecipientFromAddress(addr1),
				Amount:    int64(i) + 100,
				Asset:     NewOptionalAssetWaves(),
			}
			actions = append(actions, action)
		}
		for i := byte(0); i < transferGroup; i++ {
			action := &TransferScriptAction{Recipient: rcp0, Amount: 100, Asset: NewOptionalAssetWaves()}
			actions = append(actions, action)
		}
		for i := byte(0); i < issueGroup; i++ {
			action := &IssueScriptAction{
				ID:          crypto.Digest{i},
				Name:        fmt.Sprintf("xxx#%d", i),
				Description: fmt.Sprintf("some asset #%d", i),
				Quantity:    int64(i) + 10000000,
				Decimals:    2,
				Reissuable:  false,
				Script:      Script{},
				Nonce:       int64(i),
			}
			actions = append(actions, action)
		}
		return actions
	}
	tests := []struct {
		actions           []ScriptAction
		restrictions      ActionsValidationRestrictions
		isRideV6Activated bool
		libVersion        ast.LibraryVersion
		valid             bool
	}{
		{
			actions: []ScriptAction{
				&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
				&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
				&DataEntryScriptAction{Entry: &StringDataEntry{"some key143", "some value2 string"}},
				&DataEntryScriptAction{Entry: &BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
				&DataEntryScriptAction{Entry: &DeleteDataEntry{Key: "xxx"}},
				&TransferScriptAction{Recipient: rcp0, Amount: 100, Asset: OptionalAsset{}},
			},
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV1},
			isRideV6Activated: false,
			libVersion:        ast.LibV5,
			valid:             true,
		},
		{
			actions: []ScriptAction{
				&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
				&TransferScriptAction{Recipient: rcp0, Amount: -100, Asset: OptionalAsset{}},
				&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
			},
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV1},
			isRideV6Activated: false,
			libVersion:        ast.LibV5,
			valid:             false,
		},
		{
			actions: []ScriptAction{
				&DataEntryScriptAction{Entry: &IntegerDataEntry{"some key2", -12345}},
				&DataEntryScriptAction{Entry: &BooleanDataEntry{"negative value2", true}},
				&DataEntryScriptAction{Entry: &StringDataEntry{"some key143", "some value2 string"}},
				&DataEntryScriptAction{Entry: &BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
				&DataEntryScriptAction{Entry: &DeleteDataEntry{Key: "xxx"}},
				&TransferScriptAction{Recipient: rcp0, Amount: 100, Asset: OptionalAsset{}},
			},
			restrictions: ActionsValidationRestrictions{
				DisableSelfTransfers: true,
				ScriptAddress:        addr0,
				MaxDataEntriesSize:   MaxDataEntriesScriptActionsSizeInBytesV1,
			},
			isRideV6Activated: false,
			libVersion:        ast.LibV5,
			valid:             false,
		},
		{
			actions: []ScriptAction{
				&LeaseScriptAction{Recipient: rcp0, Amount: 100},
			},
			restrictions:      ActionsValidationRestrictions{ScriptAddress: addr0},
			isRideV6Activated: false,
			libVersion:        ast.LibV5,
			valid:             false,
		},
		{
			actions: []ScriptAction{
				&LeaseScriptAction{Recipient: rcp0, Amount: 0},
				&LeaseScriptAction{Recipient: rcp0, Amount: -100},
			},
			restrictions:      ActionsValidationRestrictions{},
			isRideV6Activated: false,
			libVersion:        ast.LibV5,
			valid:             false,
		},
		{
			actions: []ScriptAction{
				&DataEntryScriptAction{
					Entry: &BinaryDataEntry{"this first key contains 32 bytes", []byte("this first value contains 34 bytes")},
				},
				&DataEntryScriptAction{
					Entry: &BinaryDataEntry{"this second key contains 33 bytes", []byte("this second value contains 35 bytes")},
				},
			},
			restrictions: ActionsValidationRestrictions{
				MaxDataEntriesSize: 32 + 34 + 33 + 35,
			},
			isRideV6Activated: true,
			libVersion:        ast.LibV5,
			valid:             true,
		},
		{
			actions: []ScriptAction{
				&DataEntryScriptAction{
					Entry: &BinaryDataEntry{"this first key contains 32 bytes", []byte("this first value contains 34 bytes")},
				},
				&DataEntryScriptAction{
					Entry: &BinaryDataEntry{"this second key contains 33 bytes", []byte("this second value contains 35 bytes")},
				},
			},
			restrictions: ActionsValidationRestrictions{
				MaxDataEntriesSize: 32 + 34 + 33 + 35 - 1,
			},
			isRideV6Activated: true,
			libVersion:        ast.LibV5,
			valid:             false,
		},
		{
			actions:           generateActions(100, 10, 5, 5),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV4,
			valid:             true,
		},
		{
			actions:           generateActions(101, 10, 5, 5),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV4,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 5, 5),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV4,
			valid:             true,
		},
		{
			actions:           generateActions(10, 10, 6, 5),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV4,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 5, 6),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV4,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 5, 25),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        ast.LibV5,
			valid:             true,
		},
		{
			actions:           generateActions(10, 10, 6, 25),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        5,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 25, 6),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        5,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 100, 30),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        6,
			valid:             true,
		},
		{
			actions:           generateActions(10, 10, 101, 30),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        6,
			valid:             false,
		},
		{
			actions:           generateActions(10, 10, 100, 31),
			restrictions:      ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV2},
			isRideV6Activated: true,
			libVersion:        6,
			valid:             false,
		},
	}
	for i, test := range tests {
		err := ValidateActions(test.actions, test.restrictions, test.isRideV6Activated, test.libVersion, true)
		if test.valid {
			require.NoError(t, err, "#%d", i)
		} else {
			require.Error(t, err, "#%d", i)
		}
	}
}

func TestGenerateLeaseScriptActionID(t *testing.T) {
	for _, test := range []struct {
		recipient Recipient
		amount    int64
		nonce     int64
		tx        crypto.Digest
		id        string
	}{
		{mustRecipientFromString("3Me8JF8fhugSSa2Kx4w7v2tX377sTVtKSU5"), 100000000, 0, crypto.MustDigestFromBase58("3JGcEMaASHc7zcJwpkuFTU3WScKtMU6KDQ5KFr53GQhV"), "HrvHDiegqPhcoKamTeTsNUcQiFot8D1KqyBirsEuCMG9"},
		{mustRecipientFromString("3Me8JF8fhugSSa2Kx4w7v2tX377sTVtKSU5"), 100000000, 0, crypto.MustDigestFromBase58("45R9UJrmCmZu1HtofbHyEmaFr2r1u5xXThGmESszVuFV"), "28yGDS82NrYBC1B4XTVYbwWpJyW7JPYTX7UtVQd1Prkw"},
		{mustRecipientFromString("3Me8JF8fhugSSa2Kx4w7v2tX377sTVtKSU5"), 50000000, 0, crypto.MustDigestFromBase58("45R9UJrmCmZu1HtofbHyEmaFr2r1u5xXThGmESszVuFV"), "GmqQBZPPAHb1u7mQJ8vVp89mcaii23jAyrbDfqYiGo6U"},
		{mustRecipientFromString("3Me8JF8fhugSSa2Kx4w7v2tX377sTVtKSU5"), 100000000, 0, crypto.MustDigestFromBase58("FBmMUrQ5GXun9LrGtHPcJYWSkkfToMReux14iSb2zf4c"), "5PmSmWMmCGh7zjf8SgvzmrUZrEKVeNL2wK12p7Y3Rezi"},
		{mustRecipientFromString("3Me8JF8fhugSSa2Kx4w7v2tX377sTVtKSU5"), 50000000, 0, crypto.MustDigestFromBase58("FBmMUrQ5GXun9LrGtHPcJYWSkkfToMReux14iSb2zf4c"), "2EgitLRfQmYckjmi16b2h3YFLBz7yKS877tb1TQRXR6Y"},
	} {
		id := GenerateLeaseScriptActionID(test.recipient, test.amount, test.nonce, test.tx)
		assert.Equal(t, test.id, id.String())
	}
}

// This function is for tests only! Could produce invalid recipient.
func mustRecipientFromString(s string) Recipient {
	r, err := recipientFromString(s)
	if err != nil {
		panic(err)
	}
	return r
}

func TestAssetIDGeneration(t *testing.T) {
	for _, test := range []struct {
		name        string
		description string
		decimals    int64
		quantity    int64
		reissuable  bool
		nonce       int64
		txID        string
		assetID     string
	}{
		{"DUCK-AAAAAAAA-GB", "{\"genotype\": \"DUCK-AAAAAAAA-GB\", \"crossbreeding\": true}", 0, 1, false, 2578353, "BBcyb47NB9cbGKXNPakxKGxmdABLzhxRNsztd9hTad6", "4JzEW8LnTXuZ117iqdFXjuNBx3GG5mUvmZhnZ8V3yty7"},
		{"DUCK-AAAAAAAA-GB", "{\"genotype\": \"DUCK-AAAAAAAA-GB\", \"crossbreeding\": true}", 0, 1, false, 2578353, "BBcyb47NB9cbGKXNPakxKGxmdABLzhxRNsztd9hTad6", "4JzEW8LnTXuZ117iqdFXjuNBx3GG5mUvmZhnZ8V3yty7"},
		{"DUCK-BBBBBBBB-GR", "{\"genotype\": \"DUCK-BBBBBBBB-GR\", \"crossbreeding\": true}", 0, 1, false, 2578301, "AA33kjey1MbsQY29NB9Fy9mMcX2oFaxapsnpBk5sVhxU", "7tuYcoFnBLub562Ddsqb3s7iM9edjA9jn6zePvffLV9j"},
	} {
		txID := crypto.MustDigestFromBase58(test.txID)
		assetID := GenerateIssueScriptActionID(test.name, test.description, test.decimals, test.quantity, test.reissuable, test.nonce, txID)
		assert.Equal(t, test.assetID, assetID.String())
	}
}

func TestNegativePaymentsValidation(t *testing.T) {
	pk1, err := crypto.NewPublicKeyFromBase58("FPqvNYPoqbkwvsyoNSiYU4xeU2tFCRe6AjsHGRNT2VWn")
	require.NoError(t, err)
	a2, err := NewAddressFromString("3PMj3yGPBEa1Sx9X4TSBFeJCMMaE3wvKR4N")
	require.NoError(t, err)
	rcp := NewRecipientFromAddress(a2)

	restrictions := ActionsValidationRestrictions{MaxDataEntriesSize: MaxDataEntriesScriptActionsSizeInBytesV1}
	for i, test := range []struct {
		actions          []ScriptAction
		validatePayments bool
		error            bool
	}{
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, 0, NewOptionalAssetWaves()}}, true, false},
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, 1000, NewOptionalAssetWaves()}}, true, false},
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, -1000, NewOptionalAssetWaves()}}, true, true},
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, 0, NewOptionalAssetWaves()}}, false, false},
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, 1000, NewOptionalAssetWaves()}}, false, false},
		{[]ScriptAction{&AttachedPaymentScriptAction{&pk1, rcp, -1000, NewOptionalAssetWaves()}}, false, false},
	} {
		err := ValidateActions(test.actions, restrictions, false, 5, test.validatePayments)
		if !test.error {
			require.NoError(t, err, fmt.Sprintf("#%d", i))
		} else {
			require.Error(t, err, fmt.Sprintf("#%d", i))
		}
	}
}
