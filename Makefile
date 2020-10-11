PROJECT=gowaves
ORGANISATION=wavesplatform
SOURCE=$(shell find . -name '*.go' | grep -v vendor/)
SOURCE_DIRS = cmd pkg

VERSION=$(shell git describe --tags --always --dirty)
DEB_VER=$(shell git describe --tags --abbrev=0 | cut -c 2-)
DEB_HASH=$(shell git rev-parse HEAD)

export GO111MODULE=on

.PHONY: vendor vetcheck fmtcheck clean build gotest

all: vendor vetcheck fmtcheck build gotest mod-clean

ver:
	@echo Building version: $(VERSION)

build: build/bin/forkdetector

build/bin/forkdetector: $(SOURCE)
	@mkdir -p build/bin
	go build -o build/bin/forkdetector ./cmd/forkdetector

build-forkdetector-linux-amd64:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/forkdetector -ldflags="-X main.version=$(VERSION)" ./cmd/forkdetector

build-forkdetector-linux-arm:
	@CGO_ENABLE=0 GOOS=linux GOARCH=arm go build -o build/bin/linux-arm/forkdetector -ldflags="-X main.version=$(VERSION)" ./cmd/forkdetector

gotest:
	go test -cover ./...

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

mod-clean:
	go mod tidy

clean:
	@rm -rf build
	go mod tidy

vendor:
	go mod vendor

vetcheck:
	go vet ./...
	golangci-lint run

build-chaincmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/chaincmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp

release-chaincmp: ver build-chaincmp-linux build-chaincmp-darwin build-chaincmp-windows

dist-chaincmp: release-chaincmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/chaincmp_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/chaincmp*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_Linux-64bit.tar.gz ./chaincmp*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_macOS-64bit.tar.gz ./chaincmp*

build-wmd-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/wmd -ldflags="-X main.version=$(VERSION)" ./cmd/wmd
build-wmd-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/wmd -ldflags="-X main.version=$(VERSION)" ./cmd/wmd
build-wmd-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/wmd.exe -ldflags="-X main.version=$(VERSION)" ./cmd/wmd

release-wmd: ver build-wmd-linux build-wmd-darwin build-wmd-windows

dist-wmd: release-wmd
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/wmd_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/wmd*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/wmd_$(VERSION)_Linux-64bit.tar.gz ./wmd*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/wmd_$(VERSION)_macOS-64bit.tar.gz ./wmd*

build-retransmitter-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/retransmitter -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter
build-retransmitter-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/retransmitter -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter
build-retransmitter-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/retransmitter.exe -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter

release-retransmitter: ver build-retransmitter-linux build-retransmitter-darwin build-retransmitter-windows

build-node-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/node ./cmd/node
build-node-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/node ./cmd/node
build-node-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/node.exe ./cmd/node

release-node: ver build-node-linux build-node-darwin build-node-windows

dist-node: release-node
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/node_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/node*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/node_$(VERSION)_Linux-64bit.tar.gz ./node*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/node_$(VERSION)_macOS-64bit.tar.gz ./node*

build-custom-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/custom ./cmd/custom
build-custom-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/custom ./cmd/custom
build-custom-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/custom.exe ./cmd/custom

build-custom: ver build-custom-linux build-custom-darwin build-custom-windows

build-docker:
	docker build -t com.wavesplatform/node-it:latest .
	date "+%Y-%m-%d %H:%M:%S"

build-importer-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/importer ./cmd/importer
build-importer-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/importer ./cmd/importer
build-importer-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/importer.exe ./cmd/importer

release-importer: ver build-importer-linux build-importer-darwin build-importer-windows

dist-importer: release-importer
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/importer_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/importer*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_Linux-64bit.tar.gz ./importer*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_macOS-64bit.tar.gz ./importer*

build-wallet-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/wallet ./cmd/wallet
build-wallet-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/wallet ./cmd/wallet
build-wallet-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/wallet.exe ./cmd/wallet

release-wallet: ver build-wallet-linux build-wallet-darwin build-wallet-windows

build-rollback-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/rollback ./cmd/rollback
build-rollback-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/rollback ./cmd/rollback
build-rollback-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/rollback.exe ./cmd/rollback

