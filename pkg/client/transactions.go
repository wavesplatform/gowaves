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
)

type Transactions struct {
	options Options
}

// Creates new transaction api section
func NewTransactions(options Options) *Transactions {
	return &Transactions{
		options: options,
	}
}

// Get transaction that is in the UTX
func (a *Transactions) UnconfirmedInfo(ctx context.Context, id crypto.Digest) (proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/transactions/unconfirmed/info/%s", id.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	buf.WriteRune(']')
	out := TransactionsField{}
	err = json.Unmarshal(buf.Bytes(), &out)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	if len(out) == 0 {
		return nil, response, errors.New("invalid transaction")
	}

	return out[0], response, nil
}

// Get the number of unconfirmed transactions in the UTX pool
func (a *Transactions) UnconfirmedSize(ctx context.Context) (uint64, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/transactions/unconfirmed/size")
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return 0, nil, err
	}

	out := make(map[string]uint64)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return 0, response, err
	}

	return out["size"], response, nil
}

// Get the number of unconfirmed transactions in the UTX pool
func (a *Transactions) Unconfirmed(ctx context.Context) ([]proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/transactions/unconfirmed")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := TransactionsField{}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Get transaction info
func (a *Transactions) Info(ctx context.Context, id crypto.Digest) (proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/transactions/info/%s", id.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	buf.WriteRune(']')
	out := TransactionsField{}
	err = json.Unmarshal(buf.Bytes(), &out)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	if len(out) == 0 {
		return nil, response, errors.New("invalid transaction ")
	}

	return out[0], response, nil
}

// Get list of transactions where specified address has been involved
func (a *Transactions) Address(ctx context.Context, address proto.Address, limit uint) ([]proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/transactions/address/%s/limit/%d", address.String(), limit))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []TransactionsField
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}
	if len(out) == 0 {
		return nil, response, nil
	}
	return out[0], response, nil
}

// Broadcast a signed transaction
func (a *Transactions) Broadcast(ctx context.Context, transaction proto.Transaction) (*Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/transactions/broadcast")
	if err != nil {
		return nil, err
	}

	bts, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(bts))
	if err != nil {
		return nil, err
	}

	response, err := doHttp(ctx, a.options, req, nil)
	if err != nil {
		return response, err
	}
	return response, nil
}
