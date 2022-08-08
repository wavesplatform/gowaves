package storage

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	// if you change peers storage data format, you have to increment peersStorageCurrentVersion
	peersStorageCurrentVersion = 1
	peersStorageDir            = "peers_storage"
)

type CBORStorage struct {
	rwMutex           sync.RWMutex
	storageDir        string
	suspended         suspendedPeers
	suspendedFilePath string
	known             knownPeers // Map of all ever known peers with a publicly available declared address and the last connection attempt timestamp.
	knownFilePath     string
}

func NewCBORStorage(baseDir string, now time.Time) (*CBORStorage, error) {
	storageDir := filepath.Join(baseDir, peersStorageDir)
	return newCBORStorageInDir(storageDir, now, peersStorageCurrentVersion)
}

func newCBORStorageInDir(storageDir string, now time.Time, currVersion int) (*CBORStorage, error) {
	if err := os.MkdirAll(storageDir, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "failed to create peers storage directory %q", storageDir)
	}

	knownFile := knownFilePath(storageDir)
	if err := createFileIfNotExist(knownFile); err != nil {
		return nil, errors.Wrap(err, "failed to create known peers storage file")
	}
	suspendedFile := suspendedFilePath(storageDir)
	if err := createFileIfNotExist(suspendedFile); err != nil {
		return nil, errors.Wrap(err, "failed to create suspended peers storage file")
	}

	storage := &CBORStorage{
		storageDir:        storageDir,
		suspended:         suspendedPeers{},
		suspendedFilePath: suspendedFile,
		known:             knownPeers{},
		knownFilePath:     knownFile,
	}

	versionFile := storageVersionFilePath(storageDir)
	oldVersion, err := getPeersStorageVersion(versionFile)
	switch {
	case err == io.EOF:
		// Empty version file, set new version
		if err := storage.invalidateStorageAndUpdateVersion(versionFile, currVersion, oldVersion); err != nil {
			return nil, errors.Wrap(err, "failed set version to new peers storage")
		}
	case err != nil:
		return nil, errors.Wrap(err, "failed to validate peers storage version")
	}

	if oldVersion != currVersion {
		// Invalidating old peers storage
		zap.S().Debugf(
			"Detected different peers storage versions: old='%d', current='%d'. Removing old peers storage.",
			oldVersion,
			currVersion,
		)
		if err := storage.invalidateStorageAndUpdateVersion(versionFile, currVersion, oldVersion); err != nil {
			return nil, errors.Wrap(err, "failed invalidate storage and set new version to peers storage")
		}
	}

	if err := unmarshalCborFromFile(knownFile, &storage.known); err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "failed to load known peers from file %q", knownFile)
	}
	if err := unmarshalCborFromFile(suspendedFile, &storage.suspended); err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "failed to load suspended peers from file %q", suspendedFile)
	}

	if len(storage.suspended) != 0 {
		// Remove expired peers
		if err := storage.RefreshSuspended(now); err != nil {
			return nil, errors.Wrapf(err,
				"failed to refresh suspended peers while opening peers storage with path %q", storageDir)
		}
	}
	return storage, nil
}

func (bs *CBORStorage) Known(limit int) []KnownPeer {
	bs.rwMutex.RLock()
	defer bs.rwMutex.RUnlock()
	return bs.known.OldestFirst(limit)
}

// AddOrUpdateKnown adds known peers with now timestamp into peers storage with strong error guarantees.
func (bs *CBORStorage) AddOrUpdateKnown(known []KnownPeer, now time.Time) error {
	if len(known) == 0 {
		return nil
	}

	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	// Save existing known peers with their last attempt timestamps in backup
	backup := bs.unsafeKnownIntersection(known)

	nowInt := now.UnixNano()
	for _, k := range known {
		bs.known[k] = nowInt
	}

	if err := bs.unsafeSyncKnown(known, backup); err != nil {
		return errors.Wrapf(err, "failed to add known peers")
	}
	return nil
}

// DeleteKnown removes known peers from peers storage with strong error guarantees.
func (bs *CBORStorage) DeleteKnown(known []KnownPeer) error {
	if len(known) == 0 {
		return nil
	}

	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	// Save old values in backup
	backup := bs.unsafeKnownIntersection(known)
	// Delete entries from known map
	for _, k := range known {
		delete(bs.known, k)
	}

	// newEntries is nil because there is no new entries
	if err := bs.unsafeSyncKnown(nil, backup); err != nil {
		return errors.Wrap(err, "failed to delete known peers")
	}
	return nil
}

// DropKnown clear known in memory cache and truncates known peers storage file with strong error guarantee.
func (bs *CBORStorage) DropKnown() error {
	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()
	return bs.unsafeDropKnown()
}

func (bs *CBORStorage) Suspended(now time.Time) []SuspendedPeer {
	bs.rwMutex.RLock()
	defer bs.rwMutex.RUnlock()

	suspended := make([]SuspendedPeer, 0, len(bs.suspended))
	for _, s := range bs.suspended {
		if s.IsSuspended(now) {
			suspended = append(suspended, s)
		}
	}
	return suspended
}

