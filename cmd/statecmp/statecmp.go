package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

const (
	defaultScheme = "http"
)

var (
	logLevel    = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL.")
	nodesStr    = flag.String("nodes", "", "Addresses of nodes; comma separated.")
	startHeight = flag.Int("start-height", 1, "Start height.")
	endHeight   = flag.Int("end-height", 2, "End height.")
)

func checkAndUpdateURL(s string) (string, error) {
	var u *url.URL
	var err error
	if strings.Contains(s, "//") {
		u, err = url.Parse(s)
	} else {
		u, err = url.Parse("//" + s)
	}
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse URL %s", s)
	}
	if u.Scheme == "" {
		u.Scheme = defaultScheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.Errorf("unsupported URL scheme %s", u.Scheme)
	}
	return u.String(), nil
}

func loadStateHash(ctx context.Context, cl *client.Client, height uint64) (*proto.StateHash, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	sh, _, err := cl.Debug.StateHash(ctx, height)
	if err != nil {
		return nil, err
	}
	return sh, nil
}

func printDifferentResults(res map[proto.FieldsHashes]*nodesGroup) {
	for fh, nodes := range res {
		hashJs, err := json.Marshal(fh)
		if err != nil {
			panic(err)
		}
		zap.S().Info(string(hashJs))
		zap.S().Infof("Nodes: %v\n", nodes.nodes)
	}
}

type stateHashInfo struct {
	hash *proto.StateHash
	node string
}

type hashResult struct {
	res stateHashInfo
	err error
}

type nodesGroup struct {
	nodes []string
}

func newNodesGroup(first string) *nodesGroup {
	nodes := make([]string, 1)
	nodes[0] = first
	return &nodesGroup{nodes: nodes}
}

func (ng *nodesGroup) addNode(node string) {
	ng.nodes = append(ng.nodes, node)
}

func main() {
	flag.Parse()

	common.SetupLogger(*logLevel)
	if *endHeight <= *startHeight {
		zap.S().Fatal("End height must be greater than start height.")
	}

	nodes := strings.FieldsFunc(*nodesStr, func(r rune) bool { return r == ',' })
	if len(nodes) < 2 {
		zap.S().Fatal("Expected at least 2 nodes.")
	}
	clients := make([]*client.Client, len(nodes))
	for i, u := range nodes {
		url, err := checkAndUpdateURL(u)
		if err != nil {
			zap.S().Fatalf("Incorrect node URL: %v", err)
		}
		clients[i], err = client.NewClient(client.Options{BaseUrl: url, Client: &http.Client{}})
		if err != nil {
			zap.S().Fatalf("Failed to create client: %v", err)
		}
	}

	results := make(chan hashResult, len(clients))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	identical := true
	for h := uint64(*startHeight); h < uint64(*endHeight); h++ {
		for i, cl := range clients {
			node := nodes[i]
			go func(cl *client.Client, node string, h uint64) {
				sh, err := loadStateHash(ctx, cl, h)
				res := hashResult{
					res: stateHashInfo{
						hash: sh,
						node: node,
					},
					err: err,
				}
				results <- res
			}(cl, node, h)
		}
		differentResults := make(map[proto.FieldsHashes]*nodesGroup)
		for range clients {
			hr := <-results
			if hr.err != nil {
				cancel()
				zap.S().Fatalf("Failed to load some hash: %v", hr.err)
			}
			fh := hr.res.hash.FieldsHashes
			nodesGroup, ok := differentResults[fh]
			if !ok {
				differentResults[fh] = newNodesGroup(hr.res.node)
			} else {
				nodesGroup.addNode(hr.res.node)
			}
		}
		if len(differentResults) != 1 {
			identical = false
			zap.S().Infof("Height: %d; following state hashes are different:", h)
			zap.S().Info("=============================================================")
			printDifferentResults(differentResults)
			zap.S().Info("=============================================================")
		}
	}
	if identical {
		zap.S().Info("All hashes are identical at all heights!")
	}
	close(results)
}
