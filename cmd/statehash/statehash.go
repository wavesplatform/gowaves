package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
)

const (
	MB = 1024 * 1024
)

var (
	version      = "v0.0.0"
	errEarlyExit = errors.New("early exit")
)

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

type runConfig struct {
	node            string
	statePath       string
	blockchainType  string
	height          uint64
	extendedAPI     bool
	compare         bool
	search          bool
	onlyLegacy      bool
	disableBloom    bool
	compressionAlgo keyvalue.CompressionAlgo
}

// parseFlags parses command-line flags and returns the configuration.
// It returns nil config if the program should exit early (help/version).
func parseFlags() (*runConfig, error) {
	var cfg runConfig
	var showHelp, showVersion bool

	flag.StringVar(&cfg.node, "node", "", "Path to node's state folder")
	flag.StringVar(&cfg.statePath, "state-path", "", "Path to node's state folder")
	flag.StringVar(&cfg.blockchainType, "blockchain-type", "mainnet",
		"Blockchain type mainnet/testnet/stagenet, default value is mainnet")
	flag.Uint64Var(&cfg.height, "at-height", 0, "Height to get state hash at, defaults to the top most value")
	flag.BoolVar(&cfg.extendedAPI, "extended-api", false, "Open state with extended API")
	flag.BoolVar(&cfg.compare, "compare", false, "Compare the state hash with the node's state hash at the same height")
	flag.BoolVar(&cfg.search, "search", false, "Search for the topmost equal state hashes")
	flag.BoolVar(&showHelp, "help", false, "Show usage information and exit")
	flag.BoolVar(&showVersion, "version", false, "Print version information and quit")
	flag.BoolVar(&cfg.onlyLegacy, "legacy", false, "Compare only legacy state hashes")
	flag.BoolVar(&cfg.disableBloom, "disable-bloom", false, "Disable bloom filter")
	flag.TextVar(&cfg.compressionAlgo, "db-compression-algo", keyvalue.CompressionDefault,
		fmt.Sprintf("Set the compression algorithm for the state database. Supported: %v",
			keyvalue.CompressionAlgoStrings(),
		),
	)
	flag.Parse()

	if showHelp {
		showUsage()
		return nil, errEarlyExit
	}
	if showVersion {
		fmt.Printf("Waves RIDE Appraiser %s\n", version)
		return nil, errEarlyExit
	}
	if cfg.search {
		cfg.compare = true
	}
	if cfg.statePath == "" || len(strings.Fields(cfg.statePath)) > 1 {
		slog.Error("Invalid path to state", "path", cfg.statePath)
		return nil, errors.New("invalid state path")
	}
	return &cfg, nil
}

func setupFDLimits() {
	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		slog.Error("Initialization failed", logging.Error(err))
		os.Exit(1)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		slog.Error("Initialization failed", logging.Error(err))
		os.Exit(1)
	}
}

func openState(ctx context.Context, cfg *runConfig) (state.State, error) {
	ss, err := settings.BlockchainSettingsByTypeName(cfg.blockchainType)
	if err != nil {
		slog.Error("Failed to load blockchain settings", logging.Error(err))
		return nil, err
	}

	params := state.DefaultStateParams()
	params.VerificationGoroutinesNum = 2 * runtime.NumCPU()
	params.DbParams.WriteBuffer = 16 * MB
	params.DbParams.DisableBloomFilter = cfg.disableBloom
	params.StoreExtendedApiData = cfg.extendedAPI
	params.BuildStateHashes = true
	params.ProvideExtendedApi = false
	params.DbParams.CompressionAlgo = cfg.compressionAlgo

	st, err := state.NewState(ctx, cfg.statePath, false, params, ss, false, nil)
	if err != nil {
		slog.Error("Failed to open state", slog.String("path", cfg.statePath), logging.Error(err))
		return nil, err
	}
	return st, nil
}

func run() (err error) {
	slog.SetDefault(slog.New(logging.NewHandler(logging.LoggerPrettyNoColor, slog.LevelInfo)))

	cfg, err := parseFlags()
	if err != nil {
		if errors.Is(err, errEarlyExit) {
			return nil
		}
		return err
	}

	setupFDLimits()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

	st, err := openState(ctx, cfg)
	if err != nil {
		return err
	}
	defer func(st state.State) {
		if clErr := st.Close(); clErr != nil {
			slog.Error("Failed to close state", logging.Error(clErr))
			err = errors.Join(err, clErr)
		}
	}(st)

	c, err := createClient(cfg.node)
	if err != nil {
		return err
	}
	if cfg.compare {
		if cmpErr := compareAtHeight(ctx, st, c, 1, cfg.onlyLegacy); cmpErr != nil {
			return cmpErr
		}
	}

	height := cfg.height
	if height == 0 { // determine the topmost height
		height, err = st.Height()
		if err != nil {
			slog.Error("Failed to get current blockchain height", logging.Error(err))
			return err
		}
	}
	lsh, err := getLocalStateHash(st, height)
	if err != nil {
		slog.Error("Failed to get state hash", slog.Any("height", height), logging.Error(err))
		return err
	}
	slog.Info("State hash at height", "height", height, "stateHash", stateHashToString(lsh))
	if cfg.compare {
		return compareAndSearch(ctx, lsh, c, st, height, cfg)
	}
	return nil
}

func compareAndSearch(
	ctx context.Context,
	lsh *proto.StateHashDebug,
	c *client.Client,
	st state.State,
	height uint64,
	cfg *runConfig,
) error {
	ok, rsh, cmpErr := compareWithRemote(ctx, lsh, c, height, cfg.onlyLegacy)
	if cmpErr != nil {
		slog.Error("Failed to compare", logging.Error(cmpErr))
		return cmpErr
	}
	if ok {
		slog.Info("[OK] State hash is equal to remote state hash at the same height")
		return nil
	}
	slog.Warn("[NOT OK] State hashes are different")
	slog.Info("Remote state hash", "height", height, "stateHash", stateHashToString(rsh))
	if cfg.search {
		return searchLastEqualStateLash(ctx, c, st, height, cfg.onlyLegacy)
	}
	return nil
}

