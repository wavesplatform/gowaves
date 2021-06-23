package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultURL    = "https://nodes.wavesnodes.com"
	defaultScheme = "http"
)

var (
	version              = "v0.0.0"
	interruptSignals     = []os.Signal{os.Interrupt}
	errInvalidParameters = errors.New("invalid parameters")
	errUserTermination   = errors.New("user termination")
	errFailure           = errors.New("operation failure")
	errFork              = errors.New("the node is on fork")
	errUnavailable       = errors.New("remote service is unavailable")
)

func main() {
	err := run()
	if err != nil {
		switch err {
		case errInvalidParameters:
			showUsageAndExit()
			os.Exit(2)
		case errUserTermination:
			os.Exit(130)
		case errFork:
			os.Exit(1)
		case errUnavailable:
			os.Exit(69)
		case errFailure:
			os.Exit(70)
		}
	}
}

func run() error {
	var showHelp bool
	var showVersion bool
	var node string
	var reference string
	var verbose bool
	var silent bool

	flag.StringVarP(&node, "node", "n", "", "URL of the node")
	flag.StringVarP(&reference, "references", "r", defaultURL, "A list of space-separated URLs of reference nodes, for example \"http://127.0.0.1:6869 https://nodes.wavesnodes.com\"")
	flag.BoolVarP(&showHelp, "help", "h", false, "Print usage information (this message) and quit")
	flag.BoolVarP(&showVersion, "version", "v", false, "Print version information and quit")
	flag.BoolVar(&verbose, "verbose", false, "Logs additional information; incompatible with \"silent\"")
	flag.BoolVar(&silent, "silent", false, "Produce no output except this help message; incompatible with \"verbose\"")
	flag.Parse()

	if showHelp {
		showUsageAndExit()
		return nil
	}
	if showVersion {
		fmt.Printf("chaincmp %s\n", version)
		return nil
	}

	if silent && verbose {
		return errInvalidParameters
	}
	setupLogger(silent, verbose)

	if node == "" || len(strings.Fields(node)) > 1 {
		zap.S().Errorf("Invalid node's URL '%s'", node)
		return errInvalidParameters
	}
	node, err := checkAndUpdateURL(node)
	if err != nil {
		zap.S().Errorf("Incorrect node's URL: %s", err.Error())
		return errInvalidParameters
	}
	other := strings.Fields(reference)
	for i, u := range other {
		u, err = checkAndUpdateURL(u)
		if err != nil {
			zap.S().Error("Incorrect reference's URL: %s", err.Error())
			return errInvalidParameters
		}
		other[i] = u
	}

	zap.S().Debugf("Node to check: %s", node)
	zap.S().Debugf("Reference nodes (%d): %s", len(other), other)

	urls := append([]string{node}, other...)
	zap.S().Debugf("Requesting height from %d nodes", len(urls))

	interrupt := interruptListener()

	clients := make([]*client.Client, len(urls))
	for i, u := range urls {
		c, err := client.NewClient(client.Options{BaseUrl: u, Client: &http.Client{}})
		if err != nil {
			zap.S().Errorf("Failed to create client for URL '%s': %s", u, err)
			return errFailure
		}
		clients[i] = c
	}

	hs, err := heights(interrupt, clients)
	if err != nil {
		zap.S().Errorf("Failed to retrieve heights from all nodes: %s", err)
		if interrupted(interrupt) {
			return errUserTermination
		}
		return errUnavailable
	}
	for i, h := range hs {
		zap.S().Debugf("%d: Height = %d", i, h)
	}

	stop := min(hs)
	zap.S().Infof("Lowest height: %d", stop)

	ch, err := findLastCommonHeight(interrupt, clients, 1, stop)
	if err != nil {
		zap.S().Errorf("Failed to find last common height: %s", err)
		if interrupted(interrupt) {
			return errUserTermination
		}
		return err
	}

	h := hs[0]
	zap.S().Debugf("Node height: %d", h)
	refLowest := min(hs[1:])
	zap.S().Debugf("The lowest height of reference nodes: %d", refLowest)

	switch {
	case ch == h && ch < refLowest: // The node is behind the reference nodes
		zap.S().Infof("Node '%s' is %d blocks behind the lowest reference node", node, refLowest-h)
		return nil
	case ch == refLowest && ch < h: // The node is ahead of the reference nodes
		zap.S().Infof("Node '%s' is %d blocks ahead of the lowest reference node", node, h-refLowest)
		return nil
	case ch < h && ch < refLowest:
		fl := h - ch
		zap.S().Warnf("Node '%s' is on fork of length %d blocks since last common block at height %d", node, fl, ch)
		switch {
		case fl < 10:
			zap.S().Infof("The fork is very short, highly likely the node is OK")
			return nil
		case fl < 100:
			zap.S().Warn("The fork is short and possibly the node will rollback and switch on the correct fork automatically")
			zap.S().Warnf("But if you want to rollback manually, refer the documentation at https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html")
			return errFork
		case fl < 1980:
			zap.S().Warnf("Manual rollback of the node '%s' is possible, do it as soon as possible!", node)
			zap.S().Warnf("Please, read the documentation at https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html")
			return errFork
		default:
			zap.S().Warnf("Rollback of node '%s' is not an option, the fork is too long, consider restarting the node from scratch!", node)
			zap.S().Warnf("Please, refer the documentation at https://docs.wavesplatform.com/en/waves-full-node/options-for-getting-actual-blockchain.html")
			return errFork
		}
	default:
		zap.S().Infof("Node '%s' is OK", node)
		return nil
	}
}

