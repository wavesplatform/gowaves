package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	logLevel      = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL.")
	nodesStr      = flag.String("nodes", "", "Addresses of nodes; comma separated.")
	startHeight   = flag.Int("start-height", 1, "Start height.")
	endHeight     = flag.Int("end-height", 2, "End height.")
	goroutinesNum = flag.Int("goroutines-num", 15, "Number of goroutines that will run for downloading state hashes.")
	tries         = flag.Int("tries-num", 5, "Number of tries to download.")
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
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var sh *proto.StateHash
	var err error
	for i := 0; i < *tries; i++ {
		sh, _, err = cl.Debug.StateHash(ctx, height)
		if err == nil {
			return sh, nil
		}
	}
	return nil, err
}

type printer struct {
	lock sync.Mutex
}

func (p *printer) printDifferentResults(height uint64, res map[proto.FieldsHashes]*nodesGroup) {
	p.lock.Lock()
	defer p.lock.Unlock()

	zap.S().Infof("Height: %d; following state hashes are different:", height)
	zap.S().Info("=============================================================")
	for fh, nodes := range res {
		hashJs, err := json.Marshal(fh)
		if err != nil {
			panic(err)
		}
		zap.S().Info(string(hashJs))
		zap.S().Infof("Nodes: %v\n", nodes.nodes)
	}
	zap.S().Info("=============================================================")
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

func manageHeight(ctx context.Context, height uint64, nodes []string, clients []*client.Client, p *printer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan hashResult, len(clients))
	for i, cl := range clients {
		node := nodes[i]
		go func(cl *client.Client, node string) {
			sh, err := loadStateHash(ctx, cl, height)
			res := hashResult{
				res: stateHashInfo{
					hash: sh,
					node: node,
				},
				err: err,
			}
			results <- res
		}(cl, node)
	}
	differentResults := make(map[proto.FieldsHashes]*nodesGroup)
	for range clients {
		hr := <-results
		if hr.err != nil {
			cancel()
			return hr.err
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
		p.printDifferentResults(height, differentResults)
	}
	return nil
}

func download(heightChan chan uint64, errChan chan error, nodes []string, clients []*client.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &printer{}
	var wg sync.WaitGroup
	for i := 0; i < *goroutinesNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for height := range heightChan {
				if err := manageHeight(ctx, height, nodes, clients, p); err != nil {
					cancel()
					zap.S().Errorf("Failed to load some hashes: %v", err)
					errChan <- err
					break
				}
			}
		}()
	}
	wg.Wait()
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

	heightChan := make(chan uint64)
	errChan := make(chan error, *goroutinesNum)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		download(heightChan, errChan, nodes, clients)
		wg.Done()
	}()
	for h := uint64(*startHeight); h < uint64(*endHeight); h++ {
		gotErr := false
		select {
		case heightChan <- h:
		case <-errChan:
			gotErr = true
		}
		if gotErr {
			break
		}
	}
	close(heightChan)
	wg.Wait()
	zap.S().Info("Finished to compare states.")
}
