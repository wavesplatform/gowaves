package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultURL = "https://nodes.wavesnodes.com"
)

var version string

func main() {
	var showHelp bool
	var showVersion bool
	var node string
	var reference string
	var verbose bool

	flag.StringVar(&node, "n", "", "URL of the node")
	flag.StringVar(&node, "node", "", "URL of the node")
	flag.StringVar(&reference, "r", defaultURL, "List of URLs of reference nodes")
	flag.StringVar(&reference, "references", defaultURL, "List of URLs of reference nodes")
	flag.BoolVar(&showHelp, "h", false, "Print usage information (this message) and quit")
	flag.BoolVar(&showHelp, "help", false, "Print usage information (this message) and quit")
	flag.BoolVar(&showVersion, "v", false, "Print version information and quit")
	flag.BoolVar(&showVersion, "version", false, "Print version information and quit")
	flag.BoolVar(&verbose, "vvv", false, "Logs additional information")
	flag.BoolVar(&verbose, "verbose", false, "Logs additional information")
	flag.Usage = showUsageAndExit
	flag.Parse()

	if showHelp {
		showUsageAndExit()
	}
	if showVersion {
		showVersionAndExit()
	}

	al := zap.NewAtomicLevel()
	ec := zap.NewDevelopmentEncoderConfig()
	if verbose {
		al.SetLevel(zap.DebugLevel)
	}
	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.Lock(os.Stdout), al))
	defer logger.Sync()
	log := logger.Sugar()

	appCtx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		log.Infof("Shutting down")
		os.Exit(1)
	}()

	if node == "" || len(strings.Fields(node)) > 1 {
		showUsageAndExit()
	}
	other := strings.Fields(reference)

	log.Debugf("Node to check: %s", node)
	log.Debugf("Reference nodes (%d): %s", len(other), other)

	urls := append([]string{node}, other...)
	log.Debugf("Requesting height from %d nodes", len(urls))

	clients := make([]*client.Client, len(urls))
	for i, u := range urls {
		c, err := client.NewClient(client.Options{BaseUrl: u, Client: &http.Client{}})
		if err != nil {
			log.Errorf("Failed to create client for URL '%s': %s", u, err)
			cancel()
			os.Exit(1)
		}
		clients[i] = c
	}

	hs, err := heights(appCtx, clients)
	if err != nil {
		log.Errorf("Failed to retrieve heights from all nodes: %s", err)
	}
	for i, h := range hs {
		log.Debugf("%d: Height = %d", i, h)
	}

	stop := min(hs)
	log.Infof("Lowest height: %d", stop)

	ch, err := findLastCommonHeight(appCtx, log, clients, 1, stop)
	if err != nil {
		log.Errorf("Failed to find last common height: %s", err)
		cancel()
		os.Exit(1)
	}

	sigCnt, err := differentSignaturesCount(appCtx, log, clients, ch+1)
	if err != nil {
		log.Errorf("Failed to get blocks: %s", err)
		cancel()
		os.Exit(1)
	}

	h := hs[0]
	refLowest := min(hs[1:])
	if sigCnt != 1 {
		fl := h - ch
		log.Warnf("Node '%s' is on fork!!!", node)
		log.Warnf("Last common height: %d", ch)
		log.Warnf("Node '%s' on fork of length %d since block number %d", node, fl, ch)
		if fl < 1980 {
			if fl < 100 {
				log.Warnf("The fork is short and possibly the node will rollback and switch on the correct fork automatically.", node)
				log.Warnf("But if you want to rollback manually, refer the documentation at https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html.")
			} else {
				log.Warnf("Manual rollback of the node '%s' is possible, do it as soon as possible!", node)
				log.Warnf("Please, read the documentation at https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html.")
			}
		} else {
			log.Warnf("Rollback of node '%s' is not available, the fork is too long, consider restarting the node from scratch!", node)
			log.Warnf("Please, refer the documentation at https://docs.wavesplatform.com/en/waves-full-node/options-for-getting-actual-blockchain.html.")
		}
	} else {
		if h < refLowest {
			log.Infof("Node '%s' is %d blocks behind the lowest reference node", node, refLowest-h)
		} else if h == refLowest {
			log.Infof("Node '%s' is OK", node)
		} else {
			log.Infof("Node '%s' is %d blocks ahead of the lowest reference node", node, refLowest-h)
		}
	}
}