func createClient(node string) (*client.Client, error) {
	if node == "" || len(strings.Fields(node)) > 1 {
		slog.Error("Invalid node's URL", "URL", node)
		return nil, errors.New("invalid node")
	}
	node, err := checkAndUpdateURL(node)
	if err != nil {
		slog.Error("Incorrect node's URL", logging.Error(err))
		return nil, err
	}
	c, err := client.NewClient(client.Options{BaseUrl: node, Client: &http.Client{}})
	if err != nil {
		slog.Error("Failed to create client", slog.String("URL", node), logging.Error(err))
		return nil, err
	}
	return c, nil
}

func compareAtHeight(ctx context.Context, st state.StateInfo, c *client.Client, h uint64, onlyLegacy bool) error {
	rsh, err := getRemoteStateHash(ctx, c, h)
	if err != nil {
		slog.Error("Failed to get remote state hash", slog.Any("height", h), logging.Error(err))
		return err
	}
	lsh, err := getLocalStateHash(st, h)
	if err != nil {
		slog.Error("Failed to get local state hash", slog.Any("height", h), logging.Error(err))
		return err
	}
	_, err = compareStateHashes(lsh, rsh, onlyLegacy)
	if err != nil {
		slog.Error("State hashes are different", slog.Any("height", h), logging.Error(err))
		return err
	}
	return nil
}

func searchLastEqualStateLash(
	ctx context.Context,
	c *client.Client,
	st state.State,
	height proto.Height,
	onlyLegacy bool,
) error {
	h, err := findLastEqualStateHashes(ctx, c, st, height, onlyLegacy)
	if err != nil {
		slog.Error("Failed to find equal hashes", logging.Error(err))
		return err
	}
	slog.Info("State hashes are equal", "height", h)
	lsh, err := getLocalStateHash(st, h+1)
	if err != nil {
		slog.Error("Failed to get state hash", slog.Any("height", h+1), logging.Error(err))
		return err
	}
	slog.Info("Local state hash at height", "height", h+1, "stateHash", stateHashToString(lsh))
	rsh, err := getRemoteStateHash(ctx, c, h+1)
	if err != nil {
		slog.Error("Failed to get remote state hash at height 1", logging.Error(err))
		return err
	}
	slog.Info("Remote state hash at height", "height", h+1, "stateHash", stateHashToString(rsh))
	return nil
}

func findLastEqualStateHashes(
	ctx context.Context,
	c *client.Client,
	st state.State,
	stop uint64,
	onlyLegacy bool,
) (uint64, error) {
	var err error
	var r uint64
	var lsh, rsh *proto.StateHashDebug
	var start uint64 = 1
	for start <= stop {
		middle := (start + stop) / 2
		lsh, err = getLocalStateHash(st, middle)
		if err != nil {
			return middle, err
		}
		rsh, err = getRemoteStateHash(ctx, c, middle)
		if err != nil {
			return middle, err
		}
		ok, cmpErr := compareStateHashes(lsh, rsh, onlyLegacy)
		if cmpErr != nil {
			return middle, cmpErr
		}
		if !ok {
			stop = middle - 1
			r = stop
		} else {
			start = middle + 1
			r = middle
		}
	}
	return r, nil
}

func stateHashToString(sh *proto.StateHashDebug) string {
	js, err := json.Marshal(sh)
	if err != nil {
		slog.Error("Failed to render state hash to text", logging.Error(err))
		os.Exit(1)
	}
	return string(js)
}

func compareStateHashes(sh1, sh2 *proto.StateHashDebug, onlyLegacy bool) (bool, error) {
	if sh1.BlockID != sh2.BlockID {
		return false, fmt.Errorf("different block IDs: '%s' != '%s'", sh1.BlockID.String(), sh2.BlockID.String())
	}
	legacyEqual := sh1.SumHash == sh2.SumHash
	if onlyLegacy {
		return legacyEqual, nil
	}
	return legacyEqual && sh1.SnapshotHash == sh2.SnapshotHash, nil
}

func compareWithRemote(
	ctx context.Context,
	sh *proto.StateHashDebug,
	c *client.Client,
	h uint64,
	onlyLegacy bool,
) (bool, *proto.StateHashDebug, error) {
	rsh, err := getRemoteStateHash(ctx, c, h)
	if err != nil {
		return false, nil, err
	}
	ok, err := compareStateHashes(sh, rsh, onlyLegacy)
	return ok, rsh, err
}

func getRemoteStateHash(ctx context.Context, c *client.Client, h uint64) (*proto.StateHashDebug, error) {
	sh, _, err := c.Debug.StateHashDebug(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to get state hash at %d height: %w", h, err)
	}
	return sh, nil
}

func getLocalStateHash(st state.StateInfo, h uint64) (*proto.StateHashDebug, error) {
	const localVersion = "local"
	lsh, err := st.LegacyStateHashAtHeight(h)
	if err != nil {
		return nil, fmt.Errorf("failed to get legacy state hash at %d height: %w", h, err)
	}
	snapSH, err := st.SnapshotStateHashAtHeight(h)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot state hash at %d height: %w", h, err)
	}
	shd := proto.NewStateHashJSDebug(*lsh, h, localVersion, snapSH)
	return &shd, nil
}

func showUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "\nUsage of statehash %s\n", version)
	flag.PrintDefaults()
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
		u.Scheme = "http"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}
	return u.String(), nil
}
