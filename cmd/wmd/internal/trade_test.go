package internal

import (
	"github.com/stretchr/testify/assert"
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
		{"7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "35u3djrR6du2YDLwCkP1N1SXah4PkggQZVV3eGosXjiS", "6SkVLhHY79UcAaU1sbRWHcKcr8ipWrVK3et3kM4JN5v8", proto.Buy, "2sBjKeKCgTBYTpGARMKosU1uYJWct68RSZFHKvMzPieU", "5vRtEa2ygi3pAvE4xnypytJqM83Qsra6CTQNX9mtfD4m", "FbAq8kWEJjdzD7StCkhqfd4hrqf3P6ATju7usGgHVC14", 12345, 67890},
		{"2sBjKeKCgTBYTpGARMKosU1uYJWct68RSZFHKvMzPieU", "5vRtEa2ygi3pAvE4xnypytJqM83Qsra6CTQNX9mtfD4m", "FbAq8kWEJjdzD7StCkhqfd4hrqf3P6ATju7usGgHVC14", proto.Sell, "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "35u3djrR6du2YDLwCkP1N1SXah4PkggQZVV3eGosXjiS", "6SkVLhHY79UcAaU1sbRWHcKcr8ipWrVK3et3kM4JN5v8", 67890, 12345},
	}
	for _, tc := range tests {
		aa, _ := crypto.NewDigestFromBase58(tc.amountAsset)
		pa, _ := crypto.NewDigestFromBase58(tc.priceAsset)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		bpk, _ := crypto.NewPublicKeyFromBase58(tc.buyer)
		spk, _ := crypto.NewPublicKeyFromBase58(tc.seller)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tr := Trade{AmountAsset: aa, PriceAsset: pa, TransactionID: id, Buyer: bpk, Seller: spk, Matcher: mpk, Price: tc.price, Amount: tc.amount, Timestamp: ts}
		b, err := tr.MarshalBinary()
		assert.NoError(t, err)
		var atr Trade
		err = atr.UnmarshalBinary(b)
		assert.NoError(t, err)
		assert.ElementsMatch(t, aa, atr.AmountAsset)
		assert.ElementsMatch(t, pa, atr.PriceAsset)
		assert.ElementsMatch(t, id, atr.TransactionID)
		assert.ElementsMatch(t, bpk, atr.Buyer)
		assert.ElementsMatch(t, spk, atr.Seller)
		assert.ElementsMatch(t, mpk, atr.Matcher)
		assert.Equal(t, tc.price, atr.Price)
		assert.Equal(t, tc.amount, atr.Amount)
		assert.Equal(t, ts, atr.Timestamp)
		assert.Equal(t, tr, atr)
	}
}
