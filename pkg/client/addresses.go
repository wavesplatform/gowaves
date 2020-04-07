package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
	"strings"
)

type Addresses struct {
	options Options
}

// NewAddresses create new address block
func NewAddresses(options Options) *Addresses {
	return &Addresses{
		options: options,
	}
}

type AddressesBalance struct {
	Address       proto.Address `json:"address"`
	Confirmations uint64        `json:"confirmations"`
	Balance       uint64        `json:"balance"`
}

// Balance returns account's balance by its address
func (a *Addresses) Balance(ctx context.Context, address proto.Address) (*AddressesBalance, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("addresses/balance/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
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
	Address    proto.Address `json:"address"`
	Regular    uint64        `json:"regular"`
	Generating uint64        `json:"generating"`
	Available  uint64        `json:"available"`
	Effective  uint64        `json:"effective"`
}

// BalanceDetails returns account's detail balance by its address
func (a *Addresses) BalanceDetails(ctx context.Context, address proto.Address) (*AddressesBalanceDetails, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/balance/details/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
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
	Address    proto.Address `json:"address"`
	Complexity uint64        `json:"complexity"`
	ExtraFee   uint64        `json:"extraFee"`
}

// ScriptInfo gets account's script information
func (a *Addresses) ScriptInfo(ctx context.Context, address proto.Address) (*AddressesScriptInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/scriptInfo/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
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

// Get wallet accounts addresses
func (a *Addresses) Addresses(ctx context.Context) ([]proto.Address, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/addresses")
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []proto.Address
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesValidate struct {
	Address proto.Address `json:"address"`
	Valid   bool          `json:"valid"`
}

// Check whether address is valid or not
func (a *Addresses) Validate(ctx context.Context, address proto.Address) (*AddressesValidate, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/validate/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
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
	Address       proto.Address `json:"address"`
	Confirmations uint64        `json:"confirmations"`
	Balance       uint64        `json:"balance"`
}

// Account's balance
func (a *Addresses) EffectiveBalance(ctx context.Context, address proto.Address) (*AddressesEffectiveBalance, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/effectiveBalance/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
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

type addressesPublicKey struct {
	Address *proto.Address `json:"address"`
}

// Generate address from public key
func (a *Addresses) PublicKey(ctx context.Context, publicKey string) (*proto.Address, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/publicKey/%s", publicKey))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(addressesPublicKey)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	if out.Address == nil {
		return nil, response, &ParseError{Err: errors.New("failed parse address")}
	}

	return out.Address, response, nil
}

type AddressesSignText struct {
	Message   string           `json:"message"`
	PublicKey crypto.PublicKey `json:"publicKey"`
	Signature crypto.Signature `json:"signature"`
}

// Sign a message with a private key associated with address
func (a *Addresses) SignText(ctx context.Context, address proto.Address, message string) (*AddressesSignText, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/signText/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
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

type VerifyText struct {
	Valid bool
}

type VerifyTextReq struct {
	Message   string           `json:"message"`
	PublicKey crypto.PublicKey `json:"publickey"`
	Signature crypto.Signature `json:"signature"`
}

// Check a signature of a message signed by an account
func (a *Addresses) VerifyText(ctx context.Context, address proto.Address, body VerifyTextReq) (bool, *Response, error) {
	if a.options.ApiKey == "" {
		return false, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/verifyText/%s", address.String()))
	if err != nil {
		return false, nil, err
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return false, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return false, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(VerifyText)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return false, response, err
	}

	return out.Valid, response, nil

}

type BalanceAfterConfirmations struct {
	Address       proto.Address `json:"address"`
	Confirmations uint64        `json:"confirmations"`
	Balance       uint64        `json:"balance"`
}

// Balance of address after confirmations
func (a *Addresses) BalanceAfterConfirmations(
	ctx context.Context, address proto.Address, confirmations uint64) (*BalanceAfterConfirmations, *Response, error) {

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/balance/%s/%d", address.String(), confirmations))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(BalanceAfterConfirmations)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
