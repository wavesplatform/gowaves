package state

import (
	"encoding/base64"
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
		prevStateHashBase64  string
		expectedStateHashHex string
		transactionIDBase58  string
	}{
		{
			testCaseName:         "waves_balances",
			pbInBase64:           "CiQKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgYQgJTr3AMKJAoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSBhCAqNa5Bw==", //nolint:lll
			prevStateHashBase64:  "",
			expectedStateHashHex: "954bf440a83542e528fe1e650471033e42d97c5896cc571aec39fccc912d7db0",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "asset_balances",
			pbInBase64:           "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQ==", //nolint:lll
			prevStateHashBase64:  "954bf440a83542e528fe1e650471033e42d97c5896cc571aec39fccc912d7db0",
			expectedStateHashHex: "534e27c3a787536e18faf844ff217a8f14e5323dfcd3cc5b9ab3a8e261f60cf7",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "data_entries",
			pbInBase64:           "YloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgB", //nolint:lll
			prevStateHashBase64:  "534e27c3a787536e18faf844ff217a8f14e5323dfcd3cc5b9ab3a8e261f60cf7",
			expectedStateHashHex: "b1440780a268eeaf9f6bb285a97ee35582cb84382576e84432b2e61b86d64581",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "account_script",
			pbInBase64:           "Wi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoB",
			prevStateHashBase64:  "b1440780a268eeaf9f6bb285a97ee35582cb84382576e84432b2e61b86d64581",
			expectedStateHashHex: "ca42620b03b437e025bec14152c3f7d8ff65b8fb1062b5013363186484176cb7",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "asset_script",
			pbInBase64:           "QisKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgcGAQaw0U/P",
			prevStateHashBase64:  "ca42620b03b437e025bec14152c3f7d8ff65b8fb1062b5013363186484176cb7",
			expectedStateHashHex: "4e9cbb5349a31d2954d57f67d2fc5cf73dd1ce90b508299cf5f92b1b45ca668f",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "new_lease",
			pbInBase64:           "EiIKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1GICa4uEQEiIKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEICuzb4UGmYKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqNEiBQx1mvQnelVPDNtCyxR82wU73hEX9bBDsNfUlEuNBgaBoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoggPKLqAk=", //nolint:lll
			prevStateHashBase64:  "4e9cbb5349a31d2954d57f67d2fc5cf73dd1ce90b508299cf5f92b1b45ca668f",
			expectedStateHashHex: "8615df0268bcc76e851a9925e07f212a875a1bd047b0cccbc4ad3d842895f16e",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "cancelled_lease",
			pbInBase64:           "EiIKGgFUMCPLqLW81X2Atgaj2KwF9QkaJq47Cev9GICo1rkHEhwKGgFUYSJd8vzI9rq7GdIuDy65JMc8zi497E98IiIKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqN", //nolint:lll
			prevStateHashBase64:  "8615df0268bcc76e851a9925e07f212a875a1bd047b0cccbc4ad3d842895f16e",
			expectedStateHashHex: "3bb24694ea57c1d6b2eec2c549d5ba591853bd7f21959027d2793d5c0846cc8d",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "sponsorship",
			pbInBase64:           "aiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwq",
			prevStateHashBase64:  "3bb24694ea57c1d6b2eec2c549d5ba591853bd7f21959027d2793d5c0846cc8d",
			expectedStateHashHex: "4fd2ceeb81d4d9c7ebad4391fbd938cfc40564088d2cc71801d308d56eca9b75",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "alias",
			pbInBase64:           "SiYKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgh3YXZlc2V2bw==",
			prevStateHashBase64:  "4fd2ceeb81d4d9c7ebad4391fbd938cfc40564088d2cc71801d308d56eca9b75",
			expectedStateHashHex: "0f02911227a9835c1248822f4e500213c4fc4c05a83a5d27680f67d1d1f6a8ee",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "order_fill",
			pbInBase64:           "UisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAP", //nolint:lll
			prevStateHashBase64:  "0f02911227a9835c1248822f4e500213c4fc4c05a83a5d27680f67d1d1f6a8ee",
			expectedStateHashHex: "4d0d2c893b435d1bbc3464c59ceda196961b94a81cfb9bb2c50fe03c06f23d00",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "new_asset",
			pbInBase64:           "KkYKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBniABKkYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBnhgIMi8KIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEAEaCQT/////////9jIlCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxoBAQ==", //nolint:lll
			prevStateHashBase64:  "4d0d2c893b435d1bbc3464c59ceda196961b94a81cfb9bb2c50fe03c06f23d00",
			expectedStateHashHex: "e2baa6d7e863fc1f5f6cec326b01e577c4509e927f3b13ed7818af9075be82c3",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "reissued_asset",
			pbInBase64:           "MigKIDhvjT3TTlJ+v4Ni205vcYc1m9WWgnQPFovjmJI1H62yGgQ7msoA",
			prevStateHashBase64:  "e2baa6d7e863fc1f5f6cec326b01e577c4509e927f3b13ed7818af9075be82c3",
			expectedStateHashHex: "a161ea70fa027f6127763cdb946606e3d65445915ec26c369e0ff28b37bee8cd",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "renamed_asset",
			pbInBase64:           "OkMKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgduZXduYW1lGhZzb21lIGZhbmN5IGRlc2NyaXB0aW9u",
			prevStateHashBase64:  "a161ea70fa027f6127763cdb946606e3d65445915ec26c369e0ff28b37bee8cd",
			expectedStateHashHex: "7a43e1fb599e8a921ecb2a83b4b871bd46db4569bf5c4f6c9225479191450a58",
			transactionIDBase58:  "",
		},
		{
			testCaseName:         "failed_transaction",
			pbInBase64:           "CiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQ4PHE1wlwAQ==",
			prevStateHashBase64:  "7a43e1fb599e8a921ecb2a83b4b871bd46db4569bf5c4f6c9225479191450a58",
			expectedStateHashHex: "dfa190a84d59edda03428c93c4b1be4c50f12adf3c11528ef0bdd8db1edaf49b",
			transactionIDBase58:  "C6tHv5UkPaC53WFEr1Kv4Nb6q7hHdypDThjyYwRUUhQ8",
		},
		{
			testCaseName:         "elided_transaction",
			pbInBase64:           "cAI=",
			prevStateHashBase64:  "dfa190a84d59edda03428c93c4b1be4c50f12adf3c11528ef0bdd8db1edaf49b",
			expectedStateHashHex: "002f4f7f3741668c10a8ba92b4b183680fd6659bafd36037be6f9a636510b128",
			transactionIDBase58:  "Feix2sUAxsqhUH5kwRJBqdXur3Fj2StgCksbhdt67fXc",
		},
		{
			testCaseName:         "all_together",
			pbInBase64:           "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQokChoBVGD9UO8g3kVxIH37nIi+fBwviiLHCtiPtRIGEICU69wDCiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQgKjWuQcSIgoaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7UYgJri4RASIgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoQgK7NvhQSIgoaAVQwI8uotbzVfYC2BqPYrAX1CRomrjsJ6/0YgKjWuQcSHAoaAVRhIl3y/Mj2ursZ0i4PLrkkxzzOLj3sT3waZgoguIIzLIWCBbxl3Ysa38C0yvtZan6R9ZvOU33eldmrOo0SIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoGhoBVELFyWNz9Q/YE1BgTxwU8qbJJWra/RmwKiCA8ouoCSIiCiC4gjMshYIFvGXdixrfwLTK+1lqfpH1m85Tfd6V2as6jSpGCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4gASpGCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4YCDIvCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThABGgkE//////////YyJQogXmafggpn0Ihth0eM8EOirHhcx69V3DHOEHU5S9NQolsaAQEyKAogOG+NPdNOUn6/g2LbTm9xhzWb1ZaCdA8Wi+OYkjUfrbIaBDuaygA6QwogeJ3AESPVNg9wgq/UtGq4v+i1Fgu/tSbAQ+X8eDpPiU4SB25ld25hbWUaFnNvbWUgZmFuY3kgZGVzY3JpcHRpb25KJgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCHdhdmVzZXZvUisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAPWi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoBYloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgBaiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwqcAE=", //nolint:lll
			prevStateHashBase64:  "002f4f7f3741668c10a8ba92b4b183680fd6659bafd36037be6f9a636510b128",
			expectedStateHashHex: "a65304008a49f4ae10cd4af0e61c5d59ba048f0766846d9239d0b28275a0184b",
			transactionIDBase58:  "5gEi2kgbMSfUzdDXRKovEbEezq5ACpr8WTeafwkKQmHW",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.testCaseName, func(t *testing.T) {
			pbBytes, err := base64.StdEncoding.DecodeString(testCase.pbInBase64)
			require.NoError(t, err)

			txSnapshotProto := new(g.TransactionStateSnapshot)
			err = txSnapshotProto.UnmarshalVT(pbBytes)
			require.NoError(t, err)

			prevHash, err := base64.StdEncoding.DecodeString(testCase.prevStateHashBase64)
			require.NoError(t, err)

			txSnapshot, err := proto.TxSnapshotsFromProtobuf(scheme, txSnapshotProto)
			assert.NoError(t, err)

			var transactionID crypto.Digest
			if txIDStr := testCase.transactionIDBase58; txIDStr != "" {
				transactionID, err = crypto.NewDigestFromBase58(txIDStr)
				require.NoError(t, err)
			}
			hasher := newTxSnapshotHasher(blockHeight, transactionID)
			defer hasher.Release()

			for i, snapshot := range txSnapshot {
				err = snapshot.Apply(&hasher)
				require.NoErrorf(t, err, "failed to apply %d-th atomic snapshot", i+1)
			}

			hash, err := hasher.CalculateHash(prevHash)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedStateHashHex, hash.Hex())
		})
	}
}
