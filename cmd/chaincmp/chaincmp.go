package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultURL     = "https://nodes.wavesnodes.com"
	defaultScheme  = "http"
	defaultTimeout = 30 * time.Second
)

var (
	version              = "v0.0.0"
	errInvalidParameters = errors.New("invalid parameters")
	errUserTermination   = errors.New("user termination")
	errFailure           = errors.New("operation failure")
	errFork              = errors.New("the node is on fork")
	errUnavailable       = errors.New("remote service is unavailable")
)

func main() {
	err := run()
	if err != nil {
		switch {
		case errors.Is(err, errInvalidParameters):
			showUsageAndExit()
			os.Exit(2)
		case errors.Is(err, errUserTermination):
			os.Exit(130)
		case errors.Is(err, errFork):
			os.Exit(1)
		case errors.Is(err, errUnavailable):
			os.Exit(69)
		case errors.Is(err, errFailure):
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
		return errors.Join(errInvalidParameters, err)
	}
	other := strings.Fields(reference)
	for i, u := range other {
		u, err = checkAndUpdateURL(u)
		if err != nil {
			slog.Error("Incorrect reference's URL", logging.Error(err))
			return errors.Join(errInvalidParameters, err)
		}
		other[i] = u
	}

	slog.Debug("Node to check", "URL", node)
	slog.Debug("Reference nodes", "count", len(other), "nodes", other)

	urls := append([]string{node}, other...)
	slog.Debug("Requesting height from nodes", "count", len(urls))

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

	clients := make([]*client.Client, len(urls))
	for i, u := range urls {
		c, cErr := client.NewClient(client.Options{BaseUrl: u, Client: &http.Client{}})
		if cErr != nil {
			slog.Error("Failed to create client", slog.String("URL", u), logging.Error(cErr))
			return errors.Join(errFailure, cErr)
		}
		clients[i] = c
	}

	hs, err := heights(ctx, clients)
	if err != nil {
		slog.Error("Failed to retrieve heights from all nodes", logging.Error(err))
		if errors.Is(ctx.Err(), context.Canceled) {
			return errors.Join(errUserTermination, err)
		}
		return errors.Join(errUnavailable, err)
	}
	for i, h := range hs {
		slog.Debug("Height at node", "node", i, "height", h)
	}

	stop := slices.Min(hs)
	slog.Info("Lowest height", "height", stop)

	ch, err := findLastCommonHeight(ctx, clients, 1, stop)
	if err != nil {
		slog.Error("Failed to find last common height", logging.Error(err))
		if errors.Is(ctx.Err(), context.Canceled) {
			return errors.Join(errUserTermination, err)
		}
		return err
	}

	h := hs[0]
	slog.Debug("Node height", "height", h)
	refLowest := slices.Min(hs[1:])
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
				"But if you want to rollback manually, refer to the documentation at " +
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
			slog.Warn("Please, refer to the documentation at " +
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
		return "", fmt.Errorf("failed to parse URL '%s': %w", s, err)
	}
	if u.Scheme == "" {
		u.Scheme = defaultScheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}
	return u.String(), nil
}

func findLastCommonHeight(ctx context.Context, clients []*client.Client, start, stop uint64) (uint64, error) {
	var r uint64
	for start <= stop {
		select {
		case <-ctx.Done():
			return 0, errors.Join(errUserTermination, ctx.Err())
		default:
			middle := (start + stop) / 2
			c, err := differentIDsCount(ctx, clients, middle)
			if err != nil {
				return 0, fmt.Errorf("failed to get blocks signatures at height %d: %w", middle, err)
			}
			if c >= 2 {
				stop = middle - 1
				r = stop
			} else {
				start = middle + 1
				r = middle
			}
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

func differentIDsCount(ctx context.Context, clients []*client.Client, height uint64) (int, error) {
	ch := make(chan nodeBlockInfo, len(clients))
	info := make(map[int]nodeBlockInfo)
	m := make(map[proto.BlockID]bool)
	for i, c := range clients {
		go func(id int, cl *client.Client) {
			ctx1, cancel1 := context.WithTimeout(ctx, defaultTimeout)
			defer cancel1()
			header, resp, hErr := cl.Blocks.HeadersAt(ctx1, height)
			if hErr != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					ctx2, cancel2 := context.WithTimeout(ctx, defaultTimeout)
					defer cancel2()
					block, _, bErr := cl.Blocks.At(ctx2, height)
					if bErr != nil {
						ch <- nodeBlockInfo{id: id, err: bErr}
						return
					}
					ch <- nodeBlockInfo{id: id, height: block.Height, blockID: block.ID, generator: block.Generator, blockTime: block.Timestamp}
					return
				}
				ch <- nodeBlockInfo{id: id, err: hErr}
				return
			}
			ch <- nodeBlockInfo{id: id, height: header.Height, blockID: header.ID, generator: header.Generator, blockTime: header.Timestamp}
		}(i, c)
	}
	for range clients {
		bi := <-ch
		if bi.err != nil {
			return 0, fmt.Errorf("failed to get block header from %dth client: %w", bi.id, bi.err)
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

func showUsageAndExit() {
	_, _ = fmt.Fprintf(os.Stderr, "\nUsage of chaincmp %s\n", version)
	flag.PrintDefaults()
}

type nodeHeight struct {
	id     int
	height uint64
	err    error
}

func height(ctx context.Context, c *client.Client, id int, ch chan nodeHeight) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	bh, _, err := c.Blocks.Height(ctx)
	if err != nil {
		ch <- nodeHeight{id: id, err: err}
		return
	}
	ch <- nodeHeight{id: id, height: bh.Height}
}

func heights(ctx context.Context, clients []*client.Client) ([]uint64, error) {
	ch := make(chan nodeHeight, len(clients))
	heights := make(map[int]uint64)

	for i, c := range clients {
		go height(ctx, c, i, ch)
	}

	for range clients {
		nh := <-ch
		if nh.err != nil {
			return nil, fmt.Errorf("failed to retrieve height from %dth client: %w", nh.id, nh.err)
		}
		heights[nh.id] = nh.height
	}

	r := make([]uint64, len(heights))
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
