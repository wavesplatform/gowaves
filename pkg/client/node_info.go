package client

import (
	"context"
	"net/http"
)

type NodeInfo struct {
	options Options
}

func NewNodeInfo(options Options) *NodeInfo {
	return &NodeInfo{
		options: options,
	}
}

// Version returns waves node version.
func (ni *NodeInfo) Version(ctx context.Context) (string, *Response, error) {
	type versionResponse struct {
		Version string `json:"version"`
	}
	url, err := joinUrl(ni.options.BaseUrl, "/node/version")
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}
	out := new(versionResponse)
	response, err := doHttp(ctx, ni.options, req, out)
	if err != nil {
		return "", response, err
	}
	return out.Version, response, nil
}
