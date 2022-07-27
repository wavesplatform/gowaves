package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	Address       proto.WavesAddress `json:"address"`
	Confirmations uint64             `json:"confirmations"`
	Balance       uint64             `json:"balance"`
}

// Balance returns account's balance by its address
func (a *Addresses) Balance(ctx context.Context, address proto.WavesAddress) (*AddressesBalance, *Response, error) {
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
	Address    proto.WavesAddress `json:"address"`
	Regular    uint64             `json:"regular"`
	Generating uint64             `json:"generating"`
	Available  uint64             `json:"available"`
	Effective  uint64             `json:"effective"`
}

// BalanceDetails returns account's detail balance by its address
func (a *Addresses) BalanceDetails(ctx context.Context, address proto.WavesAddress) (*AddressesBalanceDetails, *Response, error) {
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
	Address              proto.WavesAddress `json:"address"`
	Script               string             `json:"script"`
	ScriptText           string             `json:"scriptText"`
	Version              uint64             `json:"version"`
	Complexity           uint64             `json:"complexity"`
	VerifierComplexity   uint64             `json:"verifierComplexity"`
	CallableComplexities map[string]uint64  `json:"callableComplexities"`
	ExtraFee             uint64             `json:"extraFee"`
}

// ScriptInfo gets account's script information
func (a *Addresses) ScriptInfo(ctx context.Context, address proto.WavesAddress) (*AddressesScriptInfo, *Response, error) {
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

// Addresses gets wallet accounts addresses
func (a *Addresses) Addresses(ctx context.Context) ([]proto.WavesAddress, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/addresses")
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []proto.WavesAddress
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AddressesValidate struct {
	Address proto.WavesAddress `json:"address"`
	Valid   bool               `json:"valid"`
}

// Validate checks whether address is valid or not
func (a *Addresses) Validate(ctx context.Context, address proto.WavesAddress) (*AddressesValidate, *Response, error) {
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
	Address       proto.WavesAddress `json:"address"`
	Confirmations uint64             `json:"confirmations"`
	Balance       uint64             `json:"balance"`
}

// EffectiveBalance gets account's balance
func (a *Addresses) EffectiveBalance(ctx context.Context, address proto.WavesAddress) (*AddressesEffectiveBalance, *Response, error) {
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
	Address *proto.WavesAddress `json:"address"`
}

// PublicKey generates address from public key
func (a *Addresses) PublicKey(ctx context.Context, publicKey string) (*proto.WavesAddress, *Response, error) {
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

// SignText signs a message with a private key associated with address
func (a *Addresses) SignText(ctx context.Context, address proto.WavesAddress, message string) (*AddressesSignText, *Response, error) {
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

// VerifyText checks a signature of a message signed by an account
func (a *Addresses) VerifyText(ctx context.Context, address proto.WavesAddress, body VerifyTextReq) (bool, *Response, error) {
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
	Address       proto.WavesAddress `json:"address"`
	Confirmations uint64             `json:"confirmations"`
	Balance       uint64             `json:"balance"`
}

// BalanceAfterConfirmations returns balance of an address after given number of confirmations.
func (a *Addresses) BalanceAfterConfirmations(
	ctx context.Context, address proto.WavesAddress, confirmations uint64) (*BalanceAfterConfirmations, *Response, error) {

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

// AddressesData returns all data entries for given address
func (a *Addresses) AddressesData(ctx context.Context, address proto.WavesAddress) (proto.DataEntries, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/data/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(proto.DataEntries)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}
	return *out, response, nil
}

// AddressesDataKey returns data entry for given address and key
func (a *Addresses) AddressesDataKey(ctx context.Context, address proto.WavesAddress, key string) (proto.DataEntry, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/data/%s/%s", address.String(), key))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	buff := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buff)
	if err != nil {
		return nil, response, err
	}

	out, err := proto.NewDataEntryFromJSON(buff.Bytes())
	if err != nil {
		return nil, response, err
	}
	return out, response, nil
}

// AddressesDataKeys returns data entry for given address and keys
func (a *Addresses) AddressesDataKeys(ctx context.Context, address proto.WavesAddress, keys []string) (proto.DataEntries, *Response, error) {
	type addressesDataKeys struct {
		Keys []string `json:"keys"`
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/addresses/data/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	b := new(bytes.Buffer)
	if err = json.NewEncoder(b).Encode(addressesDataKeys{Keys: keys}); err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", url.String(), b)
	if err != nil {
		return nil, nil, err
	}

	out := new(proto.DataEntries)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}
	return *out, response, nil
}
