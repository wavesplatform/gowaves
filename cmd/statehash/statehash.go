package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"go.uber.org/zap"
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
		node           string
		statePath      string
		blockchainType string
		height         uint64
		extendedAPI    bool
		compare        bool
		search         bool
		showHelp       bool
		showVersion    bool
	)

	common.SetupLogger("INFO")

	flag.StringVar(&node, "node", "", "Path to node's state folder")
	flag.StringVar(&statePath, "state-path", "", "Path to node's state folder")
	flag.StringVar(&blockchainType, "blockchain-type", "mainnet", "Blockchain type mainnet/testnet/stagenet, default value is mainnet")
	flag.Uint64Var(&height, "at-height", 0, "Height to get state hash at, defaults to the top most value")
	flag.BoolVar(&extendedAPI, "extended-api", false, "Open state with extended API")
	flag.BoolVar(&compare, "compare", false, "Compare the state hash with the node's state hash at the same height")
	flag.BoolVar(&search, "search", false, "Search for the topmost equal state hashes")
	flag.BoolVar(&showHelp, "help", false, "Show usage information and exit")
	flag.BoolVar(&showVersion, "version", false, "Print version information and quit")
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
		zap.S().Fatalf("Initialization error: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}

	if statePath == "" || len(strings.Fields(statePath)) > 1 {
		zap.S().Errorf("Invalid path to state '%s'", statePath)
		return errors.New("invalid state path")
	}

	ss, err := settings.BlockchainSettingsByTypeName(blockchainType)
	if err != nil {
		zap.S().Errorf("Failed to load blockchain settings: %v", err)
		return err
	}

	params := state.DefaultStateParams()
	params.VerificationGoroutinesNum = 2 * runtime.NumCPU()
	params.DbParams.WriteBuffer = 16 * MB
	params.StoreExtendedApiData = extendedAPI
	params.BuildStateHashes = true
	params.ProvideExtendedApi = false

	st, err := state.NewState(statePath, false, params, ss)
	if err != nil {
		zap.S().Errorf("Failed to open state at '%s': %v", statePath, err)
		return err
	}
	defer func() {
		if err := st.Close(); err != nil {
			zap.S().Fatalf("Failed to close State: %v", err)
		}
	}()

	var c *client.Client
	if compare {
		if node == "" || len(strings.Fields(node)) > 1 {
			zap.S().Errorf("Invalid node's URL '%s'", node)
			return errors.New("invalid node")
		}
		node, err := checkAndUpdateURL(node)
		if err != nil {
			zap.S().Errorf("Incorrect node's URL: %s", err.Error())
			return err
		}
		c, err = client.NewClient(client.Options{BaseUrl: node, Client: &http.Client{}})
		if err != nil {
			zap.S().Errorf("Failed to create client for URL '%s': %s", node, err)
			return err
		}
		rsh, err := getRemoteStateHash(c, 1)
		if err != nil {
			zap.S().Errorf("Failed to get remote state hash at height 1: %v", err)
			return err
		}
		lsh, err := st.StateHashAtHeight(1)
		if err != nil {
			zap.S().Errorf("Failed to get local state hash at 1: %v", err)
			return err

		}
		_, err = compareStateHashes(lsh, rsh)
		if err != nil {
			zap.S().Errorf("State hashes at height 1 are different: %v", err)
			return err
		}
	}

	if height == 0 {
		h, err := st.Height()
		if err != nil {
			zap.S().Errorf("Failed to get current blockchain height: %v", err)
			return err
		}
		height = h
	}
	lsh, err := st.StateHashAtHeight(height)
	if err != nil {
		zap.S().Errorf("Failed to get state hash at %d: %v", height, err)
		return err
	}
	zap.S().Infof("State hash at height %d:\n%s", height, stateHashToString(lsh))
	if compare {
		ok, rsh, err := compareWithRemote(lsh, c, height)
		if err != nil {
			zap.S().Errorf("Failed to compare: %v", err)
			return err
		}
		if !ok {
			zap.S().Warnf("[NOT OK] State hashes are different")
			zap.S().Infof("Remote state hash at height %d:\n%s", height, stateHashToString(rsh))
			if search {
				h, err := findLastEqualStateHashes(c, st, height)
				if err != nil {
					zap.S().Errorf("Failed to find equal hashes: %v", err)
					return err
				}
				zap.S().Infof("State hashes are equal at height %d", h)
				lsh, err = st.StateHashAtHeight(h + 1)
				if err != nil {
					zap.S().Errorf("Failed to get state hash at %d: %v", h+1, err)
					return err
				}
				zap.S().Infof("Local state hash at height %d:\n%s", h+1, stateHashToString(lsh))
				rsh, err = getRemoteStateHash(c, h+1)
				if err != nil {
					zap.S().Errorf("Failed to get remote state hash at height 1: %v", err)
					return err
				}
				zap.S().Infof("Remote state hash at height %d:\n%s", h+1, stateHashToString(rsh))
			}
			return nil
		}
		zap.S().Info("[OK] State hash is equal to remote state hash at the same height")
	}
	return nil
}

func findLastEqualStateHashes(c *client.Client, st state.State, stop uint64) (uint64, error) {
	var err error
	var r uint64
	var lsh, rsh *proto.StateHash
	var start uint64 = 1
	for start <= stop {
		middle := (start + stop) / 2
		lsh, err = st.StateHashAtHeight(middle)
		if err != nil {
			return middle, err
		}
		rsh, err = getRemoteStateHash(c, middle)
		if err != nil {
			return middle, err
		}
		ok, err := compareStateHashes(lsh, rsh)
		if err != nil {
			return middle, err
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

func stateHashToString(sh *proto.StateHash) string {
	js, err := sh.MarshalJSON()
	if err != nil {
		zap.S().Fatalf("Failed to render state hash to text: %v", err)
	}
	return string(js)
}

func compareStateHashes(sh1, sh2 *proto.StateHash) (bool, error) {
	if sh1.BlockID != sh2.BlockID {
		return false, fmt.Errorf("different block IDs: '%s' != '%s'", sh1.BlockID.String(), sh2.BlockID.String())
	}
	return sh1.SumHash == sh2.SumHash, nil
}

func compareWithRemote(sh *proto.StateHash, c *client.Client, h uint64) (bool, *proto.StateHash, error) {
	rsh, err := getRemoteStateHash(c, h)
	if err != nil {
		return false, nil, err
	}
	ok, err := compareStateHashes(sh, rsh)
	return ok, rsh, err
}

func getRemoteStateHash(c *client.Client, h uint64) (*proto.StateHash, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sh, _, err := c.Debug.StateHash(ctx, h)
	return sh, err
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
		return "", fmt.Errorf("failed to parse URL '%s': %v", s, err)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}
	return u.String(), nil
}
