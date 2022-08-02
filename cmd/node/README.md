# node - Waves node implemented in Go

## Usage

```
usage: node [flags]
  -log-level          Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.
  -state-path         Path to node's state directory
  -blockchain-type    Blockchain type: mainnet/testnet/stagenet
  -peers              Addresses of peers to connect to
  -declared-address   Address to listen on
  -api-address        Address for REST API
  -grpc-address       Address for gRPC API
  -enable-grpc-api    Enables or disables gRPC API
  -build-extended-api Builds extended API. Note that state must be reimported in case it wasn't imported with similar flag set
  -serve-extended-api Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point
  -seed               Seed for miner
  -binds-address      Bind address for incoming connections. If empty, will be same as declared address
```
Parameter `-state-path` has no default value, so you have to provide the path to node state directory.

By default, most parameters have values for MainNet. 

To start a node on MainNet execute the following command.

```bash
./node -state-path [path to node state directory]
``` 

To start a TestNet node use the command below.

```bash
./node -state-path [path to node state directory] -peers 52.51.92.182:6863,52.231.205.53:6863,52.30.47.67:6863,52.28.66.217:6863 -blockchain-type testnet
``` 

## Running node on Linux

The easiest way to run node on Linux is to install it from DEB package. 
Download relevant DEB package from the [Releases](https://github.com/wavesplatform/gowaves/releases) page and install it with one of the following commands.

```bash
sudo dpkg -i gowaves-mainnet-v0.10.0.deb

sudo dpkg -i gowaves-testnet-v0.10.0.deb
 
sudo dpkg -i gowaves-stagenet-v0.10.0.deb
```

Corresponding `systemd` will be created. 
To start and stop, for example, the MainNet service use:

```bash
sudo systemctl start gowaves-mainnet.service

sudo systemctl stop gowaves-mainnet.service
```

To check the logs use `journalctl` utility.

```bash
sudo journalctl -u gowaves-mainnet.service -f
```

## Building DEB packages

To build DEB packages execute the command:

```bash
make release-node
```

The DEB files are placed in the `build/dist` folder.
For example, DEB package for MainNet will be named `gowaves-mainnet-0.10.0.deb`.
