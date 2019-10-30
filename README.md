# gowaves
Go libraries and tools for Waves blockchain

[![Build Status](https://travis-ci.org/wavesplatform/gowaves.svg?branch=master)](https://travis-ci.org/wavesplatform/gowaves)
[![Go Report Card](https://goreportcard.com/badge/github.com/wavesplatform/gowaves)](https://goreportcard.com/report/github.com/wavesplatform/gowaves)
[![codecov](https://codecov.io/gh/wavesplatform/gowaves/branch/master/graph/badge.svg)](https://codecov.io/gh/wavesplatform/gowaves)
[![GoDoc](https://godoc.org/github.com/wavesplatform/gowaves?status.svg)](https://godoc.org/github.com/wavesplatform/gowaves)

## How to import blockchain from file

 * At first, build `importer` tool: `make release-importer`.
 * Run `importer -h` to read help info.
 * To import MainNet blockchain, you will usually need to execute:
`importer -blockchain-path </path/to/blocks/file> -data-path </path/to/node/state/dir> -blocks-number <desired_height-1>`.
 * To run the node after import, switch to `/path/to/node/state`, and start node as usually.

## Tools

* [chaincmp](https://github.com/wavesplatform/gowaves/blob/master/cmd/chaincmp/README.md) - utility to compare blockchains on few nodes
* [wmd](https://github.com/wavesplatform/gowaves/blob/master/cmd/wmd/README.md) - service to provide a market data for Waves DEX transactions
