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
	"runtime"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/client"
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
	version = "v0.0.0"
)

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	var (
		node               string
		statePath          string
		blockchainType     string
		height             uint64
		extendedAPI        bool
		compare            bool
		search             bool
		showHelp           bool
		showVersion        bool
		onlyLegacy         bool
		disableBloomFilter bool
	)

	slog.SetDefault(slog.New(logging.NewHandler(logging.LoggerPrettyNoColor, slog.LevelInfo)))

	flag.StringVar(&node, "node", "", "Path to node's state folder")
	flag.StringVar(&statePath, "state-path", "", "Path to node's state folder")
	flag.StringVar(&blockchainType, "blockchain-type", "mainnet", "Blockchain type mainnet/testnet/stagenet, default value is mainnet")
	flag.Uint64Var(&height, "at-height", 0, "Height to get state hash at, defaults to the top most value")
	flag.BoolVar(&extendedAPI, "extended-api", false, "Open state with extended API")
	flag.BoolVar(&compare, "compare", false, "Compare the state hash with the node's state hash at the same height")
	flag.BoolVar(&search, "search", false, "Search for the topmost equal state hashes")
	flag.BoolVar(&showHelp, "help", false, "Show usage information and exit")
	flag.BoolVar(&showVersion, "version", false, "Print version information and quit")
	flag.BoolVar(&onlyLegacy, "legacy", false, "Compare only legacy state hashes")
	flag.BoolVar(&disableBloomFilter, "disable-bloom", false, "Disable bloom filter")
	flag.Parse()

	if showHelp {
		showUsage()
		return nil
	}
	if showVersion {
		fmt.Printf("Waves RIDE Appraiser %s\n", version)
		return nil
	}

	if search {
		compare = true
	}

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

	if statePath == "" || len(strings.Fields(statePath)) > 1 {
		slog.Error("Invalid path to state", "path", statePath)
		return errors.New("invalid state path")
	}

	ss, err := settings.BlockchainSettingsByTypeName(blockchainType)
	if err != nil {
		slog.Error("Failed to load blockchain settings", logging.Error(err))
		return err
	}

	params := state.DefaultStateParams()
	params.VerificationGoroutinesNum = 2 * runtime.NumCPU()
	params.DbParams.WriteBuffer = 16 * MB
	params.DbParams.DisableBloomFilter = disableBloomFilter
	params.StoreExtendedApiData = extendedAPI
	params.BuildStateHashes = true
	params.ProvideExtendedApi = false

	st, err := state.NewState(statePath, false, params, ss, false)
	if err != nil {
		slog.Error("Failed to open state", slog.String("path", statePath), logging.Error(err))
		return err
	}
	defer func(st state.StateModifier) {
		if err := st.Close(); err != nil {
			slog.Error("Failed to close state", logging.Error(err))
			os.Exit(1)
		}
	}(st)

	c, err := createClient(node)
	if err != nil {
		return err
	}
	if compare {
		if cmpErr := compareAtHeight(st, c, 1, onlyLegacy); cmpErr != nil {
			return cmpErr
		}
	}

	if height == 0 { // determine the topmost height
		h, err := st.Height()
		if err != nil {
			slog.Error("Failed to get current blockchain height", logging.Error(err))
			return err
		}
		height = h
	}
	lsh, err := getLocalStateHash(st, height)
	if err != nil {
		slog.Error("Failed to get state hash", slog.Any("height", height), logging.Error(err))
		return err
	}
	slog.Info("State hash at height", "height", height, "stateHash", stateHashToString(lsh))
	if compare {
		ok, rsh, cmpErr := compareWithRemote(lsh, c, height, onlyLegacy)
		if cmpErr != nil {
			slog.Error("Failed to compare", logging.Error(cmpErr))
			return cmpErr
		}
		if !ok {
			slog.Warn("[NOT OK] State hashes are different")
			slog.Info("Remote state hash", "height", height, "stateHash", stateHashToString(rsh))
			if search {
				if sErr := searchLastEqualStateLash(c, st, height, onlyLegacy); sErr != nil {
					return sErr
				}
			}
			return nil
		}
		slog.Info("[OK] State hash is equal to remote state hash at the same height")
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

func compareAtHeight(st state.StateInfo, c *client.Client, h uint64, onlyLegacy bool) error {
	rsh, err := getRemoteStateHash(c, h)
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

func searchLastEqualStateLash(c *client.Client, st state.State, height proto.Height, onlyLegacy bool) error {
	h, err := findLastEqualStateHashes(c, st, height, onlyLegacy)
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
	rsh, err := getRemoteStateHash(c, h+1)
	if err != nil {
		slog.Error("Failed to get remote state hash at height 1", logging.Error(err))
		return err
	}
	slog.Info("Remote state hash at height", "height", h+1, "stateHash", stateHashToString(rsh))
	return nil
}

func findLastEqualStateHashes(c *client.Client, st state.State, stop uint64, onlyLegacy bool) (uint64, error) {
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
		rsh, err = getRemoteStateHash(c, middle)
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
	sh *proto.StateHashDebug,
	c *client.Client,
	h uint64,
	onlyLegacy bool,
) (bool, *proto.StateHashDebug, error) {
	rsh, err := getRemoteStateHash(c, h)
	if err != nil {
		return false, nil, err
	}
	ok, err := compareStateHashes(sh, rsh, onlyLegacy)
	return ok, rsh, err
}

func getRemoteStateHash(c *client.Client, h uint64) (*proto.StateHashDebug, error) {
	sh, _, err := c.Debug.StateHashDebug(context.Background(), h)
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
