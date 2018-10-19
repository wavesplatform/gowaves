package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Alias struct {
	options Options
}

func NewAlias(options Options) *Alias {
	return &Alias{
		options: options,
	}
}

func (a *Alias) Get(ctx context.Context, alias string) (string, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/alias/by-alias/%s", a.options.BaseUrl, alias),
		nil)
	if err != nil {
		return "", nil, err
	}

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["address"], response, nil
}

func (a *Alias) GetByAddress(ctx context.Context, address proto.Address) ([]*proto.Alias, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/alias/by-address/%s", a.options.BaseUrl, address.String()),
		nil)
	if err != nil {
		return nil, nil, err
	}

	var body []string
	response, err := doHttp(ctx, a.options, req, &body)
	if err != nil {
		return nil, response, err
	}

	var out []*proto.Alias
	for _, row := range body {
		alias, err := proto.NewAliasFromString(row)
		if err != nil {
			return nil, response, err
		}
		out = append(out, alias)
	}

	return out, response, nil
}
