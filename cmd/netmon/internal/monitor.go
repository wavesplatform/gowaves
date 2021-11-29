package internal

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const NodeHistorySize = 100

type nodeStatus int

const (
	NotResponding nodeStatus = iota
	Contacted
	OnHeight
	StateHashReceived
	Behind
	OnTip
)

type nodeInfo struct {
	node    string
	status  nodeStatus
	version string
	height  int
	sh      *proto.StateHash
}

type nodeHistory struct {
	log []nodeInfo
}

func newNodeHistory(info nodeInfo) *nodeHistory {
	l := make([]nodeInfo, 0, NodeHistorySize+1)
	return &nodeHistory{log: append(l, info)}
}

func (h *nodeHistory) push(info nodeInfo) {
	if len(h.log) == 0 {
		h.log = make([]nodeInfo, 0, NodeHistorySize+1)
	}
	h.log = append(h.log, info)
	if len(h.log) > NodeHistorySize {
		h.log = h.log[1:]
	}
}

type NodeMonitor struct {
	interval time.Duration
	nodes    []*nodeClient
	history  map[string]*nodeHistory
}

func NewNodeMonitor(nodes string, interval, timeout int) (*NodeMonitor, error) {
	ns := strings.Fields(nodes)
	cs := make([]*nodeClient, len(ns))
	for i, n := range ns {
		var err error
		cs[i], err = newNodeClient(n, timeout)
		if err != nil {
			return nil, err
		}
	}
	return &NodeMonitor{
		interval: time.Duration(interval) * time.Second,
		nodes:    cs,
		history:  make(map[string]*nodeHistory),
	}, nil
}

func (s *NodeMonitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.interval)
		for {
			s.poll(ctx)
			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *NodeMonitor) poll(ctx context.Context) {
	cc, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	rch := make(chan nodeInfo)
	defer close(rch)

	wg.Add(len(s.nodes))
	for i := range s.nodes {
		n := s.nodes[i]
		go func() {
			queryNode(cc, n, rch)
		}()
	}
	go func() {
		for info := range rch {
			if h, ok := s.history[info.node]; ok {
				h.push(info)
			} else {
				s.history[info.node] = newNodeHistory(info)
			}
			wg.Done()
		}
	}()
	wg.Wait()
}

func queryNode(ctx context.Context, node *nodeClient, rch chan nodeInfo) {
	r := nodeInfo{node: node.url, status: NotResponding}
	v, err := node.version(ctx)
	if err != nil {
		rch <- r
		return
	}
	r.version = v
	r.status = Contacted
	h, err := node.height(ctx)
	if err != nil {
		rch <- r
		return
	}
	if h <= 0 {
		rch <- r
		return
	}
	r.height = h
	r.status = OnHeight
	sh, err := node.stateHash(ctx, h-1)
	if err != nil {
		rch <- r
		return
	}
	r.sh = sh
	r.status = StateHashReceived
	rch <- r
	return
}

func (s *NodeMonitor) Health() (NetworkHealth, error) {
	//TODO: Here comes
	return NetworkHealth{}, nil
}
