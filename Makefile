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
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go" | grep -v ".*bn254/.*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

mod-clean:
	go mod tidy

clean:
	@rm -rf build
	go mod tidy

vendor:
	go mod vendor

vetcheck:
	go list ./... | grep -v bn254 | xargs go vet
	golangci-lint run --skip-dirs pkg/crypto/internal/groth16/bn256/utils/bn254

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

dist-wmd: release-wmd build-wmd-deb-package
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

dist-node: release-node build-node-mainnet-deb-package build-node-testnet-deb-package
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

build-wmd-deb-package: release-wmd
	@mkdir -p build/dist
	@mkdir -p ./build/wmd/DEBIAN
	@cp ./cmd/wmd/symbols.txt ./build/wmd
	@cp ./build/bin/linux-amd64/wmd ./build/wmd
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Waves Market Data System Service|g; s|PACKAGE|wmd|g; s|EXECUTABLE|wmd|g; s|PARAMS|-db /var/lib/wmd/ -address 0.0.0.0:6990 -node grpc.wavesnodes.com:6870 -symbols /usr/share/wmd/symbols.txt -sync-interval 10|g; s|NAME|wmd|g" ./dpkg/service.service > ./build/wmd/wmd.service
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Waves Market Data System Service/g; s/PACKAGE/wmd/g" ./dpkg/control > ./build/wmd/DEBIAN/control
	@sed "s/PACKAGE/wmd/g; s/NAME/wmd/g; s/EXECUTABLE/wmd/g; s|EXTRAS|sudo cp symbols.txt /usr/share/wmd/|g;" ./dpkg/postinst > ./build/wmd/DEBIAN/postinst
	@sed "s/PACKAGE/wmd/g" ./dpkg/preinst > ./build/wmd/DEBIAN/preinst
	@chmod 0775 ./build/wmd/DEBIAN/control
	@chmod 0775 ./build/wmd/DEBIAN/preinst
	@chmod 0775 ./build/wmd/DEBIAN/postinst
	@dpkg-deb --build ./build/wmd
	@mv ./build/wmd.deb ./build/dist/wmd_${VERSION}.deb
	@rm -rf ./build/wmd

build-node-mainnet-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-mainnet/DEBIAN
	@cp ./build/bin/linux-amd64/node ./build/gowaves-mainnet/
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for MainNet System Service|g; s|PACKAGE|gowaves-mainnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-mainnet/ -api-address 0.0.0.0:8080|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-mainnet/gowaves-mainnet.service
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for MainNet System Service/g; s/PACKAGE/gowaves-mainnet/g" ./dpkg/control > ./build/gowaves-mainnet/DEBIAN/control
	@sed "s/PACKAGE/gowaves-mainnet/g; s/NAME/gowaves/g; s/EXECUTABLE/node/g; s|EXTRAS||g;" ./dpkg/postinst > ./build/gowaves-mainnet/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/preinst > ./build/gowaves-mainnet/DEBIAN/preinst
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/control
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/preinst
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/postinst
	@dpkg-deb --build ./build/gowaves-mainnet
	@mv ./build/gowaves-mainnet.deb ./build/dist/gowaves-mainnet_${VERSION}.deb
	@rm -rf ./build/gowaves-mainnet

build-node-testnet-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-testnet/DEBIAN
	@cp ./build/bin/linux-amd64/node ./build/gowaves-testnet/
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for TestNet System Service|g; s|PACKAGE|gowaves-testnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-testnet/ -api-address 0.0.0.0:8090 -blockchain-type testnet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-testnet/gowaves-testnet.service
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for TestNet System Service/g; s/PACKAGE/gowaves-testnet/g" ./dpkg/control > ./build/gowaves-testnet/DEBIAN/control
	@sed "s/PACKAGE/gowaves-testnet/g; s/NAME/gowaves/g; s/EXECUTABLE/node/g; s|EXTRAS||g;" ./dpkg/postinst > ./build/gowaves-testnet/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-testnet/g" ./dpkg/preinst > ./build/gowaves-testnet/DEBIAN/preinst
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/control
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/preinst
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/postinst
	@dpkg-deb --build ./build/gowaves-testnet
	@mv ./build/gowaves-testnet.deb ./build/dist/gowaves-testnet_${VERSION}.deb
	@rm -rf ./build/gowaves-testnet
