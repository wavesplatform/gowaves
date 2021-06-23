package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Leasing struct {
	options Options
}

// NewLeasing creates new leasing.
func NewLeasing(options Options) *Leasing {
	return &Leasing{
		options: options,
	}
}

// Active gets lease transactions.
func (a *Leasing) Active(ctx context.Context, address proto.WavesAddress) ([]*proto.LeaseWithSig, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/leasing/active/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*proto.LeaseWithSig
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
