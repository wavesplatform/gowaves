package internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"math"
	"math/big"
	"time"
)

type TickerInfo struct {
	Symbol                       string        `json:"symbol"`
	AmountAssetID                crypto.Digest `json:"amountAssetID"`
	AmountAssetName              string        `json:"amountAssetName"`
	AmountAssetDecimals          byte          `json:"amountAssetDecimals"`
	AmountAssetTotalSupply       Decimal       `json:"amountAssetTotalSupply"`
	AmountAssetMaxSupply         Decimal       `json:"amountAssetMaxSupply"`
	AmountAssetCirculatingSupply Decimal       `json:"amountAssetCirculatingSupply"`
	PriceAssetID                 crypto.Digest `json:"priceAssetID"`
	PriceAssetName               string        `json:"priceAssetName"`
	PriceAssetDecimals           byte          `json:"priceAssetDecimals"`
	PriceAssetTotalSupply        Decimal       `json:"priceAssetTotalSupply"`
	PriceAssetMaxSupply          Decimal       `json:"priceAssetMaxSupply"`
	PriceAssetCirculatingSupply  Decimal       `json:"priceAssetCirculatingSupply"`
	DayOpen                      Decimal       `json:"24h_open"`
	DayHigh                      Decimal       `json:"24h_high"`
	DayLow                       Decimal       `json:"24h_low"`
	DayClose                     Decimal       `json:"24h_close"`
	DayVWAP                      Decimal       `json:"24h_vwap"`
	DayVolume                    Decimal       `json:"24h_volume"`
	DayPriceVolume               Decimal       `json:"24h_priceVolume"`
	Timestamp                    uint64        `json:"timestamp"`
}

func NewTickerInfo(symbol string, amountAsset, priceAsset AssetInfo, candle Candle) TickerInfo {
	as := int64(math.Pow10(int(amountAsset.Decimals)))
	x := big.NewInt(0).SetUint64(candle.Average)
	y := big.NewInt(0).SetUint64(candle.Volume)
	z := big.NewInt(as)
	xy := x.Mul(x, y)
	pv := xy.Div(xy, z).Uint64()
	ts := uint64(time.Now().UnixNano() / 1000000)
	return TickerInfo{
		Symbol:                       symbol,
		AmountAssetID:                amountAsset.ID,
		AmountAssetName:              amountAsset.Name,
		AmountAssetDecimals:          amountAsset.Decimals,
		AmountAssetTotalSupply:       Decimal{0, uint(amountAsset.Decimals)},
		AmountAssetMaxSupply:         Decimal{0, uint(amountAsset.Decimals)},
		AmountAssetCirculatingSupply: Decimal{0, uint(amountAsset.Decimals)},
		PriceAssetID:                 priceAsset.ID,
		PriceAssetName:               priceAsset.Name,
		PriceAssetDecimals:           priceAsset.Decimals,
		PriceAssetTotalSupply:        Decimal{0, uint(priceAsset.Decimals)},
		PriceAssetMaxSupply:          Decimal{0, uint(priceAsset.Decimals)},
		PriceAssetCirculatingSupply:  Decimal{0, uint(priceAsset.Decimals)},
		DayOpen:                      Decimal{candle.Open, uint(priceAsset.Decimals)},
		DayHigh:                      Decimal{candle.High, uint(priceAsset.Decimals)},
		DayLow:                       Decimal{candle.Low, uint(priceAsset.Decimals)},
		DayClose:                     Decimal{candle.Close, uint(priceAsset.Decimals)},
		DayVWAP:                      Decimal{candle.Average, uint(priceAsset.Decimals)},
		DayVolume:                    Decimal{candle.Volume, uint(amountAsset.Decimals)},
		DayPriceVolume:               Decimal{pv, uint(priceAsset.Decimals)},
		Timestamp:                    ts,
	}
}

type ByTickers []TickerInfo

func (a ByTickers) Len() int {
	return len(a)
}

func (a ByTickers) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByTickers) Less(i, j int) bool {
	x := a[i].Symbol
	y := a[j].Symbol
	switch {
	case x == "" && y != "":
		return false
	case x != "" && y == "":
		return true
	case x != "" && y != "":
		return x < y
	default:
		return false
	}
}
