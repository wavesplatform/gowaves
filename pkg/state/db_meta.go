package state

import (
	stderrs "errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
)

const (
	dbMetaFileName = ".db.meta"
	maxDBMetaSize  = proto.MiB // limits the size of the DB meta file read
)

// checkAndUpdateDBMeta checks the compatibility of the state with the given parameters and updates the DB meta file.
// The DB meta file is created if it doesn't exist or is empty.
func checkAndUpdateDBMeta(dataDir string, amend bool, params StateParams) (stateInfo, error) {
	info, ok, err := loadDBMeta(dataDir)
	if err != nil {
		return stateInfo{}, err
	}
	if !ok {
		info = stateInfo{
			Version:            StateVersion,
			Amend:              amend,
			HasExtendedAPIData: params.StoreExtendedApiData,
			HasStateHashes:     params.BuildStateHashes,
			DBCompressionAlgo:  params.DbParams.CompressionAlgo,
		}
	}
	info, err = checkAndUpdateCompatibilityV2(info, params, amend) // check and update compatibility
	if err != nil {
		return stateInfo{}, wrapErr(stateerr.IncompatibilityError, err)
	}
	if wErr := writeDBMeta(dataDir, info); wErr != nil {
		return stateInfo{}, wErr
	}
	return info, nil
}

// readDBMeta reads the DB meta file, it returns an error if the file doesn't exist or is empty.
func readDBMeta(dataDir string) (stateInfo, error) {
	info, ok, err := loadDBMeta(dataDir)
	if err != nil {
		return stateInfo{}, err
	}
	if !ok {
		return stateInfo{}, errors.New("DB meta file not found or empty")
	}
	return info, nil
}

// loadDBMeta reads the DB meta file. The returned flag is false if the file doesn't exist or is empty.
func loadDBMeta(dataDir string) (_ stateInfo, _ bool, err error) {
	f, err := os.Open(filepath.Join(filepath.Clean(dataDir), dbMetaFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return stateInfo{}, false, nil
	}
	if err != nil {
		return stateInfo{}, false, errors.Wrap(err, "failed to open DB meta file")
	}
	defer func() {
		if clErr := f.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrap(clErr, "failed to close DB meta file"))
		}
	}()
	data, err := io.ReadAll(io.LimitReader(f, maxDBMetaSize))
	if err != nil {
		return stateInfo{}, false, errors.Wrap(err, "failed to read DB meta file")
	}
	if len(data) == 0 {
		return stateInfo{}, false, nil
	}
	var info stateInfo
	if umErr := info.unmarshalBinary(data); umErr != nil {
		return stateInfo{}, false, errors.Wrap(umErr, "failed to unmarshal DB meta file")
	}
	return info, true, nil
}

// writeDBMeta replaces the DB meta file atomically: the data is written and synced to a temporary file in the same
// directory, which is renamed to the meta file afterward. The previous meta file stays intact if writing fails.
func writeDBMeta(dataDir string, info stateInfo) (err error) {
	data, err := info.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal DB meta")
	}
	dataDir = filepath.Clean(dataDir)
	f, err := os.CreateTemp(dataDir, dbMetaFileName+".*.tmp") // the file is created with 0600 permissions
	if err != nil {
		return errors.Wrap(err, "failed to create temporary DB meta file")
	}
	tmpName := f.Name()
	defer func() {
		if err != nil {
			_ = f.Close() // the file can be already closed, the error is irrelevant in this case
			if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
				err = stderrs.Join(err, errors.Wrap(rmErr, "failed to remove temporary DB meta file"))
			}
		}
	}()
	if _, wErr := f.Write(data); wErr != nil {
		return errors.Wrap(wErr, "failed to write temporary DB meta file")
	}
	if sErr := f.Sync(); sErr != nil {
		return errors.Wrap(sErr, "failed to sync temporary DB meta file")
	}
	if clErr := f.Close(); clErr != nil {
		return errors.Wrap(clErr, "failed to close temporary DB meta file")
	}
	// Rename replaces the existing file atomically on POSIX systems, on Windows MoveFileEx is used with
	// MOVEFILE_REPLACE_EXISTING flag.
	if rnErr := os.Rename(tmpName, filepath.Join(dataDir, dbMetaFileName)); rnErr != nil {
		return errors.Wrap(rnErr, "failed to replace DB meta file")
	}
	return syncDir(dataDir)
}

// syncDir makes the rename of a file in the directory durable. Directories can't be synced on Windows,
// the rename is durable there without it.
func syncDir(dir string) (err error) {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(filepath.Clean(dir))
	if err != nil {
		return errors.Wrap(err, "failed to open directory")
	}
	defer func() {
		if clErr := d.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrap(clErr, "failed to close directory"))
		}
	}()
	if sErr := d.Sync(); sErr != nil {
		return errors.Wrap(sErr, "failed to sync directory")
	}
	return nil
}