func checkAndUpdateURL(s string) (string, error) {
	var u *url.URL
	var err error
	if strings.Contains(s, "//") {
		u, err = url.Parse(s)
	} else {
		u, err = url.Parse("//" + s)
	}
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse URL '%s'", s)
	}
	if u.Scheme == "" {
		u.Scheme = defaultScheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}
	return u.String(), nil
}

func findLastCommonHeight(interrupt <-chan struct{}, clients []*client.Client, start, stop int) (int, error) {
	var r int
	for start <= stop {
		if interrupted(interrupt) {
			return 0, errUserTermination
		}
		middle := (start + stop) / 2
		c, err := differentIdsCount(clients, middle)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get blocks signatures at height %d", middle)
		}
		if c >= 2 {
			stop = middle - 1
			r = stop
		} else {
			start = middle + 1
			r = middle
		}
	}
	return r, nil
}

type nodeBlockInfo struct {
	id        int
	blockID   proto.BlockID
	height    uint64
	generator proto.WavesAddress
	blockTime uint64
	err       error
}

func differentIdsCount(clients []*client.Client, height int) (int, error) {
	ch := make(chan nodeBlockInfo, len(clients))
	info := make(map[int]nodeBlockInfo)
	m := make(map[proto.BlockID]bool)
	for i, c := range clients {
		go func(id int, cl *client.Client) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			header, resp, err := cl.Blocks.HeadersAt(ctx, uint64(height))
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
					defer cancel()
					block, _, err := cl.Blocks.At(ctx, uint64(height))
					if err != nil {
						ch <- nodeBlockInfo{id: id, err: err}
						return
					}
					ch <- nodeBlockInfo{id: id, height: block.Height, blockID: block.ID, generator: block.Generator, blockTime: block.Timestamp}
					return
				}
				ch <- nodeBlockInfo{id: id, err: err}
				return
			}
			ch <- nodeBlockInfo{id: id, height: header.Height, blockID: header.ID, generator: header.Generator, blockTime: header.Timestamp}
		}(i, c)
	}
	for range clients {
		bi := <-ch
		if bi.err != nil {
			return 0, errors.Wrapf(bi.err, "failed to get block header from %dth client", bi.id)
		}
		info[bi.id] = bi
		m[bi.blockID] = true
	}
	for i := 0; i < len(info); i++ {
		v := info[i]
		t := time.Unix(0, int64(v.blockTime*1000000))
		zap.S().Debugf("id: %d, h: %d, block: %s, generator: %s, time: %s", i, v.height, v.blockID.String(), v.generator.String(), t.String())
	}
	return len(m), nil
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
	_, _ = fmt.Fprintf(os.Stderr, "\nUsage of chaincmp %s\n", version)
	flag.PrintDefaults()
}

type nodeHeight struct {
	id     int
	height int
	err    error
}

func height(interrupt <-chan struct{}, c *client.Client, id int, ch chan nodeHeight) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	go func() {
		<-interrupt
		cancel()
	}()

	bh, _, err := c.Blocks.Height(ctx)
	if err != nil {
		ch <- nodeHeight{id, 0, err}
		return
	}
	ch <- nodeHeight{id, int(bh.Height), nil}
}

func heights(interrupt <-chan struct{}, clients []*client.Client) ([]int, error) {
	ch := make(chan nodeHeight, len(clients))
	heights := make(map[int]int)

	for i, c := range clients {
		go height(interrupt, c, i, ch)
	}

	for range clients {
		nh := <-ch
		if nh.err != nil {
			return nil, errors.Wrapf(nh.err, "failed to retrieve height from %dth client", nh.id)
		}
		heights[nh.id] = nh.height
	}

	r := make([]int, len(heights))
	for i, height := range heights {
		r[i] = height
	}
	return r, nil
}

func setupLogger(silent, verbose bool) {
	al := zap.NewAtomicLevel()
	al.SetLevel(zap.InfoLevel)
	if silent {
		al.SetLevel(zap.FatalLevel)
	}
	if verbose {
		al.SetLevel(zap.DebugLevel)
	}
	ec := zap.NewDevelopmentEncoderConfig()
	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.Lock(os.Stdout), al))
	zap.ReplaceGlobals(logger)
}

func interruptListener() <-chan struct{} {
	r := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, interruptSignals...)
		sig := <-signals
		zap.S().Infof("Caught signal '%s', shutting down...", sig)
		close(r)
		for sig := range signals {
			zap.S().Infof("Caught signal '%s' again, already shutting down", sig)
		}
	}()
	return r
}

func interrupted(interrupt <-chan struct{}) bool {
	select {
	case <-interrupt:
		return true
	default:
	}
	return false
}
