package client

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

type Addresses struct {
	options Options
}

func NewAddresses(options Options) *Addresses {
	return &Addresses{
		options: options,
	}
}

type AddressesBalance struct {
	Address       string `json:"address"`
	Confirmations uint64 `json:"confirmations"`
	Balance       uint64 `json:"balance"`
}

func (a *Addresses) Balance(ctx context.Context, address string) (*AddressesBalance, *Response, error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/balance/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesBalance)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil

}

type AddressesBalanceDetails struct {
	Address    string `json:"address"`
	Regular    uint64 `json:"regular"`
	Generating uint64 `json:"generating"`
	Available  uint64 `json:"available"`
	Effective  uint64 `json:"effective"`
}

func (a *Addresses) BalanceDetails(ctx context.Context, address string) (*AddressesBalanceDetails, *Response, error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/balance/details/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesBalanceDetails)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil

}

type AddressesScriptInfo struct {
	Address    string `json:"address"`
	Complexity uint64 `json:"complexity"`
	ExtraFee   uint64 `json:"extra_fee"`
}

func (a *Addresses) GetScriptInfo(ctx context.Context, address string) (*AddressesScriptInfo, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/scriptInfo/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesScriptInfo)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Addresses) Addresses(ctx context.Context) ([]string, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	var out []string
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesValidate struct {
	Address string `json:"address"`
	Valid   bool   `json:"valid"`
}

func (a *Addresses) Validate(ctx context.Context, address string) (*AddressesValidate, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/validate/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesValidate)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesEffectiveBalance struct {
	Address       string `json:"address"`
	Confirmations uint64 `json:"confirmations"`
	Balance       uint64 `json:"balance"`
}

func (a *Addresses) EffectiveBalance(ctx context.Context, address string) (*AddressesEffectiveBalance, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/effectiveBalance/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesEffectiveBalance)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesPublicKey struct {
	Address string `json:"address"`
}

func (a *Addresses) PublicKey(ctx context.Context, publicKey string) (*AddressesPublicKey, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/publicKey/%s", a.options.BaseUrl, publicKey),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesPublicKey)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesSignText struct {
	Message   string `json:"message"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}

func (a *Addresses) SignText(ctx context.Context, address string, message string) (*AddressesSignText, *Response, error) {

	if a.options.ApiKey == "" {
		return nil, nil, errors.New("no api key provided")
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/addresses/signText/%s", a.options.BaseUrl, address),
		strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(AddressesSignText)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
