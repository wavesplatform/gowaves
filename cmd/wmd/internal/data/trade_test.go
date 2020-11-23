package data

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"

	"testing"
	"time"
)

func TestTradeBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		amountAsset string
		priceAsset  string
		id          string
		orderType   proto.OrderType
		seller      string
		buyer       string
		matcher     string
		price       uint64
		amount      uint64
	}{
		{"7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "35u3djrR6du2YDLwCkP1N1SXah4PkggQZVV3eGosXjiS", "6SkVLhHY79UcAaU1sbRWHcKcr8ipWrVK3et3kM4JN5v8", proto.Buy, "3P7Rp9qp9qZYgGYtUiP7twR8MzESdqZ4Hsx", "3P2uyk57HgSpBJkBjLBY5Eu2Vpd98j2WTAq", "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3", 12345, 67890},
		{"2sBjKeKCgTBYTpGARMKosU1uYJWct68RSZFHKvMzPieU", "5vRtEa2ygi3pAvE4xnypytJqM83Qsra6CTQNX9mtfD4m", "FbAq8kWEJjdzD7StCkhqfd4hrqf3P6ATju7usGgHVC14", proto.Sell, "3PLCkxibx666sB4oNs3fHZZk6MDfSC82YNA", "3PAmhzHgxzxqVttGFRgVCFUFHoGHqmuchec", "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3", 67890, 12345},
	}
	for _, tc := range tests {
		aa, _ := crypto.NewDigestFromBase58(tc.amountAsset)
		pa, _ := crypto.NewDigestFromBase58(tc.priceAsset)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		ba, _ := proto.NewAddressFromString(tc.buyer)
		sa, _ := proto.NewAddressFromString(tc.seller)
		ma, _ := proto.NewAddressFromString(tc.matcher)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tr := Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: id, Buyer: ba, Seller: sa, Matcher: ma, Price: tc.price, Amount: tc.amount, Timestamp: ts}
		b, err := tr.MarshalBinary()
		require.NoError(t, err)
		var atr Trade
		err = atr.UnmarshalBinary(b)
		require.NoError(t, err)
		assert.ElementsMatch(t, aa, atr.AmountAsset)
		assert.ElementsMatch(t, pa, atr.PriceAsset)
		assert.ElementsMatch(t, id, atr.TransactionID)
		assert.ElementsMatch(t, ba, atr.Buyer)
		assert.ElementsMatch(t, sa, atr.Seller)
		assert.ElementsMatch(t, ma, atr.Matcher)
		assert.Equal(t, tc.price, atr.Price)
		assert.Equal(t, tc.amount, atr.Amount)
		assert.Equal(t, ts, atr.Timestamp)
		assert.Equal(t, tr, atr)
	}
}

func TestNewTradeInfo(t *testing.T) {
	for _, test := range []struct {
		tr     Trade
		aap    uint
		pap    uint
		price  string
		amount string
	}{
		{Trade{Price: 3422660097000000, Amount: 1}, 2, 8, "34.22660097", "0.01"},
		{Trade{Price: 22327, Amount: 14184001434}, 8, 8, "0.00022327", "141.84001434"},
	} {
		info := NewTradeInfo(test.tr, test.aap, test.pap)
		assert.Equal(t, test.price, info.Price.String())
		assert.Equal(t, test.amount, info.Amount.String())
	}
}
