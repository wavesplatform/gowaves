## Brief overview of State architecture

#### Storage

Blocks have unique numbers (not heights!). When state accepts a new block, it assigns
unique number for this block ID. Block nums start from zero and increase.
These numbers are used in storage instead of long block IDs.

Most of blockchain entities are stored using `historyStorage`.
For each address balance, asset info, lease, alias and many more,
the history in following format is stored in levelDB:

`[(data, blockNum), (data, blockNum), ...]`

When new entries are added to history, block nums of history's earliest entries are checked
for being too old (more than `rollbackMaxBlocks` in the past). In such case, `historyStorage` removes
them from the history using `historyFormatter`.

##### History record encoding (state version 28)

A history record is stored under the key of the entity, its value is:

`entityType (1 byte) | entry | entry | ...`

Each entry is `data | blockNum (4 bytes, big-endian)`. The type of the entity defines whether its
entries have a fixed size:

* Fixed-size entities (aliases, order volumes and so on) store entries one after another without sizes.
  The record size is known from the entity properties.
* Variable-size entities (balances, data entries, asset info and so on) prefix each entry with its size,
  which is the length of `data` plus 4 bytes of `blockNum`. Since state version 28 the size is a canonical
  uvarint (1 or 2 bytes for most entries). Up to state version 27 it was a fixed 4-byte big-endian
  `uint32`. The decoder is strict: sizes must be canonical and the last entry must end exactly at the end
  of the record.

Ownership of the decoded data: the decoded entries point into the buffer they were decoded from, they are not
copied. That buffer must not be modified or reused while the record is in use. Values returned by
`keyvalue.KeyVal.Get` are owned by the caller and can be decoded directly. Values of iterators are reused
by the iterator and must be copied before decoding, or the needed data must be copied after decoding
(as `topEntryIterator` does with the top entry). The `keyvalue` batch copies keys and values on `Put`, so
the caller can reuse its buffers right after `Put` (`historyStorage.flush` encodes all records into one
reused buffer).

`historyFormatter` filters out the entries of rolled back blocks and cuts the entries older than
`rollbackMaxBlocks`. The minimal acceptable block number and the watermark of valid block numbers (all
block numbers starting from it are known to be valid) are cached in `stateDB`. The minimal acceptable block
number is cached until the next batch write or reset. The watermark changes only when blocks are rolled back:
it is invalidated when a block is removed and again when the batch with the removal is written.

##### Storage version and migrations

`StateVersion` in `constants.go` is the version of the storage format, it's stored in the database. The
node refuses to open a state of another version (`ErrIncompatibleStateParams`). Each change of the storage
format increments `StateVersion` and adds a migration from the previous version in `migration.go`.

A state of an older supported version (currently 26 and 27) must be converted with the `migrate` utility
before starting the node of the new version:

```sh
go build -o migrate ./cmd/migrate   # or make build-migrate-native
./migrate -state-path <path to the node's state directory>
```

Stop the node and back up the state directory first. An interrupted migration can be restarted, it
continues from the last converted record. The migration from version 27 to 28 (`migration_v27.go`)
re-encodes the histories of all variable-size entities with the uvarint entry sizes. It reads and rewrites
every such history, so it takes time proportional to the size of the state. Fixed-size histories and other
records are not touched.

#### Rollback

There is list of valid block nums. Each entry in history (see above) is checked against this list.
Rolling back block is equal to simply removing its unique number from the list of valid block nums.

#### Transactions validation

TODO: complete.
