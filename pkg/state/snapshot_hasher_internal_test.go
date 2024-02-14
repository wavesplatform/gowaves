package state

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTxSnapshotHasher(t *testing.T) {
	const (
		scheme      = proto.TestNetScheme
		blockHeight = 10
	)
	testCases := []struct {
		testCaseName         string
		pbInBase64           string
		prevStateHashHex     string
		expectedStateHashHex string
		transactionIDBase58  string
	}{
		{
			testCaseName:         "waves_balances",
			pbInBase64:           "CiQKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgYQgJTr3AMKJAoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSBhCAqNa5Bw==", //nolint:lll
			prevStateHashHex:     crypto.MustFastHash(nil).Hex(),
			expectedStateHashHex: "f0a8b6745534c2d20412f40cdb097b7050898e44531a661ef64fc5be0744ac72",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "asset_balances",
			pbInBase64:           "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQ==", //nolint:lll
			prevStateHashHex:     "f0a8b6745534c2d20412f40cdb097b7050898e44531a661ef64fc5be0744ac72",
			expectedStateHashHex: "16c4803d12ee8e9d6c705ca6334fd84f57c0e78c4ed8a9a3dc6c28dcd9b29a34",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "data_entries",
			pbInBase64:           "YloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgB", //nolint:lll
			prevStateHashHex:     "16c4803d12ee8e9d6c705ca6334fd84f57c0e78c4ed8a9a3dc6c28dcd9b29a34",
			expectedStateHashHex: "d33269372999bfd8f7afdf97e23bc343bcf3812f437e8971681a37d56868ec8a",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "account_script",
			pbInBase64:           "Wi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoB",
			prevStateHashHex:     "d33269372999bfd8f7afdf97e23bc343bcf3812f437e8971681a37d56868ec8a",
			expectedStateHashHex: "dcdf7df91b11fdbeb2d99c4fd64abb4657adfda15eed63b1d4730aa2b6275ee2",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "asset_script",
			pbInBase64:           "QisKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgcGAQaw0U/P",
			prevStateHashHex:     "dcdf7df91b11fdbeb2d99c4fd64abb4657adfda15eed63b1d4730aa2b6275ee2",
			expectedStateHashHex: "d3c7f2aeb1d978ecebc2fe1f0555e4378cef5171db460d8bbfebef0e59c3a44c",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "new_lease",
			pbInBase64:           "EiIKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1GICa4uEQEiIKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEICuzb4UGmYKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqNEiBQx1mvQnelVPDNtCyxR82wU73hEX9bBDsNfUlEuNBgaBoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoggPKLqAk=", //nolint:lll
			prevStateHashHex:     "d3c7f2aeb1d978ecebc2fe1f0555e4378cef5171db460d8bbfebef0e59c3a44c",
			expectedStateHashHex: "2665ce187b867f2dae95699882d9fd7c31039c505b8af93ed22cada90524ff37",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "cancelled_lease",
			pbInBase64:           "EiIKGgFUMCPLqLW81X2Atgaj2KwF9QkaJq47Cev9GICo1rkHEhwKGgFUYSJd8vzI9rq7GdIuDy65JMc8zi497E98IiIKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqN", //nolint:lll
			prevStateHashHex:     "2665ce187b867f2dae95699882d9fd7c31039c505b8af93ed22cada90524ff37",
			expectedStateHashHex: "dafc56fb4f5e13ddd3e82547874e154c5c61ac556e76e9e9766b5d7ccbc1e1be",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "sponsorship",
			pbInBase64:           "aiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwq",
			prevStateHashHex:     "dafc56fb4f5e13ddd3e82547874e154c5c61ac556e76e9e9766b5d7ccbc1e1be",
			expectedStateHashHex: "d9eab5091d57c18c38e0a8702e7cbe6f133e109281f2ef0f2bc88686b458f31f",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "alias",
			pbInBase64:           "SiYKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgh3YXZlc2V2bw==",
			prevStateHashHex:     "d9eab5091d57c18c38e0a8702e7cbe6f133e109281f2ef0f2bc88686b458f31f",
			expectedStateHashHex: "eaa251c161cfe875932275ce6ff8873cd169099e021f09245f4069ccd58d6669",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "order_fill",
			pbInBase64:           "UisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAP", //nolint:lll
			prevStateHashHex:     "eaa251c161cfe875932275ce6ff8873cd169099e021f09245f4069ccd58d6669",
			expectedStateHashHex: "de22575b5c2ef7de6388c0ea96e6d0f172802f4c8e33684473c91af65866b1d4",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "new_asset",
			pbInBase64:           "KkYKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBniABKkYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBnhgIMi8KIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEAEaCQT/////////9jIlCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxoBAQ==", //nolint:lll
			prevStateHashHex:     "de22575b5c2ef7de6388c0ea96e6d0f172802f4c8e33684473c91af65866b1d4",
			expectedStateHashHex: "5f09358e944a386ad12b4f6e22c79a5c614967f6da40465e30d878e9b58e75e2",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "reissued_asset",
			pbInBase64:           "MigKIDhvjT3TTlJ+v4Ni205vcYc1m9WWgnQPFovjmJI1H62yGgQ7msoA",
			prevStateHashHex:     "5f09358e944a386ad12b4f6e22c79a5c614967f6da40465e30d878e9b58e75e2",
			expectedStateHashHex: "6d5e0f4e2a4b650541b66711bbc687f51fea7bc3aa35b43642e21ab3dd064743",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "renamed_asset",
			pbInBase64:           "OkMKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgduZXduYW1lGhZzb21lIGZhbmN5IGRlc2NyaXB0aW9u",
			prevStateHashHex:     "6d5e0f4e2a4b650541b66711bbc687f51fea7bc3aa35b43642e21ab3dd064743",
			expectedStateHashHex: "885ac4b03397e63cdc1a2e3fe60d2aae0d4701e5cfb8c19ca80feb912a028a48",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "failed_transaction",
			pbInBase64:           "CiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQ4PHE1wlwAQ==",
			prevStateHashHex:     "885ac4b03397e63cdc1a2e3fe60d2aae0d4701e5cfb8c19ca80feb912a028a48",
			expectedStateHashHex: "4185fb099c6dd4f483d4488045cc0912f02b9c292128b90142367af680ce2a32",
			transactionIDBase58:  "C6tHv5UkPaC53WFEr1Kv4Nb6q7hHdypDThjyYwRUUhQ8",
		},
		{
			testCaseName:         "elided_transaction",
			pbInBase64:           "cAI=",
			prevStateHashHex:     "4185fb099c6dd4f483d4488045cc0912f02b9c292128b90142367af680ce2a32",
			expectedStateHashHex: "7a15507d73ff9f98c3c777e687e23a4c8b33d02212203be73f0518403e91d431",
			transactionIDBase58:  "Feix2sUAxsqhUH5kwRJBqdXur3Fj2StgCksbhdt67fXc",
		},
		{
			testCaseName:         "all_together",
			pbInBase64:           "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQokChoBVGD9UO8g3kVxIH37nIi+fBwviiLHCtiPtRIGEICU69wDCiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQgKjWuQcSIgoaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7UYgJri4RASIgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoQgK7NvhQSIgoaAVQwI8uotbzVfYC2BqPYrAX1CRomrjsJ6/0YgKjWuQcSHAoaAVRhIl3y/Mj2ursZ0i4PLrkkxzzOLj3sT3waZgoguIIzLIWCBbxl3Ysa38C0yvtZan6R9ZvOU33eldmrOo0SIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoGhoBVELFyWNz9Q/YE1BgTxwU8qbJJWra/RmwKiCA8ouoCSIiCiC4gjMshYIFvGXdixrfwLTK+1lqfpH1m85Tfd6V2as6jSpGCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4gASpGCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4YCDIvCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThABGgkE//////////YyJQogXmafggpn0Ihth0eM8EOirHhcx69V3DHOEHU5S9NQolsaAQEyKAogOG+NPdNOUn6/g2LbTm9xhzWb1ZaCdA8Wi+OYkjUfrbIaBDuaygA6QwogeJ3AESPVNg9wgq/UtGq4v+i1Fgu/tSbAQ+X8eDpPiU4SB25ld25hbWUaFnNvbWUgZmFuY3kgZGVzY3JpcHRpb25KJgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCHdhdmVzZXZvUisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAPWi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoBYloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgBaiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwqcAE=", //nolint:lll
			prevStateHashHex:     "7a15507d73ff9f98c3c777e687e23a4c8b33d02212203be73f0518403e91d431",
			expectedStateHashHex: "6502773294f32cc1702d374ffc1e67ee278cd63c5f00432f80f64a689fcb17f9",
			transactionIDBase58:  "5gEi2kgbMSfUzdDXRKovEbEezq5ACpr8WTeafwkKQmHW",
		},
		{
			testCaseName:         "asset_volume_two's_complement",
			pbInBase64:           "MicKIOfYm9p3M/NiYXCvwCU3ho5eVFpwE5iekWev4QXhZMvuEAEaAcg=",
			prevStateHashHex:     "6502773294f32cc1702d374ffc1e67ee278cd63c5f00432f80f64a689fcb17f9",
			expectedStateHashHex: "b5f7e36556cb0d9a72bc9612be017a3cf174cfcb059d86c91621bfe7e8b74ff1",
			transactionIDBase58:  "Gc2kPdPb1qrCPMy1Ga6SD5PDs2Equa6aazxhKjtDzrv1", // valid txID from testnet
		},
	}

	hasher, hErr := newTxSnapshotHasherDefault()
	require.NoError(t, hErr)
	defer hasher.Release()

	for _, testCase := range testCases {
		t.Run(testCase.testCaseName, func(t *testing.T) {
			pbBytes, err := base64.StdEncoding.DecodeString(testCase.pbInBase64)
			require.NoError(t, err)

			txSnapshotProto := new(g.TransactionStateSnapshot)
			err = txSnapshotProto.UnmarshalVT(pbBytes)
			require.NoError(t, err)

			prevHashBytes, err := hex.DecodeString(testCase.prevStateHashHex)
			require.NoError(t, err)
			prevHash, err := crypto.NewDigestFromBytes(prevHashBytes)
			require.NoError(t, err)

			txSnapshot, err := proto.TxSnapshotsFromProtobuf(scheme, txSnapshotProto)
			assert.NoError(t, err)

			var transactionID crypto.Digest
			if txIDStr := testCase.transactionIDBase58; txIDStr != "" {
				transactionID, err = crypto.NewDigestFromBase58(txIDStr)
				require.NoError(t, err)
			}

			hash, err := calculateTxSnapshotStateHash(hasher, transactionID.Bytes(), blockHeight, prevHash, txSnapshot)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedStateHashHex, hash.Hex())
		})
	}
}

