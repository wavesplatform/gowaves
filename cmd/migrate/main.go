// Command migrate converts the state storage of the node from one of the previous versions to the current version.
// The node must be stopped during the migration. Make a backup of the state directory before the migration.
// The migration can be safely restarted after an interruption.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ccoveille/go-safecast/v2"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Failed to migrate state", logging.Error(err))
		os.Exit(1)
	}
	slog.Info("State migration completed successfully")
}

func run() error {
	var (
		lp        = logging.Parameters{}
		statePath = flag.String("state-path", "", "Path to node's state directory")
	)
	lp.Initialize()
	flag.Parse()
	if err := lp.Parse(); err != nil {
		return fmt.Errorf("failed to parse application parameters: %w", err)
	}
	slog.SetDefault(slog.New(logging.DefaultHandler(lp)))
	slog.Info("Gowaves State Migration", "version", versioning.Version, "stateVersion", state.StateVersion)

	if *statePath == "" {
		return errors.New("empty state path")
	}
	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	if _, rErr := fdlimit.RaiseMaxFDs(maxFDs); rErr != nil {
		return fmt.Errorf("initialization failed: %w", rErr)
	}
	params := state.DefaultStorageParams().DbParams
	const fdSigma = 10
	c, err := safecast.Convert[int](maxFDs - fdSigma)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	params.OpenFilesCacheCapacity = c

	slog.Info("Migrating state", "path", *statePath)
	return state.MigrateState(*statePath, params)
}
