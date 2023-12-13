package proto_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTxSnapshotMarshalToPBAndUnmarshalFromPB(t *testing.T) {
	const scheme = proto.TestNetScheme
	testCases := []struct {
		testCaseName string
		pbInBase64   string
	}{
		{
			testCaseName: "waves_balances",
			pbInBase64:   "CiQKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgYQgJTr3AMKJAoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSBhCAqNa5Bw==", //nolint:lll
		},
		{
			testCaseName: "asset_balances",
			pbInBase64:   "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQ==", //nolint:lll
		},
		{
			testCaseName: "data_entries",
			pbInBase64:   "YloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgB", //nolint:lll
		},
		{
			testCaseName: "account_script",
			pbInBase64:   "Wi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoB",
		},
		{
			testCaseName: "asset_script",
			pbInBase64:   "QisKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgcGAQaw0U/P",
		},
		{
			testCaseName: "new_lease",
			pbInBase64:   "EiIKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1GICa4uEQEiIKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEICuzb4UGmYKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqNEiBQx1mvQnelVPDNtCyxR82wU73hEX9bBDsNfUlEuNBgaBoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoggPKLqAk=", //nolint:lll
		},
		{
			testCaseName: "cancelled_lease",
			pbInBase64:   "EiIKGgFUMCPLqLW81X2Atgaj2KwF9QkaJq47Cev9GICo1rkHEhwKGgFUYSJd8vzI9rq7GdIuDy65JMc8zi497E98IiIKILiCMyyFggW8Zd2LGt/AtMr7WWp+kfWbzlN93pXZqzqN", //nolint:lll
		},
		{
			testCaseName: "sponsorship",
			pbInBase64:   "aiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwq",
		},
		{
			testCaseName: "alias",
			pbInBase64:   "SiYKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgh3YXZlc2V2bw==",
		},
		{
			testCaseName: "order_fill",
			pbInBase64:   "UisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAP", //nolint:lll
		},
		{
			testCaseName: "new_asset",
			pbInBase64:   "KkYKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBniABKkYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEiDcYGFqY9MotHTpDpskoycN/Mt62bZfPxIC4fpU0ZTBnhgIMi8KIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEAEaCQT/////////9jIlCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxoBAQ==", //nolint:lll
		},
		{
			testCaseName: "reissued_asset",
			pbInBase64:   "MigKIDhvjT3TTlJ+v4Ni205vcYc1m9WWgnQPFovjmJI1H62yGgQ7msoA",
		},
		{
			testCaseName: "renamed_asset",
			pbInBase64:   "OkMKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEgduZXduYW1lGhZzb21lIGZhbmN5IGRlc2NyaXB0aW9u",
		},
		{
			testCaseName: "failed_transaction",
			pbInBase64:   "CiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQ4PHE1wlwAQ==",
		},
		{
			testCaseName: "elided_transaction",
			pbInBase64:   "cAI=",
		},
		{
			testCaseName: "all_together",
			pbInBase64:   "CkMKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EiUKIF5mn4IKZ9CIbYdHjPBDoqx4XMevVdwxzhB1OUvTUKJbEJBOCkQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEiYKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEKCcAQokChoBVGD9UO8g3kVxIH37nIi+fBwviiLHCtiPtRIGEICU69wDCiQKGgFUQsXJY3P1D9gTUGBPHBTypsklatr9GbAqEgYQgKjWuQcSIgoaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7UYgJri4RASIgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoQgK7NvhQSIgoaAVQwI8uotbzVfYC2BqPYrAX1CRomrjsJ6/0YgKjWuQcSHAoaAVRhIl3y/Mj2ursZ0i4PLrkkxzzOLj3sT3waZgoguIIzLIWCBbxl3Ysa38C0yvtZan6R9ZvOU33eldmrOo0SIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoGhoBVELFyWNz9Q/YE1BgTxwU8qbJJWra/RmwKiCA8ouoCSIiCiC4gjMshYIFvGXdixrfwLTK+1lqfpH1m85Tfd6V2as6jSpGCiBeZp+CCmfQiG2HR4zwQ6KseFzHr1XcMc4QdTlL01CiWxIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4gASpGCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThIg3GBhamPTKLR06Q6bJKMnDfzLetm2Xz8SAuH6VNGUwZ4YCDIvCiB4ncARI9U2D3CCr9S0ari/6LUWC7+1JsBD5fx4Ok+JThABGgkE//////////YyJQogXmafggpn0Ihth0eM8EOirHhcx69V3DHOEHU5S9NQolsaAQEyKAogOG+NPdNOUn6/g2LbTm9xhzWb1ZaCdA8Wi+OYkjUfrbIaBDuaygA6QwogeJ3AESPVNg9wgq/UtGq4v+i1Fgu/tSbAQ+X8eDpPiU4SB25ld25hbWUaFnNvbWUgZmFuY3kgZGVzY3JpcHRpb25KJgoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCHdhdmVzZXZvUisKIMkknO8yHpMUT/XKkkdlrbYCG0Dt+qvVgphfgtRbyRDMEICU69wDGNAPUisKIJZ9YwvJObbWItHAD2zhbaFOTFx2zQ4p0Xbo81GXHKeEEICU69wDGNAPWi4KIFDHWa9Cd6VU8M20LLFHzbBTveERf1sEOw19SUS40GBoEgcGAQaw0U/PGPoBYloKGgFUYP1Q7yDeRXEgffuciL58HC+KIscK2I+1EgUKA2ZvbxISCgNiYXJqC1N0cmluZ1ZhbHVlEiEKA2JhemIaAVRg/VDvIN5FcSB9+5yIvnwcL4oixwrYj7ViLwoaAVRCxcljc/UP2BNQYE8cFPKmySVq2v0ZsCoSCAoDZm9vULAJEgcKA2JhclgBaiUKIHidwBEj1TYPcIKv1LRquL/otRYLv7UmwEPl/Hg6T4lOEPwqcAE=", //nolint:lll
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.testCaseName, func(t *testing.T) {
			expectedData, err := base64.StdEncoding.DecodeString(testCase.pbInBase64)
			require.NoError(t, err)

			expectedTxSnapshotProto := new(g.TransactionStateSnapshot)
			err = expectedTxSnapshotProto.UnmarshalVT(expectedData)
			require.NoError(t, err)

			txSnapshot, err := proto.TxSnapshotsFromProtobuf(scheme, expectedTxSnapshotProto)
			assert.NoError(t, err)

			txSnapshotProto := new(g.TransactionStateSnapshot)
			for _, snapshot := range txSnapshot {
				appendErr := snapshot.AppendToProtobuf(txSnapshotProto)
				require.NoError(t, appendErr)
			}
			assert.Equal(t, expectedTxSnapshotProto, txSnapshotProto)

			marshaledData, err := txSnapshotProto.MarshalVTStrict()
			require.NoError(t, err)
			assert.Equal(t, expectedData, marshaledData)
		})
	}
}