release-rollback: ver build-rollback-linux build-rollback-darwin build-rollback-windows

dist-wallet: release-wallet
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/wallet_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/wallet*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_Linux-64bit.tar.gz ./wallet*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_macOS-64bit.tar.gz ./wallet*

dist: clean dist-chaincmp dist-wmd dist-importer dist-node dist-wallet


build-genconfig:
	go build -o build/bin/darwin-amd64/genconfig ./cmd/genconfig

mock:
	mockgen -source pkg/miner/utxpool/cleaner.go -destination pkg/miner/utxpool/mock.go -package utxpool stateWrapper
	mockgen -source pkg/node/peer_manager/peer_manager.go -destination pkg/mock/peer_manager.go -package mock PeerManager
	mockgen -source pkg/p2p/peer/peer.go -destination pkg/mock/peer.go -package mock Peer
	mockgen -source pkg/state/api.go -destination pkg/mock/state.go -package mock State
	mockgen -source pkg/node/state_fsm/default.go -destination pkg/node/state_fsm/default_mock.go -package state_fsm Default
	mockgen -source pkg/grpc/server/api.go -destination pkg/mock/grpc.go -package mock GrpcHandlers

proto:
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=plugins=grpc:$(GOPATH)/src pkg/grpc/protobuf-schemas/proto/waves/*.proto
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=plugins=grpc:$(GOPATH)/src pkg/grpc/protobuf-schemas/proto/waves/node/grpc/*.proto

build-integration-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/integration ./cmd/integration

build-wmd-deb-package-linux:
	@mkdir -p ./build/wmd/DEBIAN
	@touch ./build/wmd/DEBIAN/control ./build/wmd/DEBIAN/preinst ./build/wmd/DEBIAN/postinst
	@cp ./cmd/wmd/symbols.txt ./build/wmd
	@cp ./build/bin/linux-amd64/wmd ./build/wmd
	@chmod 0775 ./build/wmd/DEBIAN/control
	@chmod 0775 ./build/wmd/DEBIAN/preinst
	@chmod 0775 ./build/wmd/DEBIAN/postinst
	@echo "Package: wmd\n\
Version: ${DEB_VER}\n\
Section: misc\n\
Priority: extra\n\
Architecture: all\n\
Depends: bash\n\
Essential: no\n\
Maintainer: ${ORGANISATION}.com\n\
Description: It is waves market data as systemd service. Hash: ${DEB_HASH}" >> ./build/wmd/DEBIAN/control
	@echo "#!/bin/bash\n\
# Creating directordy for logs and giving its rights\n\
sudo mkdir /var/log/wmd\n\
sudo chown syslog wmd\n\
# Creating wmd.service file and put config to\n\
touch wmd.service\n\
echo \"[Unit]\n\
Description=WMD\n\
ConditionPathExists=/usr/share/wmd\n\
After=network.target\n\
[Service]\n\
Type=simple\n\
User=wmd\n\
Group=wmd\n\
LimitNOFILE=1024\n\
Restart=on-failure\n\
RestartSec=10\n\
startLimitIntervalSec=60\n\
WorkingDirectory=/usr/share/wmd\n\
ExecStart=/usr/share/wmd/wmd -db /var/lib/wmd/ -address 0.0.0.0:6990 -node grpc.wavesnodes.com:6870 -symbols /usr/share/wmd/symbols.txt -sync-interval 1\n\
# make sure log directory exists and owned by syslog\n\
PermissionsStartOnly=true\n\
ExecStartPre=/bin/mkdir -p /var/log/wmd\n\
ExecStartPre=/bin/chown syslog:adm /var/log/wmd\n\
ExecStartPre=/bin/chmod 755 /var/log/wmd\n\
StandardOutput=syslog\n\
StandardError=syslog\n\
SyslogIdentifier=wmd\n\
[Install]\n\
WantedBy=multi-user.target\" >> wmd.service\n\
sudo useradd wmd -s /sbin/nologin -M\n\
sudo mv wmd.service /lib/systemd/system/\n\
sudo chmod 755 /lib/systemd/system/wmd.service\n\
sudo mkdir /usr/share/wmd/\n\
sudo chown wmd:wmd /usr/share/wmd\n\
sudo mkdir /var/lib/wmd\n\
sudo chown wmd:wmd /var/lib/wmd\n\
sudo cp wmd /usr/share/wmd/\n\
sudo cp symbols.txt /usr/share/wmd/\n\
sudo systemctl enable wmd.service\n\
sudo systemctl start wmd.service" >> ./build/wmd/DEBIAN/postinst
	@echo "#!/bin/bash\n\
sudo rm -f symbols.txt\n\
sudo rm -f wmd" >> ./build/wmd/DEBIAN/preinst
	@dpkg-deb --build ./build/wmd
	@mv ./build/wmd.deb wmd_${VERSION}.deb
	@mv wmd_${VERSION}.deb ./build/dist
	@rm -rf ./build/wmd

build-gowaves-node-deb-package-linux:
	@mkdir -p ./build/gowaves/DEBIAN
	@touch ./build/gowaves/DEBIAN/control ./build/gowaves/DEBIAN/preinst ./build/gowaves/DEBIAN/postinst
	@cp ./build/bin/linux-amd64/node ./build/gowaves/gowaves
	@chmod 0775 ./build/gowaves/DEBIAN/control
	@chmod 0775 ./build/gowaves/DEBIAN/preinst
	@chmod 0775 ./build/gowaves/DEBIAN/postinst
	@echo "Package: gowaves\n\
Version: ${DEB_VER}\n\
Section: misc\n\
Priority: extra\n\
Architecture: all\n\
Depends: bash\n\
Essential: no\n\
Maintainer: ${ORGANISATION}.com\n\
Description: It is gowaves node as systemd service. Hash: ${DEB_HASH}" >> ./build/gowaves/DEBIAN/control
	@echo "#!/bin/bash\n\
# Creating directordy for logs and giving its rights\n\
sudo mkdir /var/log/gowaves\n\
sudo chown syslog gowaves\n\
# Creating gowaves.service file and put config to\n\
touch gowaves.service\n\
echo \"[Unit]\n\
Description=Gowaves MainNet node\n\
ConditionPathExists=/usr/share/gowaves\n\
After=network.target\n\
[Service]\n\
Type=simple\n\
User=gowaves\n\
Group=gowaves\n\
LimitNOFILE=1024\n\
Restart=on-failure\n\
RestartSec=60\n\
startLimitIntervalSec=60\n\
WorkingDirectory=/usr/share/gowaves\n\
ExecStart=/usr/share/gowaves/gowaves -state-path /var/lib/gowaves/ -api-address 0.0.0.0:8080\n\
# make sure log directory exists and owned by syslog\n\
PermissionsStartOnly=true\n\
ExecStartPre=/bin/mkdir -p /var/log/gowaves\n\
ExecStartPre=/bin/chown syslog:adm /var/log/gowaves\n\
ExecStartPre=/bin/chmod 755 /var/log/gowaves\n\
StandardOutput=syslog\n\
StandardError=syslog\n\
SyslogIdentifier=gowaves\n\
[Install]\n\
WantedBy=multi-user.target\" >> gowaves.service\n\
sudo useradd gowaves -s /sbin/nologin -M\n\
sudo mv gowaves.service /lib/systemd/system/\n\
sudo chmod 755 /lib/systemd/system/gowaves.service\n\
sudo mkdir /usr/share/gowaves/\n\
sudo chown gowaves:gowaves /usr/share/gowaves\n\
sudo mkdir /var/lib/gowaves\n\
sudo chown gowaves:gowaves /var/lib/gowaves\n\
sudo cp gowaves /usr/share/gowaves/\n\
sudo systemctl enable gowaves.service\n\
sudo systemctl start gowaves.service" >> ./build/gowaves/DEBIAN/postinst
	@echo "#!/bin/bash\n\
sudo rm -f gowaves" >> ./build/gowaves/DEBIAN/preinst
	@dpkg-deb --build ./build/gowaves
	@mv ./build/gowaves.deb gowaves_${VERSION}.deb
	@mv gowaves_${VERSION}.deb ./build/dist
	@rm -rf ./build/gowaves
