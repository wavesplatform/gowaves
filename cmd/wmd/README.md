# wmd - Waves Market Data

*Almost complete replacement for [WavesDataFeed](https://github.com/PyWaves/WavesDataFeed).*

Waves Market Data (wmd) is a service that offers the HTTP API similar to WavesDataFeed's API, but lacks the WebSocket API.
The state of `wmd` could be build using initial import of a [standard Waves blockchain file](http://blockchain.wavesnodes.com) 
or synchronizing with the mother-node's API (could take a long time).

## How it works

`wmd` starts the HTTP API and runs the synchronization with the Waves node. From that node it gets the information about new 
block, extracts transactions and builds historical market data in raw or candlestick formats.


## Distinctions from WavesDataFeed

* :heavy_minus_sign: No WebSocket API
* :heavy_minus_sign: No processing of UTX transactions
* :heavy_plus_sign: Import of binary blockchain file
* :fork_and_knife: Better forks resolution
* :rainbow: Support of mother-node's rollbacks
* :moneybag: Correct issuer's balances calculation

## Usage

```
usage: wmd [flags]
  -log-level        Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.
  -import-file      Path to binary blockchain file to import before starting synchronization.
  -node             Address of the node's gRPC API endpoint. Default value: 127.0.0.1:6870.
  -sync-interval    Synchronization interval, seconds. Default interval is 10 seconds.
  -lag              Synchronization lag behind the node, blocks. Default value 1 block.
  -address          Local network address to bind the HTTP API of the service on. Default value is :6990.
  -db               Path to data base folder. No default value.
  -matcher          Matcher's public key in form of Base58 string. Defaults to 7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy.
  -scheme           Blockchain scheme symbol. Defaults to 'W'.
  -symbols          Path to file of symbol substitutions. No default value.
  -rollback         The height to rollback to before importing a blockchain file or staring the synchronization. Default value is 0 (no rollback).

```

In simple case, then `wmd` runs on the same machine where the Waves node runs, it's should be provided with
parameters without default values only.

```bash
wmd -db /var/lib/wmd/db/ -symbols /var/lib/wmd/symbols.txt
``` 

Note that you have to create all the folders and give correct permissions on them. 
The example of `symbols.txt` file could be found at [Github](https://github.com/wavesplatform/gowaves/blob/master/cmd/wmd/symbols.txt).

To quickly build the initial state of `wmd`, please, download the actual [blockchain file](http://blockchain.wavesnodes.com) 
and execute the following command.

```bash
wmd -db /var/lib/wmd/db/ -symbols /var/lib/wmd/symbols.txt -import-file /home/user/Downloads/mainnet-1385453.dms
```

## WMD as systemd service

To turn `wmd` executable into a systemd service we have to create a unit service file at `/lib/systemd/system/wmd.service`. The content of the file is shown below.

```config
[Unit]
Description=WMD
ConditionPathExists=/usr/share/wmd
After=network.target
 
[Service]
Type=simple
User=wmd
Group=wmd
LimitNOFILE=1024

Restart=on-failure
RestartSec=10
startLimitIntervalSec=60

WorkingDirectory=/usr/share/wmd
ExecStart=/usr/share/wmd/wmd -db /var/lib/wmd/ -address 0.0.0.0:6990 -node grpc.wavesnodes.com:6870 -symbols /usr/share/wmd/symbols.txt -sync-interval 10
# make sure log directory exists and owned by syslog
PermissionsStartOnly=true
ExecStartPre=/bin/mkdir -p /var/log/wmd
ExecStartPre=/bin/chown syslog:adm /var/log/wmd
ExecStartPre=/bin/chmod 755 /var/log/wmd
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=wmd
 
[Install]
WantedBy=multi-user.target
```

Execute the following commands to create the user, and service file.

```bash
sudo useradd wmd -s /sbin/nologin -M
sudo mv wmd.service /lib/systemd/system/
sudo chmod 755 /lib/systemd/system/wmd.service
sudo mkdir /usr/share/wmd/
sudo chown wmd:wmd /usr/share/wmd
sudo mkdir /var/lib/wmd
sudo chown wmd:wmd /var/lib/wmd
sudo cp wmd /usr/share/wmd/
sudo cp symbols.txt /usr/share/wmd/
```

To enable, start and stop the service use commands:

```bash
sudo systemctl enable wmd.service
sudo systemctl start wmd.service
sudo systemctl stop wmd.service
```

To check the logs use `journalctl` utility.

```bash
sudo journalctl -u wmd -f
```

## Building

To build WMD execute the command:

```bash
make release-wmd
```

## HTTP API

### **GET** - /api/status

Returns the current status of the WMD. Status contains current height of WMD's state and the ID of the last block. 

#### CURL

```sh
curl -X GET "http://localhost:6990/api/status" \
    -H "Accept-Encoding: gzip, deflate"
```

### **GET** - /api/symbols

Returns the list of asset symbols. 

#### CURL

```sh
curl -X GET "http://localhost:6990/api/symbols" \
    -H "Accept-Encoding: gzip, deflate"
```

### **GET** - /api/markets

Get the list of all markets with 24h stats.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/markets" \
    -H "Accept-Encoding: gzip, deflate"
```

### **GET** - /api/tickers

Get tickers for all markets.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/tickers" \
    -H "Accept-Encoding: gzip, deflate"
```

### **GET** - /api/ticker/{AMOUNT_ASSET}/{PRICE_ASSET}

Get ticker for a specified asset pair.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/ticker/WAVES/BTC"
```

### **GET** - /api/trades/{AMOUNT_ASSET}/{PRICE_ASSET}/{LIMIT}

Get last `LIMIT` confirmed trades for a specified asset pair.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/trades/WAVES/BTC/10"
```

### **GET** - /api/trades/{AMOUNT_ASSET}/{PRICE_ASSET}/{FROM_TIMESTAMP}/{TO_TIMESTAMP}

Get trades within `FROM_TIMESTAMP` - `TO_TIMESTAMP` time range.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/trades/WAVES/BTC/1495296000000/1495296280000"
```

### **GET** - /api/trades/{AMOUNT_ASSET}/{PRICE_ASSET}/{ADDRESS}/{LIMIT}

Get trades for a specified asset pair and address.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/trades/WAVES/BTC/3PCfUovRHpCoGL54UakGBTSDEXTbmYMU3ib/10"
```

### **GET** - /api/candles/{AMOUNT_ASSET}/{PRICE_ASSET}/{TIMEFRAME}/{LIMIT}

Get last `LIMIT` candlesticks for the specified asset pair.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/candles/WAVES/BTC/5/10"
```

### **GET** - /api/candles/{AMOUNT_ASSET}/{PRICE_ASSET}/{TIMEFRAME}/FROM_TIMESTAMP/TO_TIMESTAMP

Get candlesticks within `FROM_TIMESTAMP` - `TO_TIMESTAMP` time range for the specified asset pair.

#### CURL

```sh
curl -X GET "http://localhost:6990/api/candles/WAVES/BTC/5/1495296000000/1495296280000"
```
