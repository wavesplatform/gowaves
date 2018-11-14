package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Transactions struct {
	options Options
}

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

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}

	b := buf.Bytes()

	tt := new(TransactionTypeVersion)
	err = json.NewDecoder(buf).Decode(tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	realType, err := GuessTransactionType(tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	err = json.Unmarshal(b, realType)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	return realType, response, nil
}

// Get the number of unconfirmed transactions in the UTX pool
func (a *Transactions) UnconfirmedSize(ctx context.Context) (uint64, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/transactions/unconfirmed/size")
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
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

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}
	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	// reference to original bytes
	b := buf.Bytes()

	var tt []*TransactionTypeVersion
	err = json.NewDecoder(buf).Decode(&tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	if len(tt) == 0 {
		return nil, response, nil
	}

	out := make([]proto.Transaction, len(tt))
	for i, row := range tt {
		realType, err := GuessTransactionType(row)
		if err != nil {
			return nil, response, &ParseError{Err: err}
		}
		out[i] = realType
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}
	return out, response, nil
}

type TransactionTypeVersion struct {
	Type    proto.TransactionType `json:"type"`
	Version byte                  `json:"version,omitempty"`
}

// Get transaction info
func (a *Transactions) Info(ctx context.Context, id crypto.Digest) (proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/transactions/info/%s", id.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}

	b := buf.Bytes()

	tt := new(TransactionTypeVersion)
	err = json.NewDecoder(buf).Decode(tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	realType, err := GuessTransactionType(tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	err = json.Unmarshal(b, realType)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	return realType, response, nil
}

// Guess transaction from type and version
func GuessTransactionType(t *TransactionTypeVersion) (proto.Transaction, error) {
	var out proto.Transaction
	switch t.Type {
	case proto.GenesisTransaction: // 1
		out = &proto.Genesis{}
	case proto.PaymentTransaction: // 2
		out = &proto.Payment{}
	case proto.IssueTransaction: // 3
		out = &proto.IssueV1{}
	case proto.TransferTransaction: // 4
		out = &proto.TransferV1{}
	case proto.ReissueTransaction: // 5
		out = &proto.ReissueV1{}
	case proto.BurnTransaction: // 6
		out = &proto.BurnV1{}
	case proto.ExchangeTransaction: // 7
		out = &proto.ExchangeV1{}
	case proto.LeaseTransaction: // 8
		out = &proto.LeaseV1{}
	case proto.LeaseCancelTransaction: // 9
		out = &proto.LeaseCancelV1{}
	case proto.CreateAliasTransaction: // 10
		out = &proto.CreateAliasV1{}
	case proto.MassTransferTransaction: // 11
		out = &proto.MassTransferV1{}
	case proto.DataTransaction: // 12
		out = &proto.DataV1{}
	case proto.SetScriptTransaction: // 13
		out = &proto.SetScriptV1{}
	case proto.SponsorshipTransaction: // 14
		out = &proto.SponsorshipV1{}
	}
	if out == nil {
		return nil, errors.Errorf("unknown transaction type %d version %d", t.Type, t.Version)
	}
	return out, nil
}

// Get list of transactions where specified address has been involved
func (a *Transactions) Address(ctx context.Context, address proto.Address, limit uint) ([]proto.Transaction, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/transactions/address/%s/limit/%d", address.String(), limit))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}
	// reference to original bytes
	b := buf.Bytes()

	var tt [][]*TransactionTypeVersion
	err = json.NewDecoder(buf).Decode(&tt)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}

	if len(tt) == 0 {
		return nil, response, nil
	}

	out := make([]proto.Transaction, len(tt[0]))
	for i, row := range tt[0] {
		realType, err := GuessTransactionType(row)
		if err != nil {
			return nil, response, &ParseError{Err: err}
		}
		out[i] = realType
	}

	j := [][]proto.Transaction{out}
	err = json.Unmarshal(b, &j)
	if err != nil {
		return nil, response, &ParseError{Err: err}
	}
	if len(j) == 0 {
		return nil, response, nil
	}
	return out, response, nil
}
