package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Peers struct {
	options Options
}

func NewPeers(options Options) *Peers {
	return &Peers{
		options: options,
	}
}

type PeerAllRow struct {
	Address  proto.PeerInfo
	LastSeen Timestamp `json:"lastSeen"`
}

type peersAllResp struct {
	Peers []*PeerAllRow `json:"peers"`
}

func (a *Peers) All(ctx context.Context) ([]*PeerAllRow, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/peers/all", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(peersAllResp)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out.Peers, response, nil
}

type peersConnected struct {
	Peers []*PeersConnectedRow `json:"peers"`
}

type PeersConnectedRow struct {
	Address            proto.PeerInfo `json:"address"`
	DeclaredAddress    proto.PeerInfo `json:"declaredAddress"`
	PeerName           string         `json:"peerName"`
	PeerNonce          uint64         `json:"peerNonce"`
	ApplicationName    string         `json:"applicationName"`
	ApplicationVersion string         `json:"applicationVersion"`
}

func (a *Peers) Connected(ctx context.Context) ([]*PeersConnectedRow, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/peers/connected", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(peersConnected)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out.Peers, response, nil
}

type PeersBlacklistedRow struct {
	Hostname  proto.PeerInfo `json:"hostname"`
	Timestamp Timestamp      `json:"timestamp"`
	Reason    string         `json:"reason"`
}

func (a *Peers) Blacklisted(ctx context.Context) ([]*PeersBlacklistedRow, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/peers/blacklisted", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	var out []*PeersBlacklistedRow
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type PeersSuspendedRow struct {
	Hostname  proto.PeerInfo `json:"hostname"`
	Timestamp Timestamp      `json:"timestamp"`
}

func (a *Peers) Suspended(ctx context.Context) ([]*PeersSuspendedRow, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/peers/suspended", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	var out []*PeersSuspendedRow
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type PeersConnect struct {
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
}

func (a *Peers) Connect(ctx context.Context, host string, port uint16) (*PeersConnect, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	bts, err := json.Marshal(map[string]interface{}{"host": host, "port": port})
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/peers/connect", a.options.BaseUrl),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(PeersConnect)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Peers) ClearBlacklist(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/peers/clearblacklist", a.options.BaseUrl),
		nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["result"], response, nil
}
