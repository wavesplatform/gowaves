package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultTimeout = 15 * time.Second

type NodeClient struct {
	cl      *http.Client
	BaseUrl string
}

func NewNodeClient(url string, timeout time.Duration) *NodeClient {
	return &NodeClient{cl: &http.Client{Timeout: timeout}, BaseUrl: url}
}

type BlockHeightResponse struct {
	Height uint64 `json:"height"`
}

func (c *NodeClient) GetBlocksHeight() (BlockHeightResponse, error) {
	req, err := http.NewRequest("GET", c.BaseUrl+"blocks/height", nil)
	if err != nil {
		return BlockHeightResponse{}, fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Add("Accept", "application/json")

	respRaw, err := c.cl.Do(req)
	if err != nil {
		return BlockHeightResponse{}, err
	}

	var resp BlockHeightResponse
	if err = json.NewDecoder(respRaw.Body).Decode(&resp); err != nil {
		return BlockHeightResponse{}, fmt.Errorf("parse error: %s", err)
	}
	return resp, nil
}

type NodeVersionResponse struct {
	Version string `json:"height"`
}

func (c *NodeClient) GetNodeVersion() (NodeVersionResponse, error) {
	req, err := http.NewRequest("GET", c.BaseUrl+"node/version", nil)
	if err != nil {
		return NodeVersionResponse{}, fmt.Errorf("failed to create request: %s", err)
	}
	req.Header.Add("Accept", "application/json")

	respRaw, err := c.cl.Do(req)
	if err != nil {
		return NodeVersionResponse{}, err
	}

	var resp NodeVersionResponse
	if err = json.NewDecoder(respRaw.Body).Decode(&resp); err != nil {
		return NodeVersionResponse{}, fmt.Errorf("parse error: %s", err)
	}
	return resp, nil
}
