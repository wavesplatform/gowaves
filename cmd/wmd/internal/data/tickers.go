package data

import (
	"bytes"
	"math"
	"math/big"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AssetID crypto.Digest

func (v AssetID) MarshalJSON() ([]byte, error) {
	if bytes.Equal(v[:], WavesID[:]) {
		return []byte("\"WAVES\""), nil
	}
	return crypto.Digest(v).MarshalJSON()
}

type TickerInfo struct {
	Symbol                       string          `json:"symbol"`
	AmountAssetID                AssetID         `json:"amountAssetID"`
	AmountAssetName              string          `json:"amountAssetName"`
	AmountAssetDecimals          byte            `json:"amountAssetDecimals"`
	AmountAssetTotalSupply       Decimal         `json:"amountAssetTotalSupply"`
	AmountAssetMaxSupply         InfiniteDecimal `json:"amountAssetMaxSupply"`
	AmountAssetCirculatingSupply Decimal         `json:"amountAssetCirculatingSupply"`
	PriceAssetID                 AssetID         `json:"priceAssetID"`
	PriceAssetName               string          `json:"priceAssetName"`
	PriceAssetDecimals           byte            `json:"priceAssetDecimals"`
	PriceAssetTotalSupply        Decimal         `json:"priceAssetTotalSupply"`
	PriceAssetMaxSupply          InfiniteDecimal `json:"priceAssetMaxSupply"`
	PriceAssetCirculatingSupply  Decimal         `json:"priceAssetCirculatingSupply"`
	DayOpen                      Decimal         `json:"24h_open"`
	DayHigh                      Decimal         `json:"24h_high"`
	DayLow                       Decimal         `json:"24h_low"`
	DayClose                     Decimal         `json:"24h_close"`
	DayVWAP                      Decimal         `json:"24h_vwap"`
	DayVolume                    Decimal         `json:"24h_volume"`
	DayPriceVolume               Decimal         `json:"24h_priceVolume"`
	Timestamp                    uint64          `json:"timestamp"`
}

func NewTickerInfo(symbol string, amountAsset, priceAsset AssetInfo, amountAssetIssuerBalance, priceAssetIssuerBalance uint64, candle Candle) TickerInfo {
	as := int64(math.Pow10(int(amountAsset.Decimals)))
	x := new(big.Int).SetUint64(candle.Average)
	y := new(big.Int).SetUint64(candle.Volume)
	z := big.NewInt(as)
	xy := x.Mul(x, y)
	pv := xy.Div(xy, z).Uint64()
	ts := uint64(time.Now().UnixNano() / 1000000)
	scale := uint(priceAsset.Decimals - amountAsset.Decimals + 8)
	aaSupply := Decimal{value: amountAsset.Supply, scale: uint(amountAsset.Decimals)}
	aaMaxSupply := aaSupply.ToInfiniteDecimal(amountAsset.Reissuable)
	aaCirculatingSupply := Decimal{value: amountAsset.Supply - amountAssetIssuerBalance, scale: uint(amountAsset.Decimals)}
	paSupply := Decimal{value: priceAsset.Supply, scale: uint(priceAsset.Decimals)}
	paMaxSupply := paSupply.ToInfiniteDecimal(priceAsset.Reissuable)
	paCirculatingSupply := Decimal{value: priceAsset.Supply - priceAssetIssuerBalance, scale: uint(priceAsset.Decimals)}
	return TickerInfo{
		Symbol:                       symbol,
		AmountAssetID:                AssetID(amountAsset.ID),
		AmountAssetName:              amountAsset.Name,
		AmountAssetDecimals:          amountAsset.Decimals,
		AmountAssetTotalSupply:       aaSupply,
		AmountAssetMaxSupply:         aaMaxSupply,
		AmountAssetCirculatingSupply: aaCirculatingSupply,
		PriceAssetID:                 AssetID(priceAsset.ID),
		PriceAssetName:               priceAsset.Name,
		PriceAssetDecimals:           priceAsset.Decimals,
		PriceAssetTotalSupply:        paSupply,
		PriceAssetMaxSupply:          paMaxSupply,
		PriceAssetCirculatingSupply:  paCirculatingSupply,
		DayOpen:                      Decimal{candle.Open, scale},
		DayHigh:                      Decimal{candle.High, scale},
		DayLow:                       Decimal{candle.Low, scale},
		DayClose:                     Decimal{candle.Close, scale},
		DayVWAP:                      Decimal{candle.Average, scale},
		DayVolume:                    Decimal{candle.Volume, uint(amountAsset.Decimals)},
		DayPriceVolume:               Decimal{pv, scale},
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
