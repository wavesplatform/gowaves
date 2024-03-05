package proto_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func Test_txSnapshotJSON_MarshalJSON_UnmarshalJSON(t *testing.T) {
	const js = `
[
  {
    "applicationStatus": "succeeded",
    "balances": [
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "asset": null,
        "balance": 49315021748316
      },
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "asset": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "balance": 100000000000
      },
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "asset": "4eKGReLgtQgbsKLaeGeSbpKwJJH2wruK66kW6QDQdULP",
        "balance": 100000000000
      }
    ],
    "leaseBalances": [
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "in": 567758,
        "out": 121342
      }
    ],
    "assetStatics": [
      {
        "id": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "issuer": "9KFDEPnavEUzmiYbQw81VC4Niu526mjECQUnn8wrVW4Q",
        "decimals": 4,
        "nft": false
      }
    ],
    "assetVolumes": [
      {
        "id": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "volume": 100000000000,
        "isReissuable": true
      }
    ],
    "assetNamesAndDescriptions": [
      {
        "id": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "name": "foo",
        "description": "bar"
      }
    ],
    "assetScripts": [
      {
        "id": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "script": "base64:AQIDBA=="
      }
    ],
    "sponsorships": [
      {
        "id": "2RnTdCBXEebomosHRLNAXieqscfjwGeyFA9j44CEXCX9",
        "minSponsoredAssetFee": 100000
      },
      {
        "id": "4eKGReLgtQgbsKLaeGeSbpKwJJH2wruK66kW6QDQdULP",
        "minSponsoredAssetFee": 100100
      }
    ],
    "newLeases": [
      {
        "id": "3py1rKXV2HcdBwPUgGwME9Yqq2jBHFCzH58mPh8eGQto",
        "amount": 456465,
        "sender": "9KFDEPnavEUzmiYbQw81VC4Niu526mjECQUnn8wrVW4Q",
        "recipient": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe"
      }
    ],
    "cancelledLeases": [
      {
        "id": "3py1rKXV2HcdBwPUgGwME9Yqq2jBHFCzH58mPh8eGQto"
      }
    ],
    "aliases": [
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "alias": "foobar"
      },
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "alias": "barfoo"
      }
    ],
    "orderFills": [
      {
        "id": "5hvJCSw7m4M2PsQyVSqdz6A5wBVDfeU423eiZZoJM2JK",
        "volume": 500,
        "fee": 100
      }
    ],
    "accountScripts": [
      {
        "publicKey": "9KFDEPnavEUzmiYbQw81VC4Niu526mjECQUnn8wrVW4Q",
        "script": "base64:AQIDBA==",
        "verifierComplexity": 199
      }
    ],
    "accountData": [
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "data": [
          {
            "key": "key1",
            "type": "boolean",
            "value": true
          },
          {
            "key": "key2",
            "type": "integer",
            "value": 21
          },
          {
            "key": "key22",
            "type": "integer",
            "value": 42
          },
          {
            "key": "key3",
            "type": "binary",
            "value": "base64:QSJA6g=="
          },
          {
            "key": "key4",
            "type": "string",
            "value": "foobar"
          },
          {
            "key": "key5",
            "value": null
          }
        ]
      }
    ]
  },
  {
    "applicationStatus": "failed",
    "balances": [
      {
        "address": "3NA26AC1aLjj6uYnuoTahauhUPPPB3VBPUe",
        "asset": null,
        "balance": 49315001748316
      }
    ],
    "leaseBalances": [],
    "assetStatics": [],
    "assetVolumes": [],
    "assetNamesAndDescriptions": [],
    "assetScripts": [],
    "sponsorships": [],
    "newLeases": [],
    "cancelledLeases": [],
    "aliases": [],
    "orderFills": [],
    "accountScripts": [],
    "accountData": []
  },
  {
    "applicationStatus": "elided",
    "balances": [],
    "leaseBalances": [],
    "assetStatics": [],
    "assetVolumes": [],
    "assetNamesAndDescriptions": [],
    "assetScripts": [],
    "sponsorships": [],
    "newLeases": [],
    "cancelledLeases": [],
    "aliases": [],
    "orderFills": [],
    "accountScripts": [],
    "accountData": []
  }
]`

	pk, err := crypto.NewPublicKeyFromBase58("9KFDEPnavEUzmiYbQw81VC4Niu526mjECQUnn8wrVW4Q")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, pk)
	require.NoError(t, err)
	asset1 := crypto.Digest{21, 53, 6, 236}
	asset2 := crypto.Digest{54, 34, 52, 63}
	leaseID := crypto.Digest{42, 1, 2, 3}
	orderID := crypto.Digest{69, 234, 62, 28, 91, 45}

	succeededTxSnap := []proto.AtomicSnapshot{
		&proto.TransactionStatusSnapshot{
			Status: proto.TransactionSucceeded,
		},
		&proto.WavesBalanceSnapshot{
			Address: addr,
			Balance: 49315021748316,
		},
		&proto.AssetBalanceSnapshot{
			Address: addr,
			AssetID: asset1,
			Balance: 100000000000,
		},
		&proto.AssetBalanceSnapshot{
			Address: addr,
			AssetID: asset2,
			Balance: 100000000000,
		},
		&proto.DataEntriesSnapshot{
			Address: addr,
			DataEntries: proto.DataEntries{
				&proto.BooleanDataEntry{
					Key:   "key1",
					Value: true,
				},
				&proto.IntegerDataEntry{
					Key:   "key2",
					Value: 21,
				},
				&proto.IntegerDataEntry{
					Key:   "key22",
					Value: 42,
				},
				&proto.BinaryDataEntry{
					Key:   "key3",
					Value: []byte{65, 34, 64, 234},
				},
				&proto.StringDataEntry{
					Key:   "key4",
					Value: "foobar",
				},
				&proto.DeleteDataEntry{
					Key: "key5",
				},
			},
		},
		&proto.AccountScriptSnapshot{
			SenderPublicKey:    pk,
			Script:             proto.Script{1, 2, 3, 4},
			VerifierComplexity: 199,
		},
		&proto.AssetScriptSnapshot{
			AssetID: asset1,
			Script:  proto.Script{1, 2, 3, 4},
		},
		&proto.LeaseBalanceSnapshot{
			Address:  addr,
			LeaseIn:  567758,
			LeaseOut: 121342,
		},
		&proto.NewLeaseSnapshot{
			LeaseID:       leaseID,
			Amount:        456465,
			SenderPK:      pk,
			RecipientAddr: addr,
		},
		&proto.CancelledLeaseSnapshot{
			LeaseID: leaseID,
		},
		&proto.SponsorshipSnapshot{
			AssetID:         asset1,
			MinSponsoredFee: 100000,
		},
		&proto.SponsorshipSnapshot{
			AssetID:         asset2,
			MinSponsoredFee: 100100,
		},
		&proto.AliasSnapshot{
			Address: addr,
			Alias:   "foobar",
		},
		&proto.AliasSnapshot{
			Address: addr,
			Alias:   "barfoo",
		},
		&proto.FilledVolumeFeeSnapshot{
			OrderID:      orderID,
			FilledVolume: 500,
			FilledFee:    100,
		},
		&proto.NewAssetSnapshot{
			AssetID:         asset1,
			IssuerPublicKey: pk,
			Decimals:        4,
			IsNFT:           false,
		},
		&proto.AssetVolumeSnapshot{
			AssetID:       asset1,
			TotalQuantity: *big.NewInt(100000000000),
			IsReissuable:  true,
		},
		&proto.AssetDescriptionSnapshot{
			AssetID:          asset1,
			AssetName:        "foo",
			AssetDescription: "bar",
		},
	}
	failedTxSnap := []proto.AtomicSnapshot{
		&proto.TransactionStatusSnapshot{
			Status: proto.TransactionFailed,
		},
		&proto.WavesBalanceSnapshot{
			Address: addr,
			Balance: 49315001748316,
		},
	}
	elidedTxSnap := []proto.AtomicSnapshot{
		&proto.TransactionStatusSnapshot{
			Status: proto.TransactionElided,
		},
	}

	// Test marshalling and unmarshalling txSnapshotJSON.
	bs := proto.BlockSnapshot{TransactionsSnapshots: []proto.TxSnapshot{
		succeededTxSnap,
		failedTxSnap,
		elidedTxSnap,
	}}
	data, err := json.Marshal(bs)
	require.NoError(t, err)
	require.JSONEq(t, js, string(data))

	var unmBs proto.BlockSnapshot
	err = json.Unmarshal(data, &unmBs)
	require.NoError(t, err)
	assert.Len(t, unmBs.TransactionsSnapshots, len(bs.TransactionsSnapshots))
	for i := range bs.TransactionsSnapshots {
		assert.ElementsMatch(t, bs.TransactionsSnapshots[i], unmBs.TransactionsSnapshots[i])
	}

	// Test empty BlockSnapshot.
	data, err = json.Marshal(proto.BlockSnapshot{TransactionsSnapshots: []proto.TxSnapshot{}})
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))

	// Test BlockSnapshot with nil txSnapshots.
	data, err = json.Marshal(proto.BlockSnapshot{TransactionsSnapshots: nil})
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))

	// Test unmarshalling empty BlockSnapshot.
	var unmEmptyBs proto.BlockSnapshot
	err = json.Unmarshal(data, &unmEmptyBs)
	require.NoError(t, err)
	assert.Len(t, unmEmptyBs.TransactionsSnapshots, 0)
	assert.Nil(t, unmEmptyBs.TransactionsSnapshots)
}