func findLastCommonHeight(rootContext context.Context, log *zap.SugaredLogger, clients []*client.Client, start, stop int) (int, error) {
	for start <= stop {
		middle := (start + stop) / 2
		if abs(start-stop) <= 1 {
			return middle, nil
		}
		c, err := differentSignaturesCount(rootContext, log, clients, middle)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get blocks signatures at height %d", middle)
		}
		if c >= 2 {
			stop = middle
		} else {
			start = middle
		}
	}
	return 0, errors.New("impossible situation")
}

type nodeHeader struct {
	id     int
	header *client.Headers
	err    error
}

func differentSignaturesCount(rootContext context.Context, log *zap.SugaredLogger, clients []*client.Client, height int) (int, error) {
	ch := make(chan nodeHeader, len(clients))
	info := make(map[int]*client.Headers)
	m := make(map[string]bool)
	for i, c := range clients {
		go func(id int, cl *client.Client) {
			ctx, cancel := context.WithTimeout(rootContext, time.Second*30)
			defer cancel()
			header, resp, err := cl.Blocks.HeadersAt(ctx, uint64(height))
			if err != nil {
				ch <- nodeHeader{id, header, err}
				return
			}
			if rc := resp.StatusCode; rc != 200 {
				ch <- nodeHeader{id, header, errors.Errorf("unexpected response code %d", rc)}
				return
			}
			ch <- nodeHeader{id, header, nil}
		}(i, c)
	}
	for range clients {
		h := <-ch
		if h.err != nil {
			return 0, errors.Wrapf(h.err, "failed to get block header from %dth client", h.id)
		}
		info[h.id] = h.header
		m[h.header.Signature] = true
	}
	for i := 0; i < len(info); i++ {
		v := info[i]
		t := time.Unix(0, int64(v.Timestamp*1000000))
		log.Debugf("id: %d, h: %d, block: %s, generator: %s, time: %s", i, v.Height, v.Signature, v.Generator, t.String())
	}
	return len(m), nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(values []int) int {
	r := values[0]
	for _, v := range values {
		if v < r {
			r = v
		}
	}
	return r
}

func showUsageAndExit() {
	const usageText = `
chaincmp [OPTIONS]

Options:
  -node, -n 		URL of the node
  -compare-to, -c 	List of URLs of reference nodes
  -help, -h			Print usage information (this message) and quit
  -version, -v		Print version information and quit
  -node, -n 		URL of the node used to broadcast transaction
`
	fmt.Fprint(os.Stdout, usageText)
	os.Exit(0)
}

func showVersionAndExit() {
	fmt.Printf("chaincmp %s\n", version)
	os.Exit(0)
}

type nodeHeight struct {
	id     int
	height int
	err    error
}

func height(rootContext context.Context, c *client.Client, id int, ch chan nodeHeight) {
	ctx, cancel := context.WithTimeout(rootContext, time.Second*30)
	defer cancel()

	bh, resp, err := c.Blocks.Height(ctx)
	if err != nil {
		ch <- nodeHeight{id, 0, err}
		return
	}
	if c := resp.StatusCode; c != 200 {
		ch <- nodeHeight{id, 0, errors.Errorf("unexpected response code %d", c)}
		return
	}
	ch <- nodeHeight{id, int(bh.Height), nil}
}

func heights(rootContext context.Context, clients []*client.Client) ([]int, error) {
	ch := make(chan nodeHeight, len(clients))
	heights := make(map[int]int)

	for i, c := range clients {
		go height(rootContext, c, i, ch)
	}

	for range clients {
		nh := <-ch
		if nh.err != nil {
			return nil, errors.Wrapf(nh.err, "failed to retrieve height from %dth client", nh.id)
		}
		heights[nh.id] = nh.height
	}

	r := make([]int, len(clients))
	for i := range clients {
		r[i] = heights[i]
	}
	return r, nil
}
