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

## Start `node` as systemd service

To turn `node` executable into a systemd service we have to create a unit service file at `/lib/systemd/system/waves.service`.
The content of the file that starts MainNet node is shown below.

```config
[Unit]
Description=Gowaves MainNet node
ConditionPathExists=/usr/share/waves
After=network.target
 
[Service]
Type=simple
User=waves
Group=waves
LimitNOFILE=1024

Restart=on-failure
RestartSec=60
startLimitIntervalSec=60

WorkingDirectory=/usr/share/waves
ExecStart=/usr/share/waves/node -state-path /var/lib/waves/ -api-address 0.0.0.0:8080

# make sure log directory exists and owned by syslog
PermissionsStartOnly=true
ExecStartPre=/bin/mkdir -p /var/log/waves
ExecStartPre=/bin/chown syslog:adm /var/log/waves
ExecStartPre=/bin/chmod 755 /var/log/waves
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=waves
 
[Install]
WantedBy=multi-user.target
```

Execute the following commands to create the user, and service file for MainNet.

```bash
sudo useradd waves -s /sbin/nologin -M
sudo mv waves.service /lib/systemd/system/
sudo chmod 755 /lib/systemd/system/waves.service
sudo mkdir /usr/share/waves/
sudo chown waves:waves /usr/share/waves
sudo mkdir /var/lib/waves
sudo chown waves:waves /var/lib/waves
sudo cp node /usr/share/waves/
```

To enable, start and stop the service use commands:

```bash
sudo systemctl enable waves.service
sudo systemctl start waves.service
sudo systemctl stop waves.service
```

To check the logs use `journalctl` utility.

```bash
sudo journalctl -u waves -f
```

To setup a TestNet node as a systemd service use the following file `waves-testnet.service` and execute the following commands.

```config
[Unit]
Description=Gowaves TestNet node
ConditionPathExists=/usr/share/waves-testnet
After=network.target
 
[Service]
Type=simple
User=waves-testnet
Group=waves-testnet
LimitNOFILE=1024

Restart=on-failure
RestartSec=60
startLimitIntervalSec=60

WorkingDirectory=/usr/share/waves-testnet
ExecStart=/usr/share/waves-testnet/node -state-path /var/lib/waves-testnet/ -api-address 0.0.0.0:8090 -peers 159.69.126.149:6863,94.130.105.239:6863,159.69.126.153:6863,94.130.172.201:6863 -blockchain-type testnet
# make sure log directory exists and owned by syslog
PermissionsStartOnly=true
ExecStartPre=/bin/mkdir -p /var/log/waves-testnet
ExecStartPre=/bin/chown syslog:adm /var/log/waves-testnet
ExecStartPre=/bin/chmod 755 /var/log/waves-testnet
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=waves-testnet
 
[Install]
WantedBy=multi-user.target
```

```bash
sudo useradd waves-testnet -s /sbin/nologin -M
sudo mv waves-testnet.service /lib/systemd/system/
sudo chmod 755 /lib/systemd/system/waves-testnet.service
sudo mkdir /usr/share/waves-testnet/
sudo chown waves-testnet:waves-testnet /usr/share/waves-testnet
sudo mkdir /var/lib/waves-testnet
sudo chown waves-testnet:waves-testnet /var/lib/waves-testnet
sudo cp node /usr/share/waves-testnet/
```

```bash
sudo systemctl enable waves-testnet.service
sudo systemctl start waves-testnet.service
sudo systemctl stop waves-testnet.service
```

```bash
sudo journalctl -u waves-testnet -f
```

## Building

To build `node` execute the command:

```bash
make release-node
```

The executable files are placed in `build/bin/[os-arch]` directories.
For example, Linux executable could be found at `build/bin/linux-amd64` directory.
