package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/api"
	"net/http"
)

// Generators is a client wrapper for generator-related API endpoints.
type Generators struct {
	options Options
}

func NewGenerators(options Options) *Generators {
	return &Generators{
		options: options,
	}
}

// GeneratorsAtResponse is the expected structure returned by /generators/at/{height}.
type GeneratorsAtResponse struct {
	Height     uint64   `json:"height"`
	Generators []string `json:"generators"`
}

// CommitmentGeneratorsAt returns the list of committed generators for the given height.
func (a *Generators) CommitmentGeneratorsAt(ctx context.Context,
	height uint64) ([]api.GeneratorInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/generators/at/%d", height))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	var out []api.GeneratorInfo
	resp, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, resp, err
	}

	return out, resp, nil
}
