package storage

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	peersStorageDir = "peers_storage"
)

type CBORStorage struct {
	rwMutex           sync.RWMutex
	storageDir        string
	suspended         suspendedPeers
	suspendedFilePath string
	known             knownPeers // nickeskov: list of all ever known peers with a publicly available declared address
	knownFilePath     string
}

func NewCBORStorage(baseDir string, now time.Time) (*CBORStorage, error) {
	storageDir := filepath.Join(baseDir, peersStorageDir)
	return newCBORStorageInDir(storageDir, now)
}

func newCBORStorageInDir(storageDir string, now time.Time) (*CBORStorage, error) {
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

	known := knownPeers{}
	if err := unmarshalCborFromFile(knownFile, &known); err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "failed to load known peers from file %q", knownFile)
	}

	suspended := suspendedPeers{}
	if err := unmarshalCborFromFile(suspendedFile, &suspended); err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "failed to load suspended peers from file %q", suspendedFile)
	}

	storage := &CBORStorage{
		storageDir:        storageDir,
		suspended:         suspended,
		suspendedFilePath: suspendedFile,
		known:             known,
		knownFilePath:     knownFile,
	}

	if len(storage.suspended) != 0 {
		// nickeskov: remove expired peers
		if err := storage.RefreshSuspended(now); err != nil {
			return nil, errors.Wrapf(err,
				"failed to refresh suspended peers while opening peers storage with path %q", storageDir)
		}
	}
	return storage, nil
}

func (bs *CBORStorage) Known() []KnownPeer {
	bs.rwMutex.RLock()
	defer bs.rwMutex.RUnlock()

	known := make([]KnownPeer, 0, len(bs.known))
	for k := range bs.known {
		known = append(known, k)
	}
	return known
}

// AddKnown adds known peers into peers storage with strong error guarantees.
func (bs *CBORStorage) AddKnown(known []KnownPeer) error {
	if len(known) == 0 {
		return nil
	}

	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	// nickeskov: save old values in backup
	backup := bs.unsafeKnownIntersection(known)
	// nickeskov: fast path if all known peers already in storage
	if len(backup) == len(known) {
		return nil
	}

	// nickeskov: add new values into known map
	for _, k := range known {
		bs.known[k] = struct{}{}
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

	// nickeskov: save old values in backup
	backup := bs.unsafeKnownIntersection(known)
	// nickeskov: delete entries from known map
	for _, k := range known {
		delete(bs.known, k)
	}

	// nickeskov: newEntries is nil because there no new entries
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

	// nickeskov: save old values in backup
	backup := bs.unsafeSuspendedIntersection(suspended)
	// nickeskov: add new values into suspended map
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

	// nickeskov: save old values in backup
	backup := bs.unsafeSuspendedIntersection(suspended)
	// nickeskov: delete entries from known map
	for _, s := range suspended {
		delete(bs.suspended, s.IP)
	}

	// nickeskov: newEntries is nil because there no new entries
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
		// nickeskov: peers don't expired
		return nil
	}

	if err := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended); err != nil {
		// nickeskov: restore previous values into map to eliminate side effects
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
// In case of error we can loose suspended peers storage file, but honestly it's almost impossible case.
func (bs *CBORStorage) DropStorage() error {
	bs.rwMutex.Lock()
	defer bs.rwMutex.Unlock()

	suspendedBackup := bs.suspended
	if err := bs.unsafeDropSuspended(); err != nil {
		return errors.Wrap(err, "failed to drop suspended peers storage")
	}

	if err := bs.unsafeDropKnown(); err != nil {
		bs.suspended = suspendedBackup
		// nickeskov: it's almost impossible case, but if it happens we have inconsistency in suspended peers
		// but honestly it's not fatal error
		if syncErr := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended); syncErr != nil {
			return errors.Wrapf(err, "failed to sync suspended peers storage from backup: %v", syncErr)
		}
		return errors.Wrap(err, "failed to drop known peers storage")
	}
	return nil
}

func (bs *CBORStorage) unsafeSyncKnown(newEntries, backup []KnownPeer) error {
	err := marshalToCborAndSyncToFile(bs.knownFilePath, bs.known)
	if err == nil {
		return nil
	}
	// nickeskov: remove known from map to eliminate side effects
	for _, k := range newEntries {
		delete(bs.known, k)
	}
	// nickeskov: restore from backup
	for _, b := range backup {
		bs.known[b] = struct{}{}
	}
	return errors.Wrap(err, "failed to marshal known peers and sync storage")
}

func (bs *CBORStorage) unsafeDropKnown() error {
	// nickeskov: truncate suspendedStorageFile to zero size
	if err := os.Truncate(bs.knownFilePath, 0); err != nil {
		return errors.Wrapf(err, "failed to drop known storage file %q", bs.knownFilePath)
	}
	// nickeskov: clear map
	bs.known = knownPeers{}
	return nil
}

func (bs *CBORStorage) unsafeSyncSuspended(newEntries, backup []SuspendedPeer) error {
	err := marshalToCborAndSyncToFile(bs.suspendedFilePath, bs.suspended)
	if err == nil {
		return nil
	}
	// nickeskov: remove suspended from map to eliminate side effects
	for _, s := range newEntries {
		delete(bs.suspended, s.IP)
	}
	// nickeskov: restore from backup
	for _, s := range backup {
		bs.suspended[s.IP] = s
	}
	return errors.Wrap(err, "failed to marshal suspended peers and sync storage")
}

func (bs *CBORStorage) unsafeDropSuspended() error {
	// nickeskov: truncate suspendedStorageFile to zero size
	if err := os.Truncate(bs.suspendedFilePath, 0); err != nil {
		return errors.Wrapf(err, "failed to drop suspended storage file %q", bs.suspendedFilePath)
	}
	// nickeskov: clear map
	bs.suspended = suspendedPeers{}
	return nil
}

// unsafeKnownIntersection returns values from known map which intersects with input values
func (bs *CBORStorage) unsafeKnownIntersection(known []KnownPeer) []KnownPeer {
	var intersection []KnownPeer
	for _, k := range known {
		if _, in := bs.known[k]; in {
			intersection = append(intersection, k)
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

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write %T in file %q", value, filePath)
	}
	return nil
}

// unmarshalCborFromFile read file content and trying unmarshall it into out parameter. It also
// returns error if file is empty.
func unmarshalCborFromFile(filePath string, out interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read from file with name %q", filePath)
	}

	switch err := cbor.Unmarshal(data, out); {
	case err == io.EOF:
		return io.EOF
	case err != nil:
		return errors.Wrapf(err, "failed to unmarshall CBOR into %T from file %q", out, filePath)
	}
	return nil
}

func knownFilePath(storageDir string) string {
	return filepath.Join(storageDir, "peers_known.cbor")
}

func suspendedFilePath(storageDir string) string {
	return filepath.Join(storageDir, "peers_suspended.cbor")
}

func createFileIfNotExist(path string) (err error) {
	knownFile, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to create if not exist file %q", path)
	}
	defer func() {
		if closeErr := knownFile.Close(); closeErr != nil {
			err = errors.Wrapf(err, "failed to close file %q", path)
		}
	}()
	return nil
}
