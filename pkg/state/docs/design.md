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

##### Invoke access statistics records

Records with the `invokeStatsKeyPrefix` key prefix hold the access statistics of callable functions used by the
parallel invoke execution (see below). Unlike the records of blockchain entities, they are *not* history records:

* the key is `prefix (1 byte) | dApp address ID (20 bytes) | function name`;
* the value is written directly to the key-value database, not through `historyStorage`, so it has no
  entity type byte and no block numbers;
* the records are not consensus data: they are not included in state hashes, are not versioned by blocks
  and are not touched by rollback;
* the value starts with its own format version. A record that can not be decoded (unknown version, unknown
  flags, corrupt data) is treated as absent and is overwritten after the next execution of the function.

The value encoding (version 1) is:

`version (1 byte) | flags (1 byte) | executions (uvarint) | mispredictions (uvarint) | reads | writes`

where bit 0 of `flags` is the `wide` flag, `reads` and `writes` are `number of keys (uvarint) | key | key | ...`
and every key is `role << 6 | kind (1 byte) | payload | last seen execution (uvarint)`. The payload is present
only for absolute keys (role 0), its size depends on the kind: 20 bytes for account kinds, the size of an asset
ID for assets, 32 bytes for aliases and leases. Keys are sorted, so the encoding is deterministic.

The records were added without a change of `StateVersion`: existing records are not changed, and a state
without statistics is a valid state. A sample of the record is a part of the storage format golden file
(`testdata/storage_format/v28.json`).

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

#### Parallel invoke execution

By default, transactions of a block are applied strictly one by one. The expensive part of an `InvokeScript`
transaction is the evaluation of its RIDE script, and the result of the evaluation depends on the state left
by all the previous transactions of the block. If `ValidationParams.InvokeExecutionGoroutinesNum` is greater
than zero (the `-invoke-execution-goroutines-num` option of `node` and `importer`), scripts of invokes that
are expected to be independent are evaluated in parallel, optimistically, and the results are validated
before use. With zero (the default) none of the code described here is reachable.

The correctness never depends on predictions. It depends only on the comparison of what an evaluation
actually read with what the transactions applied after the start of the evaluation actually wrote.
Predictions only decide what is worth evaluating in advance. Block snapshots, state hashes and invoke results
are the same as in the sequential mode.

##### Access keys

A state entity touched by a transaction is identified by an `accessKey` (`invoke_access.go`): a kind and a
fixed-size payload. Kinds are finer than a whole account to avoid needless conflicts:

| Kind             | Payload        | Covers                                                             |
|------------------|----------------|--------------------------------------------------------------------|
| `AccountData`    | address ID     | all data entries of the account                                    |
| `AccountBalance` | address ID     | any balance of the account: Waves, assets, leasing, generating,    |
|                  |                | generation deposit                                                 |
| `AccountScript`  | address ID     | the script of the account and its complexities                     |
| `Asset`          | asset ID       | asset info, volume, description, script, sponsorship               |
| `Alias`          | alias          | the alias to address mapping                                       |
| `Lease`          | lease ID       | leasing info                                                       |
| `Tx`             | transaction ID | existence, height and status of the transaction                    |

A payload shorter than 32 bytes is zero-padded, a longer one is replaced by its hash. Both can only make two
entities share a key, which gives a spurious conflict and never a missed one.

An `accessSet` is a set of keys. There are two sources of actual sets:

* The **read set** of a script evaluation is recorded by `recordingSmartState` (`invoke_recording_state.go`),
  a wrapper of `types.EnrichedSmartState` that records the key before delegating every call, so failed and
  negative lookups are recorded too. Values that are constant within a block (height, block info, estimator
  version) are not recorded. For an alias recipient both the alias and the resolved address are recorded.
* The **write set** of an applied transaction is extracted from its `txSnapshot` by `writeSetFromSnapshot`.
  The extractor implements the snapshot applier interfaces, so a new kind of atomic snapshot does not compile
  until it is mapped to a key. Every applied transaction additionally writes its own `Tx` key.

##### Statistics

After every invoke applied by the block appender the keys it actually read and wrote are stored per
`(dApp address, function name)` in `invokeStatsStorage` (`invoke_stats.go`).

Statistics are role-normalised: a key of the account that sent the transaction is stored as the role `Sender`,
a key of the dApp as the role `DApp`, other keys are stored as they are. If the sender is the dApp, both roles
are stored. Before use the roles are replaced by the sender and the dApp of the transaction in question
(`instantiate`). `Tx` keys are not stored: they can conflict only inside a single block. The balance of the
block generator is removed from the stored writes: it is written by the fee of every transaction and would
make all invokes dependent (it still goes to the write log, see below).

Every key remembers the number of the last execution it was seen in. When the number of read keys or of write
keys exceeds `invokeStatsKeysSoftCap`, the least recently seen keys are evicted. If a single execution touches
more than `invokeStatsKeysHardCap` keys, the entry is marked `wide` for good: its keys are not tracked and
such invokes are never evaluated together with their predecessors. The entry also counts executions and
mispredictions.

Lifecycle: entries are loaded from the database lazily into an LRU cache of `invokeStatsCacheSize` entries,
updated in memory and written by `stateManager.flush` in the same database batch as the rest of the state.
If the application of blocks fails, the entries updated since the last flush are dropped from the cache. The
statistics are advisory and are not consensus data, see the description of the record in the storage section.
The storage is used by the appender goroutine only.

##### Dependency graph and batch selection