// TestBlockSnapshotEqual tests the comparison of two BlockSnapshot instances.
func TestBlockSnapshotEqual(t *testing.T) {
	addr1, _ := proto.NewAddressFromString("3P9o3uwx3fWZz3b53g53ARUk9sFoPW6z7HA")
	addr2, _ := proto.NewAddressFromString("3P9o3uwx3fWZz3b5aaaaaaaaaaFoPW6z7HB")
	assetID1 := crypto.MustDigestFromBase58("BrjV5AB5S7qN5tLQFbU5tpLj5qeozfVvPxEpDkmmhNP")
	assetID2 := crypto.MustDigestFromBase58("5Zv8JLH8TTvq9iCo6HtB2K7CGpTJt6JTj5yvXaDVrxEJ")
	publicKey1, _ := crypto.NewPublicKeyFromBase58("5TBjL2VdL1XmXq5dC4SYMeH5sVCGmMTeBNNYqWCuEXMn")
	leaseID1 := crypto.MustDigestFromBase58("FjnZ7aY8iqVpZc4M4uPFuDzMB6YShYd4cNmRfQP1p4Su")

	// Setup test cases
	tests := []struct {
		name           string
		blockSnapshotA proto.BlockSnapshot
		blockSnapshotB proto.BlockSnapshot
		wantEqual      bool
	}{
		{
			name: "equal snapshots with single transaction",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr1, Balance: 100}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr1, Balance: 100}},
				},
			},
			wantEqual: true,
		},
		{
			name: "different snapshots with single transaction",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr1, Balance: 100}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr2, Balance: 100}},
				},
			},
			wantEqual: false,
		},
		{
			name: "equal snapshots with multiple transactions",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr1, Balance: 100}},
					{&proto.AssetBalanceSnapshot{Address: addr2, AssetID: assetID1, Balance: 200}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.WavesBalanceSnapshot{Address: addr1, Balance: 100}},
					{&proto.AssetBalanceSnapshot{Address: addr2, AssetID: assetID1, Balance: 200}},
				},
			},
			wantEqual: true,
		},
		{
			name: "snapshots with different asset balances",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetBalanceSnapshot{Address: addr1, AssetID: assetID1, Balance: 300}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetBalanceSnapshot{Address: addr1, AssetID: assetID2, Balance: 300}},
				},
			},
			wantEqual: false,
		},
		{
			name: "snapshots with new lease and cancelled lease snapshots",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.NewLeaseSnapshot{LeaseID: leaseID1, Amount: 1000, SenderPK: publicKey1, RecipientAddr: addr1}},
					{&proto.CancelledLeaseSnapshot{LeaseID: leaseID1}},
				},
			},
			wantEqual: false,
		},
		{
			name: "snapshots with equal AssetVolumeSnapshot",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetVolumeSnapshot{AssetID: assetID1, TotalQuantity: *big.NewInt(1000), IsReissuable: true}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetVolumeSnapshot{AssetID: assetID1, TotalQuantity: *big.NewInt(1000), IsReissuable: true}},
				},
			},
			wantEqual: true,
		},
		{
			name: "snapshots with different AssetVolumeSnapshot reissuability",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetVolumeSnapshot{AssetID: assetID1, TotalQuantity: *big.NewInt(1000), IsReissuable: true}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.AssetVolumeSnapshot{AssetID: assetID1, TotalQuantity: *big.NewInt(1000), IsReissuable: false}},
				},
			},
			wantEqual: false,
		},
		{
			name: "snapshots with equal DataEntriesSnapshot",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.DataEntriesSnapshot{Address: addr1, DataEntries: proto.DataEntries{
						&proto.IntegerDataEntry{Key: "key1", Value: 100},
						&proto.BooleanDataEntry{Key: "key2", Value: true},
					}}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.DataEntriesSnapshot{Address: addr1, DataEntries: proto.DataEntries{
						&proto.IntegerDataEntry{Key: "key1", Value: 100},
						&proto.BooleanDataEntry{Key: "key2", Value: true},
					}}},
				},
			},
			wantEqual: true,
		},
		{
			name: "snapshots with different DataEntriesSnapshot",
			blockSnapshotA: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.DataEntriesSnapshot{Address: addr1, DataEntries: proto.DataEntries{
						&proto.IntegerDataEntry{Key: "key1", Value: 100},
						&proto.BooleanDataEntry{Key: "key2", Value: true},
					}}},
				},
			},
			blockSnapshotB: proto.BlockSnapshot{
				TransactionsSnapshots: []proto.TxSnapshot{
					{&proto.DataEntriesSnapshot{Address: addr1, DataEntries: proto.DataEntries{
						&proto.IntegerDataEntry{Key: "key1", Value: 200}, // Different value
						&proto.BooleanDataEntry{Key: "key2", Value: true},
					}}},
				},
			},
			wantEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal, err := tt.blockSnapshotA.Equal(tt.blockSnapshotB)
			if err != nil {
				t.Errorf("Error comparing snapshots: %v", err)
			}
			if equal != tt.wantEqual {
				t.Errorf("Expected snapshots to be equal: %v, got: %v", tt.wantEqual, equal)
			}
		})
	}
}
