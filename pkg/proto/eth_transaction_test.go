package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthereumTransaction_GetID(t *testing.T) {
	tests := []struct {
		canonicalTxHex string
		expectedIDHex  string
	}{
		{"0x02f86b010284b6ed1ad4856e3c18e22d82520894b69f3f0f21d129d91fc739e0479196bc7f40707e8080c001a02e9ef96d454f7be05ea62c0eb0fac824b6e6161b748c3331c13d988912359ef4a04981e8f8de5be878fa908f8ab128f630caec9eacfa30a2aa06a6be91a0e7db8c", "0xf5e939a88de581f164ac8308d49665d2694ed39b0ce31bfadb4f647d063153be"},
		{"0x02f8b501831fbbb48501cda3524085e8d4a5100083011e7294dac17f958d2ee523a2206206994597c13d831ec780b844a9059cbb000000000000000000000000dcdfe867dcc12d3b33be19fad8c6fe8dca945788000000000000000000000000000000000000000000000000000000000a60e5b0c001a040af00637d5e9365b6caeda07fb9645c3748ba276d59e6fe8abd73192b466deaa06400d246969a37b555bbba8de4d1dabf567548d4530a6be12441004f5400b9e2", "0x9add008e75c5d42297e8c95525715e6c740d18c9dc663a0c44ddde8847c79fc4"},
		{"0x02f902330181d7846f75ef80850df8475800830286ab94ba12222222228d8ba445958a75a0704d566bf2c880b901c452bbbe2900000000000000000000000000000000000000000000000000000000000000e00000000000000000000000006c8ed7774b061c9ea95e5826b57070ce19811f4e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000006c8ed7774b061c9ea95e5826b57070ce19811f4e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001d1ced3dd30fc9d26ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff6aa8a7b23f7b3875a966ddcc83d5b675cc9af54b00020000000000000000008e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb480000000000000000000000006b4d5e9ec2acea23d4110f4803da99e25443c5df00000000000000000000000000000000000000000000000000000002540be40000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000000c080a0bd39f7d952aed3559cc10804691cebe8075d0d710a16c58a9cdfd01bccd52cf0a03c971b321685f0095510da734fbd778fe92237267c3d415cb402a5e01ca82e0c", "0x8050e257f11d1b5aa4fbc370a08413b0860d6bc20fd2bb8965a554f5df307e18"},
		{"0xf86e82146f8513532f83b3825208949c4c39e3cd2f3d0d930e4c065af5ea4a1fcb4a6e880342e341423780008025a086bd7bec8019f17fe77be36468656c9ede915514f1fc158a4eee8a36264b8315a0205b9fa92365441fd7c06fdce3f9d431007bfeb0253032fc1f6364683bff37c5", "0x17594cb0532b464031f794a87296efe4bf8f69d8a80ca024ca2b7b6634021004"},
		{"0xf9066d8233ff851514022c80830dbba094deeb86214606ccd4ccc908c302466485c6643ffa80b90604f4acd7e8000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000005800000000000000000000000000000000000000000000000000000000000000520000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000004a00000000000000000000000002bee0effe7bf36a7de87476b42f03b4be15212880000000000000000000000000000000000000000000000000000000000000420000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000003a00000000000000000000000000000000000000000000000000000000000000340000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002c0000000000000000000000000000000000000000000000000000000006077c46700000000000000000000000000000000000000000000008567c3426fda2ddc300000000000000000000000000000000000000000000000000de0b6b3a764000000000000000000000000000000000000000000000000000598e94928efa010df0000000000000000000000000000000000000000000000859a06ee465d72c5e700000000000000000000000000000000000000000000001e13f735d38e82800000000000000000000000000000000000000000000000000000000000000001a0000000000000000000000000000000000000000000000085d84212ad677400000000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006077c45e0000000000000000000000000000000000000000000000000000000000000003be606dbe281fd7d1c7b0022054377ea9fc70e908f4339af7d47408af278e80010000000000000000000000008fa7490cedb7207281a5ceabee12773046de664e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000d0dffbd959968a400000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000414c33b5ebdd7e016864590b0502d7b491caaabfb8c3324d5710a8657257e4a145163d290e0f4a6e6c7c7f32bd123e658e3d47358a1f885a53ff21ae4a34d8c3da1b000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041ea246374b5c774b7b9941aab595cb9342b1cead78b5d9083fdc636f0b4f448dc4805b47fce465bf204f2434b546368c7bdd370d9e9ca7d59d2e72c2fc43428e81c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041c038f1bc783c5dd9dd0a5f29f9c3c98d5b8e090c558ab95d01e6460ac192c04e5999f6b96e255b90a335b6b244389e104cddcbd4208642d30196fd0be36a13191b0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000419ed5e90dcae2583f9d010530a7cb3f2a64472c0aee87e97ad5cc0abc6a584c3044c5aedc3a6d8ec6b86bc282c19df5174b2fe15b66e68dd1dd169e6081d0f9851b0000000000000000000000000000000000000000000000000000000000000026a0aeeb2fc81a6a27d1ecd0a321caed2012abcda1f95b461066bdad39006759ac30a04d6a015a42ac497908909ea58144e29c4489c2e3881910ac0ce6cba66e8cf85f", "0xb6423291ae04df21fc8f85b00800418912982410c99fa56abc30b24d704dc4b7"},
	}
	for _, tc := range tests {
		canonical, err := DecodeFromHexString(tc.canonicalTxHex)
		require.NoError(t, err)

		var ethTx EthereumTransaction
		err = ethTx.DecodeCanonical(canonical)
		require.NoError(t, err)

		// dummy scheme argument
		actualID, err := ethTx.GetID(0)
		require.NoError(t, err, "failed to generate transaction ID")

		actualIDHex := EncodeToHexString(actualID)
		require.Equal(t, tc.expectedIDHex, actualIDHex, "ids don't equal")
	}
}
