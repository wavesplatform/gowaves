package client

import (
	"context"
	"net/http"
)

type Wallet struct {
	options Options
}

func NewWallet(options Options) *Wallet {
	return &Wallet{
		options: options,
	}
}

func (a *Wallet) Seed(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/wallet/seed")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	var out struct {
		Seed string `json:"seed"`
	}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out.Seed, response, nil
}