Before the next transaction of the block is applied, `invokeExecutor.beforeTx` (`invoke_execution.go`) checks
whether it is a speculatable invoke without a valid speculative result. If so, the accesses of the window of
up to `invokeSpeculationLookahead` upcoming transactions are predicted (`invoke_scheduler.go`):

* For an `InvokeScript` transaction: static keys known from the body (the balance of the sender, the data,
  the balance and the script of the dApp, payment assets, the alias of the recipient) united with the
  instantiated statistics of the function. Without statistics only the static keys are used, so the invoke
  is evaluated optimistically and the first execution creates the statistics. `wide` statistics make the
  transaction a barrier.
* For a simple transaction the writes are derived from the body: balances of the sender and of the
  recipients, the data or the script of the sender, the asset, the alias and so on.
* Any other transaction (Ethereum transactions and `InvokeExpression` included) is a barrier: its access is
  unknown.

In the dependency graph there is an edge from transaction `i` to a later transaction `j` if the predicted
writes of `i` intersect the predicted reads or writes of `j`, or if any of the two is a barrier.
`selectBatch` picks the speculatable invokes of the window that have no incoming edges from the transactions
that are not applied yet and that do not hold a valid speculative result. The next transaction to apply has
no predecessors in the window, so it is always selected. Nothing is selected after the first barrier. A batch
that would consist of the next invoke only is not launched: that invoke is evaluated in order.

The predictions of the transactions of the block are cached by the executor, so a window is not predicted from
scratch for every invoke. The cache is dropped when the predictions may change: when the keys or the `wide`
flag of the statistics of any function change (the storage of statistics keeps a revision for that), when
an alias is created and at the boundaries of a block.

##### Speculative evaluation

`invokeSpeculator` (`invoke_speculator.go`) owns a fixed pool of worker goroutines, started lazily and stopped
when the state is closed. `runBatch` evaluates the invokes of a batch in parallel, each over its own
`recordingSmartState`, and blocks the appender goroutine until all workers finish. A batch of a single invoke
is evaluated on the appender goroutine without starting the pool. The result of a worker is a
`speculativeResult`: the evaluation (RIDE result or error and the spent complexity), the actual read set,
which includes the reads made to resolve the dApp address and its script, and `baseIndex`, the number of
transactions of the block applied when the batch was launched.

A batch is launched only between transactions, when the state holds no changes of a partially applied
transaction, and the state is not modified while the batch runs: workers only read. If the state is not
between transactions when an invoke is about to be applied, which must not happen, no batch is launched and
all speculative results are dropped: the evaluation in order sees the partial changes, the results do not.
The read paths of the
state used by RIDE are safe for concurrent readers (for example, the cache of script trees is guarded by a
mutex). A panic of a worker is contained and means "no speculative result" for that transaction, it never
fails the block.

Only the evaluation of the script is done in advance. The verification of the sender, `checkTx`, the
accounting of complexity, the validation of script actions (`fallibleValidation`), the generation and the
application of the snapshot stay sequential on the appender goroutine.

##### Validation rule and re-execution

Transactions are applied in the order of the block. After every applied transaction (successful, failed,
elided or produced by a mainnet patch) `invokeExecutor.afterTx` puts its actual write set to the write log of
the block, which keeps the index of the last writer of every key.

A speculative result is valid if and only if no key of its read set was written by a transaction with an
index greater than or equal to its `baseIndex`. If the write set of an applied transaction can not be
computed, all speculative results are dropped.

When the script caller needs the evaluation of the invoke being applied, it asks `invokeExecutor.evaluation`:

* a valid speculative result is used instead of evaluating the script;
* an invalidated result is dropped and counted as a misprediction when its invoke becomes the next
  transaction to apply. The invoke is then evaluated once more: by the batch launched at that moment, if other
  invokes are selected along with it, or in order over a `recordingSmartState`, exactly as in the
  sequential mode;
* an invoke without a speculative result is evaluated in order the same way.

In all the cases the actual reads and writes of the invoke are known afterwards. If the script was called
and the transaction was not elided, the statistics of the function are updated with them, so the dependency
that caused a misprediction is predicted next time.
An invoke holding an invalidated result is not selected by the batches launched before it becomes the next
transaction: it depends on something the prediction does not see (the balance of the block generator, which
is written by every transaction but is kept out of statistics, is an example), so evaluating it again by every
batch would only waste the work. Every invoke is therefore evaluated at most twice speculatively and once in
order.

Speculative results and the write log belong to one block: they are dropped at the start and at the end of
`appendBlock` and by `txAppender.reset`, so a failed block leaves nothing behind. The numbers of speculated,
reused, mispredicted and inline evaluations are logged per block at the debug level.

##### Scope and limitations

* Only `InvokeScriptWithProofs` transactions in the block application path (`appendTxs`) are evaluated
  speculatively. Ethereum invokes and `InvokeExpression`, the validation of unconfirmed transactions,
  mining and the application of snapshots in the light node use the sequential code path.
* The state is read-only while a batch runs and the appender goroutine waits for it, so the gain is limited
  by the share of script evaluation in the block application time.
* Dependent invokes get no speedup. A block of fully conflicting invokes is applied slightly slower than in
  the sequential mode because of the prediction, the recording of reads and the extraction of write sets.
* Without statistics an invoke with a dependency that is not visible in its body may be evaluated
  speculatively more than once in a block.
* Statistics of a `wide` function are never narrowed again.

#### Transactions validation

TODO: complete.
