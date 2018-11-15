package client

import (
	"bytes"
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Debug struct {
	options Options
}

func NewDebug(options Options) *Debug {
	return &Debug{
		options: options,
	}
}

type DebugInfo struct {
	StateHeight              uint64 `json:"stateHeight"`
	ExtensionLoaderState     string `json:"extensionLoaderState"`
	HistoryReplierCacheSizes struct {
		Blocks      uint64 `json:"blocks"`
		MicroBlocks uint64 `json:"microBlocks"`
	} `json:"historyReplierCacheSizes"`
	MicroBlockSynchronizerCacheSizes struct {
		MicroBlockOwners     uint64 `json:"microBlockOwners"`
		NextInvs             uint64 `json:"nextInvs"`
		Awaiting             uint64 `json:"awaiting"`
		SuccessfullyReceived uint64 `json:"successfullyReceived"`
	} `json:"microBlockSynchronizerCacheSizes"`
	ScoreObserverStats struct {
		LocalScore         LocalScore `json:"localScore"`
		CurrentBestChannel string     `json:"currentBestChannel"`
		ScoresCacheSize    uint64     `json:"scoresCacheSize"`
	} `json:"scoreObserverStats"`
	MinerState string `json:"minerState"`
}

type LocalScore string

func (a *LocalScore) UnmarshalJSON(data []byte) error {
	*a = LocalScore(data)
	return nil
}

// All info you need to debug
func (a *Debug) Info(ctx context.Context) (*DebugInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/info")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(DebugInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Get sizes and full hashes for last blocks
func (a *Debug) Blocks(ctx context.Context, howMany uint64) ([]map[uint64]string, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/blocks/%d", howMany))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	var out []map[uint64]string
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type DebugMinerInfo struct {
	Address       proto.Address `json:"address"`
	MiningBalance uint64        `json:"miningBalance"`
	Timestamp     uint64        `json:"timestamp"`
}

// All miner info you need to debug
func (a *Debug) MinerInfo(ctx context.Context) ([]*DebugMinerInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/minerInfo")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	var out []*DebugMinerInfo
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type DebugHistoryInfo struct {
	LastBlockIds  []crypto.Signature `json:"lastBlockIds"`
	MicroBlockIds []crypto.Signature `json:"microBlockIds"`
}

// All history info you need to debug
func (a *Debug) HistoryInfo(ctx context.Context) (*DebugHistoryInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/historyInfo")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	out := new(DebugHistoryInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Currently running node config
func (a *Debug) ConfigInfo(ctx context.Context, full bool) ([]byte, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/configInfo?full=%t", full))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}

	return buf.Bytes(), response, nil
}
