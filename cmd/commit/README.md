# Utility `commit`

The `commit` utility creates `CommitToGeneration` transaction with signed proof of possession (PoP) and
writes it's JSON to stdout.

## Command Line Options

```
  -height uint
        Height of generation period start
  -private-key string
        Waves private key in Base58 encoding
  -fee uint
        Transaction fee in wavelets (default: 0.1 Waves)
  -timestamp string
        Transaction timestamp. Accepts:
          (empty)       current time in UNIX milliseconds
          HH            today at the given hour, e.g. "14"
          HH:MM         today at the given hour and minute, e.g. "14:30"
          HH:MM:SS      today at the given time, e.g. "14:30:45"
          +<duration>   current time shifted forward, e.g. "+1h", "+30m"
          -<duration>   current time shifted backward, e.g. "-30m"
```

## Example

```bash
./commit -private-key <base58-private-key> -height 1000000
```

The transaction JSON is written to stdout and can be piped to `convert` utility for signing and
later to the node broadcast endpoint:

```bash
./commit -private-key <base58-private-key> -height <height> | \
  ./convert -private-key <base58-private-key> -to-json | \
  curl -X POST -H 'Content-Type: application/json' --data-binary @- \
  https://<node-host>/transactions/broadcast
```
