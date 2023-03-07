package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Alias struct {
	options Options
}

func NewAlias(options Options) *Alias {
	return &Alias{
		options: options,
	}
}

func (a *Alias) Get(ctx context.Context, alias string) (proto.WavesAddress, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/alias/by-alias/%s", alias))
	if err != nil {
		return proto.WavesAddress{}, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return proto.WavesAddress{}, nil, err
	}

	out := struct {
		Address proto.WavesAddress `json:"address"`
	}{}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return proto.WavesAddress{}, response, err
	}

	return out.Address, response, nil
}

func (a *Alias) GetByAddress(ctx context.Context, address proto.WavesAddress) ([]*proto.Alias, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/alias/by-address/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*proto.Alias
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
