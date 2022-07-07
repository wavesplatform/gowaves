package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const DefaultTimeout = 15 * time.Second

type NodeClient struct {
	cl      *http.Client
	BaseUrl string
}

func NewNodeClient(url string, timeout time.Duration) *NodeClient {
	return &NodeClient{cl: &http.Client{Timeout: timeout}, BaseUrl: url}
}

func (c *NodeClient) GetBlocksHeight() (*client.BlocksHeight, error) {
	req, err := http.NewRequest("GET", c.BaseUrl+"blocks/height", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Add("Accept", "application/json")

	respRaw, err := c.cl.Do(req)
	if err != nil {
		return nil, err
	}

	resp := &client.BlocksHeight{}
	if err = json.NewDecoder(respRaw.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("parse error: %s", err)
	}
	return resp, nil
}

type NodeVersionResponse struct {
	Version string `json:"version"`
}

func (c *NodeClient) GetNodeVersion() (*NodeVersionResponse, error) {
	req, err := http.NewRequest("GET", c.BaseUrl+"node/version", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Add("Accept", "application/json")

	respRaw, err := c.cl.Do(req)
	if err != nil {
		return nil, err
	}

	resp := &NodeVersionResponse{}
	if err = json.NewDecoder(respRaw.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("parse error: %s", err)
	}
	return resp, nil
}

func (c *NodeClient) GetStateHash(height uint64) (*proto.StateHash, error) {
	req, err := http.NewRequest("GET", c.BaseUrl+"debug/stateHash/"+strconv.FormatUint(height, 10), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Add("Accept", "application/json")

	respRaw, err := c.cl.Do(req)
	if err != nil {
		return nil, err
	}

	resp := &proto.StateHash{}
	if err = json.NewDecoder(respRaw.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("parse error: %s", err)
	}
	return resp, nil
}
