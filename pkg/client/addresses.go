package client

import (
	"context"
	"fmt"
	"net/http"
)

type AddressesBalance struct {
	Address       string `json:"address"`
	Confirmations uint64 `json:"confirmations"`
	Balance       uint64 `json:"balance"`
}

func (a *Client) GetAddressesBalance(ctx context.Context, address string) (*AddressesBalance, *Response, error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/balance/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesBalance)
	response, err := a.Do(ctx, req, out)
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

func (a *Client) GetAddressesBalanceDetails(ctx context.Context, address string) (*AddressesBalanceDetails, *Response, error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/balance/details/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesBalanceDetails)
	response, err := a.Do(ctx, req, out)
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

func (a *Client) GetAddressesScriptInfo(ctx context.Context, address string) (*AddressesScriptInfo, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/scriptInfo/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesScriptInfo)
	response, err := a.Do(ctx, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Client) GetAddresses(ctx context.Context) ([]string, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	var out []string
	response, err := a.Do(ctx, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesValidate struct {
	Address string `json:"address"`
	Valid   bool   `json:"valid"`
}

func (a *Client) GetAddressesValidate(ctx context.Context, address string) (*AddressesValidate, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/validate/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesValidate)
	response, err := a.Do(ctx, req, out)
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

func (a *Client) GetAddressesEffectiveBalance(ctx context.Context, address string) (*AddressesEffectiveBalance, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/effectiveBalance/%s", a.options.BaseUrl, address),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesEffectiveBalance)
	response, err := a.Do(ctx, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesPublicKey struct {
	Address string `json:"address"`
}

func (a *Client) GetAddressesPublicKey(ctx context.Context, publicKey string) (*AddressesPublicKey, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/addresses/publicKey/%s", a.options.BaseUrl, publicKey),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AddressesPublicKey)
	response, err := a.Do(ctx, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
