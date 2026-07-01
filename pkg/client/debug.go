package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

// Info returns all info you need to debug.
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
	response, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type DebugMinerInfo struct {
	Address       proto.WavesAddress `json:"address"`
	MiningBalance uint64             `json:"miningBalance"`
	Timestamp     uint64             `json:"timestamp"`
}

// MinerInfo gets all miner info you need to debug.
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
	response, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// ConfigInfo currently running node config.
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
	response, err := doHTTP(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}

	return buf.Bytes(), response, nil
}

// stateHashV2Diff is used to detect whether the requested StateHash is V1 or V2.
// If the GeneratorsHash is zero, then it's V1.
type stateHashV2Diff struct {
	GeneratorsHash proto.DigestWrapped `json:"nextCommittedGeneratorsHash"`
}

func (diff *stateHashV2Diff) isZero() bool {
	return crypto.Digest(diff.GeneratorsHash) == crypto.Digest{}
}

// stateHashDebugV2Diff is used to detect whether the requested StateHashDebug is V1 or V2.
// If the GeneratorsHash is zero and BaseTarget is zero, then it's V1.
type stateHashDebugV2Diff struct {
	GeneratorsHash proto.DigestWrapped `json:"nextCommittedGeneratorsHash"`
	BaseTarget     uint64              `json:"baseTart"`
}

func (diff *stateHashDebugV2Diff) isZero() bool {
	return crypto.Digest(diff.GeneratorsHash) == crypto.Digest{} && diff.BaseTarget == 0
}

func (a *Debug) StateHash(ctx context.Context, height uint64) (proto.StateHash, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/stateHash/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	buf := new(bytes.Buffer)
	response, err := doHTTP(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	var diff stateHashV2Diff
	if umErr := json.Unmarshal(buf.Bytes(), &diff); umErr != nil {
		return nil, response, umErr
	}
	var out proto.StateHash = new(proto.StateHashV2)
	if diff.isZero() {
		out = new(proto.StateHashV1)
	}
	if umErr := json.Unmarshal(buf.Bytes(), &out); umErr != nil {
		return nil, response, umErr
	}
	return out, response, nil
}

func (a *Debug) stateHashDebugAtPath(ctx context.Context, path string) (proto.StateHashDebug, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, path)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	buf := new(bytes.Buffer)
	response, err := doHTTP(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	var diff stateHashDebugV2Diff
	if umErr := json.Unmarshal(buf.Bytes(), &diff); umErr != nil {
		return nil, response, umErr
	}
	var out proto.StateHashDebug = new(proto.StateHashDebugV2)
	if diff.isZero() {
		out = new(proto.StateHashDebugV1)
	}
	if umErr := json.Unmarshal(buf.Bytes(), &out); umErr != nil {
		return nil, response, umErr
	}
	return out, response, nil
}

func (a *Debug) StateHashDebug(ctx context.Context, height uint64) (proto.StateHashDebug, *Response, error) {
	return a.stateHashDebugAtPath(ctx, fmt.Sprintf("/debug/stateHash/%d", height))
}

func (a *Debug) StateHashDebugLast(ctx context.Context) (proto.StateHashDebug, *Response, error) {
	return a.stateHashDebugAtPath(ctx, "/debug/stateHash/last")
}

type BalancesHistoryRow struct {
	Height  uint64 `json:"height"`
	Balance uint64 `json:"balance"`
}

func (a *Debug) BalancesHistory(ctx context.Context, address proto.WavesAddress) ([]*BalancesHistoryRow, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/balances/history/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*BalancesHistoryRow
	response, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Debug) PrintMsg(ctx context.Context, msg string) (*Response, error) {
	type printMsgRequestBody struct {
		Message string `json:"message"`
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/print")
	if err != nil {
		return nil, err
	}
	bts, err := json.Marshal(printMsgRequestBody{Message: msg})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(bts))
	if err != nil {
		return nil, err
	}
	req.Header.Add(ApiKeyHeader, a.options.ApiKey)
	req.Header.Add("Accept", "*/*")

	return doHTTP(ctx, a.options, req, nil)
}

type rollbackResponse struct {
	BlockID proto.BlockID `json:"blockId"`
}

func (a *Debug) RollbackToHeight(
	ctx context.Context,
	height uint64,
	returnTransactionsToUtx bool,
) (*proto.BlockID, *Response, error) {
	type rollbackRequestBody struct {
		Height                  uint64 `json:"rollbackTo"`
		ReturnTransactionsToUtx bool   `json:"returnTransactionsToUtx"`
	}
	url, err := joinUrl(a.options.BaseUrl, "/debug/rollback")
	if err != nil {
		return nil, nil, err
	}
	bts, err := json.Marshal(rollbackRequestBody{Height: height, ReturnTransactionsToUtx: returnTransactionsToUtx})
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), bytes.NewBuffer(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add(ApiKeyHeader, a.options.ApiKey)
	req.Header.Add("Accept", "*/*")

	out := new(rollbackResponse)
	response, err := doHTTP(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}
	return &out.BlockID, response, nil
}

func (a *Debug) RollbackTo(ctx context.Context, blockID proto.BlockID) (*proto.BlockID, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/rollback-to/%s", blockID.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add(ApiKeyHeader, a.options.ApiKey)
	req.Header.Add("Accept", "*/*")

	out := new(rollbackResponse)
	response, err := doHTTP(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return &out.BlockID, response, nil
}
