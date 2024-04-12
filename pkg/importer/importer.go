package importer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	KiB = 1024
	MiB = 1024 * KiB

	initTotalBatchSize = 5 * MiB
	sizeAdjustment     = 1 * MiB
	uint32Size         = 4

	MaxTotalBatchSize  = 20 * MiB
	MaxBlocksBatchSize = 50000
	MaxBlockSize       = 2 * MiB
)

var errNoop = errors.New("noop")

type State interface {
	AddBlocks(blocks [][]byte) error
	AddBlocksWithSnapshots(blocks [][]byte, snapshots []*proto.BlockSnapshot) error
	WavesAddressesNumber() (uint64, error)
	WavesBalance(account proto.Recipient) (uint64, error)
	AssetBalance(account proto.Recipient, assetID proto.AssetID) (uint64, error)
	ShouldPersistAddressTransactions() (bool, error)
	PersistAddressTransactions() error
}

type Importer interface {
	SkipToHeight(context.Context, uint64) error
	Import(context.Context, uint64) error
	Close() error
}

func maybePersistTxs(st State) error {
	// Check if we need to persist transactions for extended API.
	persistTxs, err := st.ShouldPersistAddressTransactions()
	if err != nil {
		return err
	}
	if persistTxs {
		return st.PersistAddressTransactions()
	}
	return nil
}

// ApplyFromFile reads blocks from blockchainPath, applying them from height startHeight and until nBlocks+1.
// Setting optimize to true speeds up the import, but it is only safe when importing blockchain from scratch
// when no rollbacks are possible at all.
func ApplyFromFile(
	ctx context.Context,
	scheme proto.Scheme,
	st State,
	blockchainPath string,
	nBlocks, startHeight uint64,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	imp, err := NewBlocksImporter(scheme, st, blockchainPath)
	if err != nil {
		return err
	}
	defer func() {
		if clErr := imp.Close(); clErr != nil {
			zap.S().Fatalf("Failed to close importer: %v", clErr)
		}
	}()
	err = imp.SkipToHeight(ctx, startHeight)
	if err != nil {
		return err
	}
	return imp.Import(ctx, nBlocks)
}

func CheckBalances(st State, balancesPath string) error {
	balances, err := os.Open(filepath.Clean(balancesPath))
	if err != nil {
		return errors.Wrapf(err, "failed to open balances file %q", balancesPath)
	}
	defer func() {
		if closeErr := balances.Close(); closeErr != nil {
			zap.S().Fatalf("Failed to close balances file: %v", closeErr)
		}
	}()
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("failed to decode state: %v", err)
	}
	addressesNumber, err := st.WavesAddressesNumber()
	if err != nil {
		return errors.Errorf("failed to get number of waves addresses: %v", err)
	}
	properAddressesNumber := uint64(len(state))
	if properAddressesNumber != addressesNumber {
		return errors.Errorf("number of addresses differ: %d and %d", properAddressesNumber, addressesNumber)
	}
	for addrStr, properBalance := range state {
		addr, err := proto.NewAddressFromString(addrStr)
		if err != nil {
			return errors.Errorf("faied to convert string to address: %v", err)
		}
		balance, err := st.WavesBalance(proto.NewRecipientFromAddress(addr))
		if err != nil {
			return errors.Errorf("failed to get balance: %v", err)
		}
		if balance != properBalance {
			return errors.Errorf("balances for address %v differ: %d and %d", addr, properBalance, balance)
		}
	}
	return nil
}
