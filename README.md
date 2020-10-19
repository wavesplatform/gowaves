# gowaves

Go implementation of Waves Node, libraries and tools for Waves blockchain.

![](https://github.com/wavesplatform/gowaves/workflows/build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/wavesplatform/gowaves)](https://goreportcard.com/report/github.com/wavesplatform/gowaves)
[![codecov](https://codecov.io/gh/wavesplatform/gowaves/branch/master/graph/badge.svg)](https://codecov.io/gh/wavesplatform/gowaves)
[![GoDoc](https://godoc.org/github.com/wavesplatform/gowaves?status.svg)](https://godoc.org/github.com/wavesplatform/gowaves)

## Waves Node

It is possible to run Waves Node on Linux, macOS or Windows. Please, download the appropriate binary file from [Releases page](https://github.com/wavesplatform/gowaves/releases).  

You can either synchronize a node over network or import a downloaded blockchain file.

### How to import blockchain from file

Blockchain files are available at [MainNet](http://blockchain.wavesnodes.com) and [TestNet](http://blockchain-testnet.wavesnodes.com) download pages.

Import could be done as follows:

 1. Download a blockchain file
 1. Download the `importer` utility from [Releases](https://github.com/wavesplatform/gowaves/releases)
 1. Run the command passing the paths to blockchain file and node's state directory as parameters. 
 The third parameter is the number of blocks to import, it should be less than desired height by one.  
 
```bash
./importer -blockchain-path [path to blockchain file] -data-path [path to node state directory] -blocks-number [height - 1]
```

Import could take up a few hours, afterward run the node as described in next section. 

Note that the Go node has its own state storage structure that is incompatible with Scala Node.

### How to run node

Run the node as follows:

 1. Download the suitable node's binary from [Releases](https://github.com/wavesplatform/gowaves/releases) 
 1. Run the command, if required, pass the path to the node's state directory.
 
```bash
./node -state-path [path to node state directory]
```

By default the node starts as MainNet node. To start TestNet node pass the `testnet` as blockchain type and comma separated list of TestNet peer's addresses:
```bash
./node -state-path [path to node state directory] -blockchain-type testnet
``` 

### How to setup block generation

Node has two parameters that allows to setup loading of private keys from a wallet file. 

```
-wallet-path [path to wallet file]
-wallet-password [password string]
```

For example:

```
./node -state-path ~/gowaves-testnet/ -blockchain-type testnet -wallet-path ~/testnet.wallet -wallet-password 'some super secret password' 
```

Once provided with such parameters node tries to load and use private keys for block generation.

#### How to create wallet file

To create a wallet file use `wallet` utility. Please download an appropriate version of `wallet` utility from the [Releases](https://github.com/wavesplatform/gowaves/releases) page.
The following command will add a seed to wallet file:

```
./wallet add -w [path to wallet file]
```

Utility asks for a seed phrase and a password to encrypt the new wallet file. If wallet file doesn't exists it will be created.

It is possible to provide not a seed phrase but a Base58 encoded seed in format compatible with waves.exchange application. To do so add `-b` flag:

```
./wallet add -w [path to wallet file] -b
```

And enter the string of Base58 encoded seed then asked.

To list the seed execute the following command and provide the password then asked. 

```
./wallet show -w [path to wallet file]
```
 
### What's done

 * Full blockchain support of Waves version 1.2
 * Full support of RIDE version 4
 * Full support of gRPC API
 * Block generation
 * Partial and very limited support of REST API
 * Fast optimized import of blockchain
 * Fast optimizes RIDE evaluation
 
### Known issues

 * Reduced REST API, only few methods are available
 
### Future plans

 * Full support of REST API
 * Extensive integration testing
 * RIDE v5, RIDE cross-DApp invocations and continuations
 
### Building from sources

Go version 1.12 or later is required to build the `node`, `importer`, `wallet` and other tools. 

Build as usual or execute the appropriate `make` task:

```bash
make release-importer
make release-node
...
```

## Other Tools

* [chaincmp](https://github.com/wavesplatform/gowaves/blob/master/cmd/chaincmp/README.md) - utility to compare blockchains on few nodes
* [wmd](https://github.com/wavesplatform/gowaves/blob/master/cmd/wmd/README.md) - service to provide a market data for Waves DEX transactions
