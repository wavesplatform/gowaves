# Utility `migrate`

The `migrate` utility converts the state storage of the node from one of the previous versions to the current
version of the storage format. Supported migrations:

| From version | To version | Changes                                                                         |
|--------------|------------|---------------------------------------------------------------------------------|
| 26           | 27         | Formats of Waves and asset balances, order of fields in legacy state hashes.    |

Migrations are applied one by one, so the state of any supported version is brought to the current one in a single
run. The state of a version older than the oldest supported migration has to be re-imported.

## Command line options

```bash
  -state-path string
        Path to node's state directory.
  -log-level string
        Set the logging level. Supported values: debug, info, warn, error. Default: info.
  -log-type string
        Set the logger output format. Supported types: text, json, pretty. Default: pretty.
```

## Usage

1. Stop the node. The node MUST NOT be running during the migration.
2. Make a backup of the state directory.
3. Run the migration with the same state directory that the node uses:
   ```bash
   migrate -state-path /path/to/state
   ```
4. Restore the owner of the state files, if the migration was run by another user (for example, `root`):
   ```bash
   chown -R gowaves:gowaves /path/to/state
   ```
   Otherwise, the node is unable to read the bloom filter and starts to rebuild it, which takes a long time.
5. Start the node.

The migration rewrites a large number of records and may take a while. It can be safely interrupted and restarted,
it continues from the last written batch of records. The node refuses to start on a partially migrated state.
