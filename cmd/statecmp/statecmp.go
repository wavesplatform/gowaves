package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultScheme = "http"
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

func loadStateHash(ctx context.Context, cl *client.Client, height uint64, tries int) (*proto.StateHash, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var sh *proto.StateHash
	var err error
	for i := 0; i < tries; i++ {
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

func manageHeight(
	ctx context.Context, height uint64, nodes []string, clients []*client.Client, p *printer, tries int,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan hashResult, len(clients))
	for i, cl := range clients {
		node := nodes[i]
		go func(cl *client.Client, node string) {
			sh, err := loadStateHash(ctx, cl, height, tries)
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

func download(
	heightChan chan uint64, errChan chan error, nodes []string, clients []*client.Client, goroutines, tries int,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &printer{}
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for height := range heightChan {
				if err := manageHeight(ctx, height, nodes, clients, p, tries); err != nil {
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
	const (
		defaultStartHeight      = 1
		defaultEndHeight        = 2
		defaultGoroutinesNumber = 15
		defaultTriesNumber      = 5
		minNodesNumber          = 2
	)
	var (
		logLevel = zap.LevelFlag("log-level", zapcore.InfoLevel,
			"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL.")
		nodesStr      = flag.String("nodes", "", "Addresses of nodes; comma separated.")
		startHeight   = flag.Int("start-height", defaultStartHeight, "Start height.")
		endHeight     = flag.Int("end-height", defaultEndHeight, "End height.")
		goroutinesNum = flag.Int("goroutines-num", defaultGoroutinesNumber,
			"Number of goroutines that will run for downloading state hashes.")
		tries = flag.Int("tries-num", defaultTriesNumber, "Number of tries to download.")
	)

	flag.Parse()

	logger := logging.SetupSimpleLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil && errors.Is(err, os.ErrInvalid) {
			panic(fmt.Sprintf("Failed to close logging subsystem: %v\n", err))
		}
	}()
	if *endHeight <= *startHeight {
		zap.S().Fatal("End height must be greater than start height.")
	}

	nodes := strings.FieldsFunc(*nodesStr, func(r rune) bool { return r == ',' })
	if len(nodes) < minNodesNumber {
		zap.S().Fatal("Expected at least 2 nodes.")
	}
	clients := make([]*client.Client, len(nodes))
	for i, nu := range nodes {
		u, err := checkAndUpdateURL(nu)
		if err != nil {
			zap.S().Fatalf("Incorrect node URL: %v", err)
		}
		clients[i], err = client.NewClient(client.Options{BaseUrl: u, Client: &http.Client{}})
		if err != nil {
			zap.S().Fatalf("Failed to create client: %v", err)
		}
	}

	heightChan := make(chan uint64)
	errChan := make(chan error, *goroutinesNum)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		download(heightChan, errChan, nodes, clients, *goroutinesNum, *tries)
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
