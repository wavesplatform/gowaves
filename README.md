# gowaves

Go implementation of Waves Node, libraries and tools for Waves blockchain.

![](https://github.com/wavesplatform/gowaves/workflows/build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/wavesplatform/gowaves)](https://goreportcard.com/report/github.com/wavesplatform/gowaves)
[![codecov](https://codecov.io/gh/wavesplatform/gowaves/branch/master/graph/badge.svg)](https://codecov.io/gh/wavesplatform/gowaves)
[![GoDoc](https://godoc.org/github.com/wavesplatform/gowaves?status.svg)](https://godoc.org/github.com/wavesplatform/gowaves)

## Waves Node

It is possible to run Waves Node on Linux, macOS or Windows. Please, download an appropriate binary file from [Releases page](https://github.com/wavesplatform/gowaves/releases).

You can either synchronize a node over network or import a downloaded blockchain file.

### How to import blockchain from file

Blockchain files are available on [MainNet](http://blockchain.wavesnodes.com), [TestNet](http://blockchain-testnet.wavesnodes.com) [StageNet](http://blockchain-stagenet.wavesnodes.com/) download pages.

Import could be done as follows:

1. Download a blockchain file
1. Download the `importer` utility from [Releases](https://github.com/wavesplatform/gowaves/releases)
1. Run the command, put the path to the blockchain file and node's state directory as parameters.
   The third parameter is the number of blocks to import, it should be less than a desired height by one.

```bash
./importer -blockchain-path [path to blockchain file] -data-path [path to node state directory] -blocks-number [height - 1]
```

Import may take a few hours, after which you can run the node as described in next section.

Please note that the Go Node has its own state storage structure that is incompatible with Scala Node.

### How to run the node

Run the node as follows:

1. Download a suitable node's binary file from [Releases](https://github.com/wavesplatform/gowaves/releases)
1. Run the command, if required, put the path to the node's state directory.

```bash
./node -state-path [path to node state directory]
```

By default, the node is run as a MainNet node. To run a TestNet node put `testnet`, as a blockchain type. You may also enter a list of comma separated peers' addresses (Optional):
```bash
./node -state-path [path to node state directory] -blockchain-type testnet
``` 

Read more about [running the node as Linux service](https://github.com/wavesplatform/gowaves/tree/master/cmd/node#readme).

### How to set block generation

Go Node has two parameters which allow the loading of private keys from a wallet file.

```
-wallet-path [path to wallet file]
-wallet-password [password string]
```

For example:

```
./node -state-path ~/gowaves-testnet/ -blockchain-type testnet -wallet-path ~/testnet.wallet -wallet-password 'some super secret password' 
```

Once the parameters were provided, the node would try loading and using private keys to generate blocks.

#### How to create a wallet file

To create a wallet file use the `wallet` utility. Please download a suitable version of the `wallet` utility from the [Releases](https://github.com/wavesplatform/gowaves/releases) page.
The following command will create and add a new seed to the wallet file:

```bash
./wallet -new
```

The utility would ask for a password to encrypt the new wallet file. If a wallet file does not exist, the file will be created.
By default, new wallet file has name `.waves` and stored in user's home directory. Different wallet's file location can be set using `-wallet` option.

Also, it's possible to import existing seed phrase. Please, use `-seed-phrase` option to do so.
```bash
./wallet -seed-phrase "words of seed phrase..."
```

If you have a Base58 encoded seed phrase from Scala node configuration file. There is an option `-seed-phrase-base58` to import it.
Also, this Base58 encoded seed phrase can be exported from Waves.Exchange wallet using `Settings | Security | Encoded Seed Phrase` menu option.
```bash
./wallet -seed-phrase-base58 <string of Base58 encoded seed phrase>
```

The last import option `-account-seed-base58` allows to import a Base58 encoded account seed. 
```bash
./wallet -account-seed-base58 <string of Base58 encoded account seed>
```

To list the seeds stored in the wallet, run the following command and provide a password.
```bash
./wallet -show
```


### Client library examples

Create sender's public key from BASE58 string:
```go
   sender, err := crypto.NewPublicKeyFromBase58("<your-public-key>")
   if err != nil {
	   panic(err)
   }
```
Create sender's private key from BASE58 string:
```go
    sk, err := crypto.NewSecretKeyFromBase58("<your-private-key>")
    if err != nil {
        panic(err)
    }
```

Create script's address:
```go
    a, err := proto.NewAddressFromString("<script's address")
    if err != nil {
        panic(err)
    }
```

Create Function Call that will be passed to the script:
```go
    fc := proto.FunctionCall{
        Name: "foo",
        Arguments: proto.Arguments{
            proto.IntegerArgument{
                Value: 12345,
            },
            proto.BooleanArgument{
                Value: true,
            },
        },
    }
```

New InvokeScript Transaction:
```go
    tx, err := proto.NewUnsignedInvokeScriptV1('T', sender, a, fc, proto.ScriptPayments{}, waves, 500000, uint64(ts))
    if err != nil {
        panic(err)
    }
```

Sign the transaction with the private key:
```go
    err = tx.Sign(sk)
```

Create new HTTP client to send the transaction to public TestNet nodes:
```go
    client, err := client.NewClient(client.Options{BaseUrl: "https://testnodes.wavesnodes.com", Client: &http.Client{}})
    if err != nil {
        panic(err)
    }
```

Send the transaction to the network:
```go
    _, err = client.Transactions.Broadcast(ctx, tx)
    if err != nil {
        panic(err)
    }
```

### What's done

* Full blockchain support of Waves Protocol version 1.4
* Full support of RIDE version 6
* Full support of gRPC API
* Full support of Metamask API
* Block generation
* Partial and very limited support of REST API
* Fast and optimized import of blockchain
* Fast and optimized RIDE evaluation

### Known issues

* Reduced REST API, only few methods are available

### Future plans

* Extensive integration testing
* Full support of REST API

### Building from sources

Go version 1.19 or later is required to build the `node`, `importer`, `wallet` and other tools.

To build a node, importer or other tools run a `make` command:

```bash
make release-importer
make release-node
...
```

## Other Tools

* [chaincmp](https://github.com/wavesplatform/gowaves/blob/master/cmd/chaincmp/README.md) - utility to compare blockchains on few nodes
* [wmd](https://github.com/wavesplatform/gowaves/blob/master/cmd/wmd/README.md) - service to provide a market data for Waves DEX transactions