func BenchmarkTxSnapshotHasher(b *testing.B) {
	const (
		scheme      = proto.TestNetScheme
		blockHeight = 10
	)
	testCase := struct {
		testCaseName         string
		pbInBase64           string
		prevStateHashHex     string
		transactionIDBase58  string
		expectedStateHashHex string
	}{
		testCaseName:         "all_together",
		pbInBase64:           "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQokChoBVGD9UO8g3kVxIH37nIi+fBwviiLHCtiPtRIGEICU69wDCiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQgKjWuQcSIgoaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7UYgJri4RASIgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoQgK7NvhQSIgoaAVQwI8uotbzVfYC2BqPYrAX1CRomrjsJ6/0YgKjWuQcSHAoaAVRhIl3y/Mj2ursZ0i4PLrkkxzzOLj3sT3waZgoguIIzLIWCBbxl3Ysa38C0yvtZan6R9ZvOU33eldmrOo0SIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoGhoBVELFyWNz9Q/YE1BgTxwU8qbJJWra/RmwKiCA8ouoCSIiCiC4gjMshYIFvGXdixrfwLTK+1lqfpH1m85Tfd6V2as6jSpGCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4gASpGCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4YCDIvCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThABGgkE//////////YyJQogXmafggpn0Ihth0eM8EOirHhcx69V3DHOEHU5S9NQolsaAQEyKAogOG+NPdNOUn6/g2LbTm9xhzWb1ZaCdA8Wi+OYkjUfrbIaBDuaygA6QwogeJ3AESPVNg9wgq/UtGq4v+i1Fgu/tSbAQ+X8eDpPiU4SB25ld25hbWUaFnNvbWUgZmFuY3kgZGVzY3JpcHRpb25KJgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCHdhdmVzZXZvUisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAPWi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoBYloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgBaiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwqcAE=", //nolint:lll
		prevStateHashHex:     "7a15507d73ff9f98c3c777e687e23a4c8b33d02212203be73f0518403e91d431",
		transactionIDBase58:  "5gEi2kgbMSfUzdDXRKovEbEezq5ACpr8WTeafwkKQmHW",
		expectedStateHashHex: "6502773294f32cc1702d374ffc1e67ee278cd63c5f00432f80f64a689fcb17f9",
	}
	pbBytes, err := base64.StdEncoding.DecodeString(testCase.pbInBase64)
	require.NoError(b, err)

	txSnapshotProto := new(g.TransactionStateSnapshot)
	err = txSnapshotProto.UnmarshalVT(pbBytes)
	require.NoError(b, err)

	prevHashBytes, err := hex.DecodeString(testCase.prevStateHashHex)
	require.NoError(b, err)
	prevHash, err := crypto.NewDigestFromBytes(prevHashBytes)
	require.NoError(b, err)

	txSnapshot, err := proto.TxSnapshotsFromProtobuf(scheme, txSnapshotProto)
	assert.NoError(b, err)

	transactionID, err := crypto.NewDigestFromBase58(testCase.transactionIDBase58)
	require.NoError(b, err)
	txID := transactionID.Bytes()

	expectedHashBytes, err := hex.DecodeString(testCase.expectedStateHashHex)
	require.NoError(b, err)
	expectedHash, err := crypto.NewDigestFromBytes(expectedHashBytes)
	require.NoError(b, err)

	hasher, err := newTxSnapshotHasherDefault()
	require.NoError(b, err)
	defer hasher.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.Run(testCase.testCaseName, func(b *testing.B) {
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				h, hErr := calculateTxSnapshotStateHash(hasher, txID, blockHeight, prevHash, txSnapshot)
				if hErr != nil {
					b.Fatalf("error occured: %+v", err)
				}
				if h != expectedHash {
					b.Fatalf("expectedHash=%s  != actual=%s", expectedHash.Hex(), h.Hex())
				}
			}
		})
	}
}
