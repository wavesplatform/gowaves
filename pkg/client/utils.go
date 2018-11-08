package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"net/http"
	"strings"
)

type Utils struct {
	options Options
}

// returns new utils
func NewUtils(options Options) *Utils {
	return &Utils{
		options: options,
	}
}

// Generate random seed
func (a *Utils) Seed(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/utils/seed", a.options.BaseUrl),
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

type UtilsHashSecure struct {
	Message string `json:"message"`
	Hash    string `json:"hash"`
}

// Return SecureCryptographicHash of specified message
func (a *Utils) HashSecure(ctx context.Context, message string) (*UtilsHashSecure, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/utils/hash/secure", a.options.BaseUrl),
		strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(UtilsHashSecure)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type UtilsHashFast struct {
	Message string `json:"message"`
	Hash    string `json:"hash"`
}

// Return FastCryptographicHash of specified message
func (a *Utils) HashFast(ctx context.Context, message string) (*UtilsHashFast, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/utils/hash/fast", a.options.BaseUrl),
		strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(UtilsHashFast)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type UtilsTime struct {
	System uint64 `json:"system"`
	NTP    uint64 `json:"NTP"`
}

// Current Node time (UTC)
func (a *Utils) Time(ctx context.Context) (*UtilsTime, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/utils/time", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(UtilsTime)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type UtilsSign struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// Return FastCryptographicHash of specified message
func (a *Utils) Sign(ctx context.Context, secretKey crypto.SecretKey, message string) (*UtilsSign, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/utils/sign/%s", a.options.BaseUrl, secretKey.String()),
		strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(UtilsSign)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Generate random seed of specified length
func (a *Utils) SeedByLength(ctx context.Context, length uint16) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/utils/seed/%d", a.options.BaseUrl, length),
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

type UtilsScriptCompile struct {
	Script     string `json:"script"`
	Complexity uint64 `json:"complexity"`
	ExtraFee   uint64 `json:"extraFee"`
}

// Compiles string code to base64 script representation
func (a *Utils) ScriptCompile(ctx context.Context, code string) (*UtilsScriptCompile, *Response, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/utils/script/compile", a.options.BaseUrl),
		strings.NewReader(code))
	if err != nil {
		return nil, nil, err
	}

	out := new(UtilsScriptCompile)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type UtilsScriptEstimate struct {
	Script     string `json:"script"`
	ScriptText string `json:"scriptText"`
	Complexity uint64 `json:"complexity"`
	ExtraFee   uint64 `json:"extraFee"`
}

// Estimates compiled code in Base64 representation
func (a *Utils) ScriptEstimate(ctx context.Context, base64code string) (*UtilsScriptEstimate, *Response, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/utils/script/estimate", a.options.BaseUrl),
		strings.NewReader(base64code))
	if err != nil {
		return nil, nil, err
	}

	out := new(UtilsScriptEstimate)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
