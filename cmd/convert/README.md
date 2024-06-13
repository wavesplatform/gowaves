# Utility `convert`

The `convert` utility is designed for converting Waves transactions between JSON to Binary formats, and vice versa.
It can also be used to sign unsigned transactions while transforming their representation. 

## Command line options

```bash
  -scheme string
        Network scheme byte. Defaults to 'W' (MainNet).
  -to-json
        Convert the transaction to JSON representation. Sign the transaction if a private key is provided.
  -to-binary
        Convert the transaction to binary representation. Sign the transaction if a private key is provided.
  -base64
        Use Base64 as the binary transaction encoding.
  -private-key string
        Private key to sign the transaction. Please provide the key in Base58 string.
  -in string
        Input file path. Defaults to empty string. If empty, reads from STDIN.
  -out string
        Output file path. Defaults to empty string. If empty, writes to STDOUT.
```
## Conversion to the same format

By default, `convert` detects the format of input data and attempts to convert the transaction to the opposite format: from binary to JSON or from JSON to binary.
However, using the options `-to-json` and `-to-binary`, it is possible to override this rule and produce the resulting transaction in the same format as the source.
This is useful with the `-sign` option to produce a signed transaction from an unsigned one.

## Piping

The result of a transaction conversion can be piped to other utilities.

For example, the transaction converted from a file can be piped to `curl`.
```bash
./convert -private-key <private key base58> -to-json -in <transaction file> | curl -X POST -H 'accept: application/json' -H 'Content-Type: application/json' --data-binary @- 'https://nodes-testnet.wavesnodes.com/transactions/broadcast' 
```

The source transaction for conversion can be read from STDIN.
```bash
./convert -base64 < <Base64 transaction file>
```

Or both:
```bash
convert -private-key <private key Base58> -to-json < ~/Temp/convert/transfer-unsigned.json | curl -X POST -H 'accept: application/json' -H 'Content-Type: application/json' --data-binary @- 'https://nodes-testnet.wavesnodes.com/transactions/broadcast'
```
