package bn254

import (
	"bytes"
	"testing"
)

func TestHashToField(t *testing.T) {
	DST := []byte("QUUX-V01-CS02-with-expander")
	for i, v := range []struct {
		msg      []byte
		outLen   int
		expected []byte
	}{
		{
			msg:      []byte(""),
			outLen:   32,
			expected: fromHex(32, "0xf659819a6473c1835b25ea59e3d38914c98b374f0970b7e4c92181df928fca88"),
		},
		{
			msg:      []byte("abc"),
			outLen:   32,
			expected: fromHex(32, "1c38f7c211ef233367b2420d04798fa4698080a8901021a795a1151775fe4da7"),
		},
		{
			msg:      []byte("abcdef0123456789"),
			outLen:   32,
			expected: fromHex(32, "8f7e7b66791f0da0dbb5ec7c22ec637f79758c0a48170bfb7c4611bd304ece89"),
		},
		{
			msg:      []byte("q128_qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"),
			outLen:   32,
			expected: fromHex(32, "72d5aa5ec810370d1f0013c0df2f1d65699494ee2a39f72e1716b1b964e1c642"),
		},
		{
			msg:      []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			outLen:   32,
			expected: fromHex(32, "3b8e704fc48336aca4c2a12195b720882f2162a4b7b13a9c350db46f429b771b"),
		},
		{
			msg:      []byte(""),
			outLen:   128,
			expected: fromHex(128, "8bcffd1a3cae24cf9cd7ab85628fd111bb17e3739d3b53f89580d217aa79526f1708354a76a402d3569d6a9d19ef3de4d0b991e4f54b9f20dcde9b95a66824cbdf6c1a963a1913d43fd7ac443a02fc5d9d8d77e2071b86ab114a9f34150954a7531da568a1ea8c760861c0cde2005afc2c114042ee7b5848f5303f0611cf297f"),
		},
		{
			msg:      []byte("abc"),
			outLen:   128,
			expected: fromHex(128, "fe994ec51bdaa821598047b3121c149b364b178606d5e72bfbb713933acc29c186f316baecf7ea22212f2496ef3f785a27e84a40d8b299cec56032763eceeff4c61bd1fe65ed81decafff4a31d0198619c0aa0c6c51fca15520789925e813dcfd318b542f8799441271f4db9ee3b8092a7a2e8d5b75b73e28fb1ab6b4573c192"),
		},
		{
			msg:      []byte("abcdef0123456789"),
			outLen:   128,
			expected: fromHex(128, "c9ec7941811b1e19ce98e21db28d22259354d4d0643e301175e2f474e030d32694e9dd5520dde93f3600d8edad94e5c364903088a7228cc9eff685d7eaac50d5a5a8229d083b51de4ccc3733917f4b9535a819b445814890b7029b5de805bf62b33a4dc7e24acdf2c924e9fe50d55a6b832c8c84c7f82474b34e48c6d43867be"),
		},
		{
			msg:      []byte("q128_qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"),
			outLen:   128,
			expected: fromHex(128, "48e256ddba722053ba462b2b93351fc966026e6d6db493189798181c5f3feea377b5a6f1d8368d7453faef715f9aecb078cd402cbd548c0e179c4ed1e4c7e5b048e0a39d31817b5b24f50db58bb3720fe96ba53db947842120a068816ac05c159bb5266c63658b4f000cbf87b1209a225def8ef1dca917bcda79a1e42acd8069"),
		},
		{
			msg:      []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			outLen:   128,
			expected: fromHex(128, "396962db47f749ec3b5042ce2452b619607f27fd3939ece2746a7614fb83a1d097f554df3927b084e55de92c7871430d6b95c2a13896d8a33bc48587b1f66d21b128a1a8240d5b0c26dfe795a1a842a0807bb148b77c2ef82ed4b6c9f7fcb732e7f94466c8b51e52bf378fba044a31f5cb44583a892f5969dcd73b3fa128816e"),
		},
	} {
		res, err := expandMsgSHA256XMD(v.msg, DST, v.outLen)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(v.expected, res) {
			t.Fatal("hash to field failed", i)
		}
	}
}
