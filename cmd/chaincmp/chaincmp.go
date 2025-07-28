package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
		slog.Error("Invalid node", "URL", node)
		return errInvalidParameters
	}
	node, err := checkAndUpdateURL(node)
	if err != nil {
		slog.Error("Incorrect node's URL", logging.Error(err))
		return errInvalidParameters
	}
	other := strings.Fields(reference)
	for i, u := range other {
		u, err = checkAndUpdateURL(u)
		if err != nil {
			slog.Error("Incorrect reference's URL", logging.Error(err))
			return errInvalidParameters
		}
		other[i] = u
	}

	slog.Debug("Node to check", "URL", node)
	slog.Debug("Reference nodes", "count", len(other), "nodes", other)

	urls := append([]string{node}, other...)
	slog.Debug("Requesting height from nodes", "count", len(urls))

	interrupt := interruptListener()

	clients := make([]*client.Client, len(urls))
	for i, u := range urls {
		c, err := client.NewClient(client.Options{BaseUrl: u, Client: &http.Client{}})
		if err != nil {
			slog.Error("Failed to create client", slog.String("URL", u), logging.Error(err))
			return errFailure
		}
		clients[i] = c
	}

	hs, err := heights(interrupt, clients)
	if err != nil {
		slog.Error("Failed to retrieve heights from all nodes", logging.Error(err))
		if interrupted(interrupt) {
			return errUserTermination
		}
		return errUnavailable
	}
	for i, h := range hs {
		slog.Debug("Height at node", "node", i, "height", h)
	}

	stop := min(hs)
	slog.Info("Lowest height", "height", stop)

	ch, err := findLastCommonHeight(interrupt, clients, 1, stop)
	if err != nil {
		slog.Error("Failed to find last common height", logging.Error(err))
		if interrupted(interrupt) {
			return errUserTermination
		}
		return err
	}

	h := hs[0]
	slog.Debug("Node height", "height", h)
	refLowest := min(hs[1:])
	slog.Debug("The lowest height of reference nodes", "height", refLowest)

	switch {
	case ch == h && ch < refLowest: // The node is behind the reference nodes
		slog.Info("Node is behind the lowest reference node", "node", node, "blocks", refLowest-h)
		return nil
	case ch == refLowest && ch < h: // The node is ahead of the reference nodes
		slog.Info("Node is ahead of the lowest reference node", "node", node, "blocks", h-refLowest)
		return nil
	case ch < h && ch < refLowest:
		fl := h - ch
		slog.Warn("Node is on fork", "node", node, "forkLength", fl, "commonHeight", ch)
		switch {
		case fl < 10:
			slog.Info("The fork is very short, highly likely the node is OK")
			return nil
		case fl < 100:
			slog.Warn(
				"The fork is short and possibly the node will rollback and switch on the correct fork automatically")
			slog.Warn(
				"But if you want to rollback manually, refer the documentation at " +
					"https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html")
			return errFork
		case fl < 1980:
			slog.Warn("Manual rollback of the node is possible, do it as soon as possible!", "node", node)
			slog.Warn("Please, read the documentation at " +
				"https://docs.wavesplatform.com/en/waves-full-node/how-to-rollback-a-node.html")
			return errFork
		default:
			slog.Warn(
				"Rollback of node is not an option, the fork is too long, consider restarting the node from scratch!",
				"node", node)
			slog.Warn("Please, refer the documentation at " +
				"https://docs.wavesplatform.com/en/waves-full-node/options-for-getting-actual-blockchain.html")
			return errFork
		}
	default:
		slog.Info("Node is OK", "node", node)
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
		slog.Debug("Different ID", "i", i, "height", v.height, "block", v.blockID.String(),
			"generator", v.generator.String(), "time", t.String())
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
	level := slog.LevelInfo
	if silent {
		level = slog.LevelError
	}
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(logging.NewHandler(logging.LoggerPrettyNoColor, level)))
}

func interruptListener() <-chan struct{} {
	r := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, interruptSignals...)
		sig := <-signals
		slog.Info("Caught signal, shutting down...", "signal", sig)
		close(r)
		for sig := range signals {
			slog.Info("Caught signal again, already shutting down", "signal", sig)
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
