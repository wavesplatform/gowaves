package state

import (
	"bytes"
	stderrs "errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

// Migrations of the state storage.
//
// Each change of the storage format MUST increment StateVersion, and either a migration from the previous version
// is added to the list returned by stateMigrations, or the state of previous versions has to be re-imported.
// The migration from version N converts the database to version N+1. Migrations are applied one by one, so the
// state of any version, starting from the oldest migration, can be brought to StateVersion.
//
// A migration consists of steps, each step converts all records with the given key prefix. Converted records are
// written in batches along with the progress of migration, so an interrupted migration continues from the last
// written record. The version of the database is updated along with the removal of the progress in the last batch
// of migration, so the node, which requires StateVersion, never opens a partially migrated database.

// migrationBatchSize is the number of converted records written to the database at once along with the progress.
const migrationBatchSize = 10_000

// stateMigration converts the state database from version `from` to version `from+1`.
type stateMigration struct {
	from  uint16
	steps func(db *keyvalue.KeyVal) []migrationStep
}

// stateMigrations returns all supported migrations ordered by the source version, without gaps,
// the last one converts the database to StateVersion.
func stateMigrations() []stateMigration {
	return []stateMigration{
		{from: stateVersionV26, steps: migrationStepsFromV26},
	}
}

// migrationStep converts all records with the given key prefix. The convert function returns data unchanged
// if the record is already of the new format, such records are not rewritten. The record is removed if the convert
// function returns nil data without an error.
type migrationStep struct {
	name    string
	prefix  byte
	convert func(key, data []byte) ([]byte, error)
}

// migrationProgress is stored in the database along with each batch of converted records, so the migration
// can be resumed from the last converted key after an interruption.
type migrationProgress struct {
	Version uint16 `cbor:"0,keyasint"` // source version of the migration in progress
	Step    int    `cbor:"1,keyasint"` // index of the step in progress
	LastKey []byte `cbor:"2,keyasint,omitempty"`
}

func (p *migrationProgress) marshalBinary() ([]byte, error) { return cbor.Marshal(p) }

func (p *migrationProgress) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, p) }

// MigrateState converts the state storage located in dataDir to StateVersion.
// The node MUST be stopped during the migration. The migration can be safely restarted after an interruption,
// it continues from the last converted record. Make a backup of the state before the migration.
func MigrateState(dataDir string, params keyvalue.KeyValParams) (err error) {
	metaInfo, err := readDBMeta(dataDir)
	if err != nil {
		return err
	}
	// The version in the meta file can be already current while the database is not migrated, for example,
	// if the meta file was recreated by the node. The version stored in the database is the source of truth.
	migrations := stateMigrations()
	if vErr := checkMigrationSupported(metaInfo.Version, migrations); vErr != nil {
		return vErr
	}
	dbDir := filepath.Join(dataDir, keyvalueDir)
	if _, sErr := os.Stat(dbDir); sErr != nil {
		return errors.Wrap(sErr, "failed to find state database")
	}
	params.CompressionAlgo = metaInfo.DBCompressionAlgo
	// The bloom filter is not used by migrations, the no-op store prevents overwriting of the stored filter on close.
	params.DisableBloomFilter = true
	params.BloomFilterStore = keyvalue.NoOpStore{}
	db, err := keyvalue.NewKeyVal(dbDir, params)
	if err != nil {
		return errors.Wrap(err, "failed to open state database")
	}
	defer func() {
		if clErr := db.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrap(clErr, "failed to close state database"))
		}
	}()
	if mErr := migrateDB(db, migrations); mErr != nil {
		return mErr
	}
	if metaInfo.Version == StateVersion {
		return nil
	}
	metaInfo.Version = StateVersion
	return writeDBMeta(dataDir, metaInfo)
}

func checkMigrationSupported(version uint16, migrations []stateMigration) error {
	if version == StateVersion {
		return nil
	}
	if version > StateVersion {
		return errors.Errorf("state storage version %d is newer than supported version %d", version, StateVersion)
	}
	if len(migrations) == 0 || version < migrations[0].from {
		return errors.Errorf("migration of state storage version %d is not supported, the state has to be re-imported",
			version)
	}
	return nil
}

func migrateDB(db *keyvalue.KeyVal, migrations []stateMigration) error {
	info, err := readStateInfoFromDB(db)
	if err != nil {
		return err
	}
	if vErr := checkMigrationSupported(info.Version, migrations); vErr != nil {
		return vErr
	}
	if info.Version == StateVersion {
		slog.Info("State database is already of the current version", "version", StateVersion)
		return nil
	}
	for _, m := range migrations {
		if m.from != info.Version {
			continue
		}
		if mErr := runMigration(db, m, info); mErr != nil {
			return errors.Wrapf(mErr, "failed to migrate state from version %d to %d", m.from, m.from+1)
		}
		info.Version = m.from + 1
	}
	if info.Version != StateVersion {
		return errors.Errorf("no migration from state version %d", info.Version)
	}
	slog.Info("State database migrated", "version", StateVersion)
	return nil
}

