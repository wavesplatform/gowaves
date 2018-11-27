package internal

import "github.com/wavesplatform/gowaves/pkg/crypto"

type MarketID struct {
	AmountAsset crypto.Digest
	PriceAsset crypto.Digest
}

type MarketData struct {
	
}

type MarketInfo struct {
	Symbol string `json:"symbol"`
	AmountAssetID crypto.Digest `json:"amountAssetID"`
	AmountAssetName string `json:"amountAssetName"`
	AmountAssetDecimals int `json:"amountAssetDecimals"`
	AmountAssetTotalSupply string `json:"amountAssetTotalSupply"`
	AmountAssetMaxSupply string `json:"amountAssetMaxSupply"`
	AmountAssetCirculatingSupply string `json:"amountAssetCirculatingSupply"`
	PriceAssetID crypto.Digest `json:"priceAssetID"`
	PriceAssetName string `json:"priceAssetName"`
	PriceAssetDecimals int `json:"priceAssetDecimals"`
	PriceAssetTotalSupply string `json:"priceAssetTotalSupply"`
	PriceAssetMaxSupply string `json:"priceAssetMaxSupply"`
	PriceAssetCirculatingSupply string `json:"priceAssetCirculatingSupply"`
	Open string `json:"24h_open"`
	High string `json:"24h_high"`
	Low string `json:"24h_low"`
	Close string `json:"24h_close"`
	Average string `json:"24h_vwap"`
	Volume string `json:"24h_volume"`
	PriceVolume string `json:"24h_priceVolume"`
	TotalTrades int `json:"totalTrades"`
	FirstTradeDay uint64 `json:"firstTradeDay"`
	LastTradeDay uint64 `json:"lastTradeDay"`
}

type Markets struct {
	pairs map[MarketID] MarketData
}