// AddSuspended adds suspended peers into peers storage with strong error guarantees.
func (bs *CBORStorage) AddSuspended(suspended []SuspendedPeer) error {
	if len(suspended) == 0 {
		return nil
	}

	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	// Save old values in backup
	backup := bs.unsafeSuspendedIntersection(suspended)
	// Add new values into suspended map
	for _, s := range suspended {
		bs.suspended[s.IP] = s
	}

	if err := bs.unsafeSyncSuspended(suspended, backup); err != nil {
		return errors.Wrap(err, "failed to add suspended peers")
	}
	return nil
}

func (bs *CBORStorage) IsSuspendedIP(ip IP, now time.Time) bool {
	bs.rwMutex.RLock()
	defer bs.rwMutex.RUnlock()
	return bs.unsafeIsSuspendedIP(ip, now)
}

func (bs *CBORStorage) IsSuspendedIPs(ips []IP, now time.Time) []bool {
	if len(ips) == 0 {
		return nil
	}

	bs.rwMutex.RLock()
	defer bs.rwMutex.RUnlock()

	isSuspended := make([]bool, 0, len(ips))
	for _, ip := range ips {
		isSuspended = append(isSuspended, bs.unsafeIsSuspendedIP(ip, now))
	}
	return isSuspended
}

// DeleteSuspendedByIP removes suspended peers from peers storage with strong error guarantees.
// Note, that only IP field in input parameter will be used.
func (bs *CBORStorage) DeleteSuspendedByIP(suspended []SuspendedPeer) error {
	if len(suspended) == 0 {
		return nil
	}

	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	// Save old values in backup
	backup := bs.unsafeSuspendedIntersection(suspended)
	// Delete entries from known map
	for _, s := range suspended {
		delete(bs.suspended, s.IP)
	}

	// newEntries is nil because there is no new entries
	if err := bs.unsafeSyncSuspended(nil, backup); err != nil {
		return errors.Wrap(err, "failed to delete suspended peers")
	}
	return nil
}

// RefreshSuspended removes expired peers from suspended peers storage with strong error guarantee.
func (bs *CBORStorage) RefreshSuspended(now time.Time) error {
	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	var backup []SuspendedPeer
	for _, s := range bs.suspended {
		if !s.IsSuspended(now) {
			backup = append(backup, s)
			delete(bs.suspended, s.IP)
		}
	}
	if len(backup) == 0 {
		// No expired peers
		return nil
	}

	if err := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended); err != nil {
		// Restore previous values into map to eliminate side effects
		for _, b := range backup {
			bs.suspended[b.IP] = b
		}
		return errors.Wrap(err, "failed to refresh suspended peers and sync storage")
	}
	return nil
}

// DropSuspended clear suspended in memory cache and truncates suspended peers storage file with strong error guarantee.
func (bs *CBORStorage) DropSuspended() error {
	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()
	return bs.unsafeDropSuspended()
}

// DropStorage clear storage memory cache and truncates storage files.
// In case of error we can lose suspended peers storage file, but honestly it's almost impossible case.
func (bs *CBORStorage) DropStorage() error {
	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	suspendedBackup := bs.suspended
	if err := bs.unsafeDropSuspended(); err != nil {
		return errors.Wrap(err, "failed to drop suspended peers storage")
	}

	if err := bs.unsafeDropKnown(); err != nil {
		bs.suspended = suspendedBackup
		// It's almost impossible case, but if it happens we have inconsistency in suspended peers,
		// but honestly it's not fatal error
		if syncErr := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended); syncErr != nil {
			return errors.Wrapf(err, "failed to sync suspended peers storage from backup: %v", syncErr)
		}
		return errors.Wrap(err, "failed to drop known peers storage")
	}
	return nil
}

func (bs *CBORStorage) invalidateStorageAndUpdateVersion(versionFile string, currVersion, oldVersion int) error {
	if err := bs.DropStorage(); err != nil {
		return errors.Wrapf(err,
			"failed to drop peers storage in case of different versions, old='%d', current='%d'",
			oldVersion,
			currVersion,
		)
	}
	if err := updatePeersStorageVersion(versionFile, currVersion); err != nil {
		return errors.Wrapf(err,
			"failed to update peers storage file, old='%d', current='%d'",
			oldVersion,
			currVersion,
		)
	}
	return nil
}

func (bs *CBORStorage) unsafeSyncKnown(newEntries []KnownPeer, backup knownPeers) error {
	if err := marshalToCborAndSyncToFile(bs.knownFilePath, bs.known); err != nil {
		// In case of failure restore initial state from backup
		for _, k := range newEntries {
			delete(bs.known, k)
		}
		for k, v := range backup {
			bs.known[k] = v
		}
		return errors.Wrap(err, "failed to marshal known peers and sync storage")
	}
	return nil
}

