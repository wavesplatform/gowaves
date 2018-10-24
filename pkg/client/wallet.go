package client

import (
	"context"
	"fmt"
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

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/wallet/seed", a.options.BaseUrl),
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

	return out["seed"], response, nil
}
