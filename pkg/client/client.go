package client

import (
	"github.com/pkg/errors"

	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

// ApiKeyHeader is an HTTP header name for API Key
const ApiKeyHeader = "X-API-Key" // #nosec: it's a header name

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Options struct {
	BaseUrl string
	Client  Doer
	ApiKey  string
}

var defaultOptions = Options{
	BaseUrl: "https://nodes.wavesnodes.com",
	Client:  &http.Client{Timeout: 3 * time.Second},
}

type Client struct {
	options      Options
	Addresses    *Addresses
	Blocks       *Blocks
	Wallet       *Wallet
	Alias        *Alias
	NodeInfo     *NodeInfo
	Peers        *Peers
	Transactions *Transactions
	Assets       *Assets
	Utils        *Utils
	Leasing      *Leasing
	Debug        *Debug
}

type Response struct {
	*http.Response
}

type HttpClient interface {
}

// NewClient creates new client instance.
// If no options provided will use default.
func NewClient(options ...Options) (*Client, error) {
	if len(options) > 1 {
		return nil, errors.New("too many options provided. Expects no or just one item")
	}

	opts := defaultOptions

	if len(options) == 1 {
		option := options[0]

		if option.BaseUrl != "" {
			opts.BaseUrl = option.BaseUrl
		}

		if option.Client != nil {
			opts.Client = option.Client
		}

		if option.ApiKey != "" {
			opts.ApiKey = option.ApiKey
		}
	}

	c := &Client{
		options:      opts,
		Addresses:    NewAddresses(opts),
		Blocks:       NewBlocks(opts),
		Wallet:       NewWallet(opts),
		Alias:        NewAlias(opts),
		Peers:        NewPeers(opts),
		NodeInfo:     NewNodeInfo(opts),
		Transactions: NewTransactions(opts),
		Assets:       NewAssets(opts),
		Utils:        NewUtils(opts),
		Leasing:      NewLeasing(opts),
		Debug:        NewDebug(opts),
	}

	return c, nil
}

func (a *Client) GetOptions() Options {
	return a.options
}

func withContext(ctx context.Context, req *http.Request) *http.Request {
	return req.WithContext(ctx)
}

func newResponse(response *http.Response) *Response {
	return &Response{
		Response: response,
	}
}

func (a *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	return doHttp(ctx, a.options, req, v)
}

func doHttp(ctx context.Context, options Options, req *http.Request, v interface{}) (*Response, error) {
	req = withContext(ctx, req)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := options.Client.Do(req)
	if err != nil {
		return nil, &RequestError{Err: err}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close() // No error handling intentionally
	}(resp.Body)

	response := newResponse(resp)

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return response, &RequestError{
			Err:  errors.Errorf("Invalid status code: expect 200 got %d", response.StatusCode),
			Body: string(body),
		}
	}

	select {
	case <-ctx.Done():
		return response, ctx.Err()
	default:
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			if _, err := io.Copy(w, resp.Body); err != nil {
				return nil, err
			}
		} else {
			if err = json.NewDecoder(resp.Body).Decode(v); err != nil {
				return response, &ParseError{Err: err}
			}
		}
	}

	return response, err
}

func joinUrl(baseRaw string, pathRaw string) (*url.URL, error) {
	baseUrl, err := url.Parse(baseRaw)
	if err != nil {
		return nil, err
	}

	pathUrl, err := url.Parse(pathRaw)
	if err != nil {
		return nil, err
	}
	// nosemgrep: go.lang.correctness.use-filepath-join.use-filepath-join
	baseUrl.Path = path.Join(baseUrl.Path, pathUrl.Path)

	query := baseUrl.Query()
	for k := range pathUrl.Query() {
		query.Set(k, pathUrl.Query().Get(k))
	}
	baseUrl.RawQuery = query.Encode()

	return baseUrl, nil
}