func runMigration(db *keyvalue.KeyVal, m stateMigration, info stateInfo) error {
	progress, err := readMigrationProgress(db, m.from)
	if err != nil {
		return err
	}
	slog.Info("Migrating state", "from", m.from, "to", m.from+1)
	steps := m.steps(db)
	for i := progress.Step; i < len(steps); i++ {
		var startAfter []byte
		if i == progress.Step {
			startAfter = progress.LastKey
		}
		slog.Info("Migrating", "step", steps[i].name, "resumeAfterKey", fmt.Sprintf("%x", startAfter))
		if cErr := convertRecordsByPrefix(db, m.from, i, steps[i], startAfter); cErr != nil {
			return errors.Wrapf(cErr, "failed to migrate %s", steps[i].name)
		}
	}
	// Update the version and remove the progress in one batch.
	info.Version = m.from + 1
	infoBytes, err := info.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal state info")
	}
	batch, err := db.NewBatch()
	if err != nil {
		return err
	}
	batch.Put(stateInfoKeyBytes, infoBytes)
	batch.Delete([]byte{migrationProgressKeyPrefix})
	if fErr := db.Flush(batch); fErr != nil {
		return errors.Wrap(fErr, "failed to update state info")
	}
	return nil
}

func convertRecordsByPrefix(
	db *keyvalue.KeyVal, version uint16, stepIndex int, step migrationStep, startAfter []byte,
) error {
	iter, err := db.NewKeyIterator([]byte{step.prefix}) // iterator works on the snapshot of the database
	if err != nil {
		return errors.Wrap(err, "failed to create iterator")
	}
	defer iter.Release()
	batch, err := db.NewBatch()
	if err != nil {
		return err
	}
	var converted, removed, kept, inBatch int
	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		if startAfter != nil && bytes.Compare(key, startAfter) <= 0 {
			continue // converted before the interruption
		}
		value, cErr := step.convert(key, iter.Value())
		if cErr != nil {
			return errors.Wrapf(cErr, "failed to convert record by key %x", key)
		}
		switch {
		case value == nil:
			batch.Delete(key)
			removed++
		case bytes.Equal(value, iter.Value()):
			kept++ // the record is already of the new format
			continue
		default:
			batch.Put(key, value)
			converted++
		}
		inBatch++
		if inBatch == migrationBatchSize {
			p := migrationProgress{Version: version, Step: stepIndex, LastKey: key}
			if fErr := flushWithProgress(db, batch, p); fErr != nil {
				return fErr
			}
			inBatch = 0
			slog.Info("Migration progress", "step", step.name, "converted", converted)
		}
	}
	if itErr := iter.Error(); itErr != nil {
		return errors.Wrap(itErr, "iterator error")
	}
	// Complete the step, the remaining records are written along with the progress pointing to the next step.
	if fErr := flushWithProgress(db, batch, migrationProgress{Version: version, Step: stepIndex + 1}); fErr != nil {
		return fErr
	}
	slog.Info("Migration step completed", "step", step.name, "converted", converted, "removed", removed,
		"kept", kept)
	return nil
}

func flushWithProgress(db *keyvalue.KeyVal, batch keyvalue.Batch, progress migrationProgress) error {
	data, err := progress.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal migration progress")
	}
	batch.Put([]byte{migrationProgressKeyPrefix}, data)
	if fErr := db.Flush(batch); fErr != nil {
		return errors.Wrap(fErr, "failed to write converted records")
	}
	return nil
}

// readMigrationProgress returns the progress of the migration from the given version. The progress of
// a migration from another version means the database is inconsistent, the error is returned in this case.
func readMigrationProgress(db *keyvalue.KeyVal, version uint16) (migrationProgress, error) {
	data, err := db.Get([]byte{migrationProgressKeyPrefix})
	if errors.Is(err, keyvalue.ErrNotFound) {
		return migrationProgress{Version: version}, nil
	}
	if err != nil {
		return migrationProgress{}, errors.Wrap(err, "failed to read migration progress")
	}
	var p migrationProgress
	if umErr := p.unmarshalBinary(data); umErr != nil {
		return migrationProgress{}, errors.Wrap(umErr, "failed to unmarshal migration progress")
	}
	if p.Version != version {
		return migrationProgress{}, errors.Errorf(
			"found progress of migration from version %d, but the database is of version %d", p.Version, version)
	}
	slog.Info("Resuming interrupted migration", "from", p.Version, "step", p.Step)
	return p, nil
}

func readStateInfoFromDB(db *keyvalue.KeyVal) (stateInfo, error) {
	data, err := db.Get(stateInfoKeyBytes)
	if err != nil {
		return stateInfo{}, errors.Wrap(err, "failed to read state info")
	}
	var info stateInfo
	if umErr := info.unmarshalBinary(data); umErr != nil {
		return stateInfo{}, errors.Wrap(umErr, "failed to unmarshal state info")
	}
	return info, nil
}
