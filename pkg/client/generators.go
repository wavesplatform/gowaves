package client

import (
	"context"
	"fmt"
	"net/http"

	nodeApi "github.com/wavesplatform/gowaves/pkg/api"
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

// CommitmentGeneratorsAt returns the list of committed generators for the given height.
func (a *Generators) CommitmentGeneratorsAt(ctx context.Context,
	height uint64) ([]nodeApi.GeneratorInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/generators/at/%d", height))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	var out []nodeApi.GeneratorInfo
	resp, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, resp, err
	}

	return out, resp, nil
}
