package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultTimeout = 15 * time.Second

type BlockHeightResponse struct {
	Height uint64 `json:"height"`
}

type NodeClient struct {
	cl      *http.Client
	BaseUrl string
}

func NewNodeClient(url string, timeout time.Duration) *NodeClient {
	return &NodeClient{cl: &http.Client{Timeout: timeout}, BaseUrl: url}
}

func (c *NodeClient) GetHeight() (BlockHeightResponse, error) {
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
