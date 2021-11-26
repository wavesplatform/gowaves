package internal

import (
	"context"
	"strings"
)

type nodeStats struct {
}

type NodeMonitor struct {
	nodes map[string]*nodeClient
	stats map[string]nodeStats
}

func NewNodeMonitor(nodes string, timeout int) (*NodeMonitor, error) {
	ns := strings.Fields(nodes)
	nm := make(map[string]*nodeClient)
	for _, n := range ns {
		c, err := newNodeClient(n, timeout)
		if err != nil {
			return nil, err
		}
		nm[c.url] = c
	}
	return &NodeMonitor{
		nodes: nm,
		stats: make(map[string]nodeStats),
	}, nil
}

func (s *NodeMonitor) Start(ctx context.Context) {
	go func() {
		for {
			//TODO: here comes polling logic
		}
	}()
}

func (s *NodeMonitor) Health() (NetworkHealth, error) {
	//TODO: Here comes
	return NetworkHealth{}, nil
}
