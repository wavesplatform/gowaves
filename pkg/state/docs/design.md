## Brief overview of State architecture

#### Storage

Blocks have unique numbers (not heights!). When state accepts new block, it assigns
unique number for this block ID. Block nums start from zero and increase.
These numbers are used in storage instead of long block IDs.

Most of blockchain entities are stored using `historyStorage`.
For each address balance, asset info, lease, alias and many more,
the history in following format is stored in levelDB:

`[(data, blockNum), (data, blockNum), ...]`

When new entries are added to history, block nums of history's earliest entries are checked
for being too old (more than `rollbackMaxBlocks` in the past). In such case, `historyStorage` removes
them from the history using `historyFormatter`.

#### Rollback

There is list of valid block nums. Each entry in history (see above) is checked against this list.
Rolling back block is equal to simply removing its unique number from the list of valid block nums.

#### Transactions validation

TODO: complete.
