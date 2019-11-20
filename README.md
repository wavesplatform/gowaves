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

Blockchain files are available at [MainNet](http://blockchain.wavesnodes.com) and [TestNet](http://blockchain.testnet.wavesnodes.com) download pages.

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
./node -state-path [path to node state directory] -peers 52.51.92.182:6863,52.231.205.53:6863,52.30.47.67:6863,52.28.66.217:6863 -blockchain-type testnet
``` 
 
### What's done

 * Full blockchain support of Waves version 1.1
 * Full support of RIDE version 3
 * Fast optimized import of blockchain
 
### Known issues

 * Unstable network synchronization, first thing to improve
 * Uneven script estimation, overestimated scripts leads to warning
 * Reduced REST API, only few methods are available
 * No block generation (mining) for now, it's implemented but intentionally switched off

### Future plans

 * Complete gRPC API and extensive integration testing
 * RIDE optimization
 * Support of RIDE v4 and Waves v1.2 new features
 * Built-in wallet for full block generation (mining) support
 
### Building from sources

Go version 1.12 or later is required to build the `node`, `importer` and other tools. 

Build as usual or execute the appropriate `make` task:

```bash
make release-importer
make release-node
```

## Other Tools

* [chaincmp](https://github.com/wavesplatform/gowaves/blob/master/cmd/chaincmp/README.md) - utility to compare blockchains on few nodes
* [wmd](https://github.com/wavesplatform/gowaves/blob/master/cmd/wmd/README.md) - service to provide a market data for Waves DEX transactions