func (bs *CBORStorage) unsafeDropKnown() error {
	// Truncate suspendedStorageFile to zero size
	if err := os.Truncate(bs.knownFilePath, 0); err != nil {
		return errors.Wrapf(err, "failed to drop known storage file %q", bs.knownFilePath)
	}
	// Clear map
	bs.known = knownPeers{}
	return nil
}

func (bs *CBORStorage) unsafeSyncSuspended(newEntries, backup []SuspendedPeer) error {
	if err := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended); err != nil {
		// Remove suspended from map to eliminate side effects
		for _, s := range newEntries {
			delete(bs.suspended, s.IP)
		}
		// Restore state before error from backup
		for _, s := range backup {
			bs.suspended[s.IP] = s
		}
		return errors.Wrap(err, "failed to marshal suspended peers and sync storage")
	}
	return nil
}

func (bs *CBORStorage) unsafeDropSuspended() error {
	// Truncate suspendedStorageFile to zero size
	if err := os.Truncate(bs.suspendedFilePath, 0); err != nil {
		return errors.Wrapf(err, "failed to drop suspended storage file %q", bs.suspendedFilePath)
	}
	// Clear map
	bs.suspended = suspendedPeers{}
	return nil
}

// unsafeKnownIntersection returns values from known map which intersects with input values
func (bs *CBORStorage) unsafeKnownIntersection(known []KnownPeer) knownPeers {
	intersection := knownPeers{}
	for _, k := range known {
		if v, in := bs.known[k]; in {
			intersection[k] = v
		}
	}
	return intersection
}

// unsafeSuspendedIntersection returns values from suspended map which intersects with input values
func (bs *CBORStorage) unsafeSuspendedIntersection(suspended []SuspendedPeer) []SuspendedPeer {
	var intersection []SuspendedPeer
	for _, newSuspended := range suspended {
		if storedPeer, in := bs.suspended[newSuspended.IP]; in {
			intersection = append(intersection, storedPeer)
		}
		bs.suspended[newSuspended.IP] = newSuspended
	}
	return intersection
}

func (bs *CBORStorage) unsafeIsSuspendedIP(ip IP, now time.Time) bool {
	s, in := bs.suspended[ip]
	if !in {
		return false
	}
	return s.IsSuspended(now)
}

func marshalToCborAndSyncToFile(filePath string, value interface{}) error {
	data, err := cbor.Marshal(value)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %T to CBOR", value)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return errors.Wrapf(err, "failed to write %T in file %q", value, filePath)
	}
	return nil
}

// unmarshalCborFromFile read file content and trying to unmarshall it into out parameter. It also
// returns error if file is empty.
func unmarshalCborFromFile(path string, out interface{}) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return errors.Wrapf(err, "failed to read from file with name %q", path)
	}

	switch err := cbor.Unmarshal(data, out); {
	case err == io.EOF:
		return io.EOF
	case err != nil:
		return errors.Wrapf(err, "failed to unmarshall CBOR into %T from file %q", out, path)
	}
	return nil
}

func knownFilePath(storageDir string) string {
	return filepath.Join(storageDir, "peers_known.cbor")
}

func suspendedFilePath(storageDir string) string {
	return filepath.Join(storageDir, "peers_suspended.cbor")
}

func storageVersionFilePath(storageDir string) string {
	return filepath.Join(storageDir, "peers_storage_version.txt")
}

func createFileIfNotExist(path string) (err error) {
	cleanedPath := filepath.Clean(path)
	knownFile, err := os.OpenFile(cleanedPath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to create if not exist file %q", path)
	}
	defer func() {
		if closeErr := knownFile.Close(); closeErr != nil {
			if err != nil {
				err = errors.Wrapf(err, "failed to close file %q, %v", path, closeErr)
			} else {
				err = errors.Wrapf(closeErr, "failed to close file %q", path)
			}
		}
	}()
	return nil
}

func updatePeersStorageVersion(storageVersionFile string, newVersion int) error {
	stringVersion := strconv.Itoa(newVersion)
	err := os.WriteFile(storageVersionFile, []byte(stringVersion), 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to write data in file %q", storageVersionFile)
	}
	return nil
}

func getPeersStorageVersion(storageVersionFile string) (int, error) {
	cleanedStorageVersionFile := filepath.Clean(storageVersionFile)
	if err := createFileIfNotExist(cleanedStorageVersionFile); err != nil {
		return 0, errors.Wrap(err, "failed to create if not exists storage version file")
	}
	versionData, err := os.ReadFile(cleanedStorageVersionFile)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read from file %q", storageVersionFile)
	}
	if len(versionData) == 0 {
		// it's a new peers storage
		return 0, io.EOF
	}
	oldVersion, err := strconv.Atoi(string(versionData))
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse peers storage version from file %q", storageVersionFile)
	}
	return oldVersion, nil
}
