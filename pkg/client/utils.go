package client

import (
	"context"
	"fmt"
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

// Seed returns generated random seed. The returned value is base58 encoded.
func (a *Utils) Seed(ctx context.Context) (string, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/utils/seed")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	var out struct {
		Seed string `json:"seed"`
	}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out.Seed, response, nil
}

type UtilsHashSecure struct {
	Message string `json:"message"`
	Hash    string `json:"hash"`
}

// HashSecure returns the Keccak-256 hash of the BLAKE2b-256 hash of a given message.
func (a *Utils) HashSecure(ctx context.Context, message string) (*UtilsHashSecure, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/utils/hash/secure")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", url.String(), strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

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

// HashFast returns the BLAKE2b-256 hash of a given message.
func (a *Utils) HashFast(ctx context.Context, message string) (*UtilsHashFast, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/utils/hash/fast")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", url.String(), strings.NewReader(message))
	if err != nil {
		return nil, nil, err
	}

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

// Time returns the current node time (UTC).
func (a *Utils) Time(ctx context.Context) (*UtilsTime, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/utils/time")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(UtilsTime)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// SeedByLength returns generated random seed of a given length in bytes. The returned value is base58 encoded
func (a *Utils) SeedByLength(ctx context.Context, length uint16) (string, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/utils/seed/%d", length))
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	var out struct {
		Seed string `json:"seed"`
	}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out.Seed, response, nil
}

type UtilsScriptCompile struct {
	Script               string            `json:"script"`
	Complexity           uint64            `json:"complexity"`
	VerifierComplexity   uint64            `json:"verifierComplexity"`
	ExtraFee             uint64            `json:"extraFee"`
	CallableComplexities map[string]uint64 `json:"callableComplexities"`
}

// ScriptCompile returns compiled base64 script representation without compaction from a given code.
// Deprecated: use ScriptCompileCode.
func (a *Utils) ScriptCompile(ctx context.Context, code string) (*UtilsScriptCompile, *Response, error) {
	return a.ScriptCompileCode(ctx, code, false)
}

// ScriptCompileCode returns compiled base64 script representation from a given code.
func (a *Utils) ScriptCompileCode(ctx context.Context, code string, compaction bool) (*UtilsScriptCompile, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/utils/script/compileCode?compact=%t", compaction))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", url.String(), strings.NewReader(code))
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
	Script               string            `json:"script"`
	ScriptText           string            `json:"scriptText"`
	Complexity           uint64            `json:"complexity"`
	VerifierComplexity   uint64            `json:"verifierComplexity"`
	ExtraFee             uint64            `json:"extraFee"`
	CallableComplexities map[string]uint64 `json:"callableComplexities"`
}

// ScriptEstimate returns estimates of compiled code in base64 representation.
func (a *Utils) ScriptEstimate(ctx context.Context, base64code string) (*UtilsScriptEstimate, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/utils/script/estimate")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", url.String(), strings.NewReader(base64code))
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
