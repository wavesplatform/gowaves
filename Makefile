PROJECT=gowaves
ORGANISATION=wavesplatform
MODULE=github.com/$(ORGANISATION)/$(PROJECT)
SOURCE=$(shell find . -name '*.go' | grep -v vendor/)
SOURCE_DIRS = cmd pkg

VERSION=$(shell git describe --tags --always --dirty)
DEB_VER=$(shell git describe --tags --abbrev=0 | cut -c 2-)
DEB_HASH=$(shell git rev-parse HEAD)

export GO111MODULE=on

.PHONY: vendor vetcheck fmtcheck clean build gotest update-go-deps

all: vendor vetcheck fmtcheck build gotest mod-clean

ci: vendor vetcheck fmtcheck build release-node gotest-race-coverage mod-clean

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
	go test -cover $$(go list ./... | grep -v "/itests")

gotest-race-coverage:
	go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./... | grep -v "/itests")

itest:
	mkdir -p build/config
	mkdir -p build/logs
	go test -timeout 20m -parallel 3 $$(go list ./... | grep "/itests")

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go" | grep -v ".*bn254/.*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

mod-clean:
	go mod tidy

update-go-deps: mod-clean
	@echo ">> updating Go dependencies"
	@for m in $$(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		go get $$m; \
	done
	go mod tidy
ifneq (,$(wildcard vendor))
	go mod vendor
endif

clean:
	@rm -rf build
	go mod tidy

vendor:
	go mod vendor

vetcheck:
	go list ./... | grep -v bn254 | xargs go vet
	golangci-lint run --skip-dirs pkg/crypto/internal/groth16/bn256/utils/bn254 --timeout 5m

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

build-blockcmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/blockcmp -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp
build-blockcmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/blockcmp -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp
build-blockcmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/blockcmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp

release-blockcmp: ver build-blockcmp-linux build-blockcmp-darwin build-blockcmp-windows

dist-blockcmp: release-blockcmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/blockcmp_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/blockcmp*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/blockcmp_$(VERSION)_Linux-64bit.tar.gz ./blockcmp*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/blockcmp_$(VERSION)_macOS-64bit.tar.gz ./blockcmp*

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
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-arm:
	@GOOS=linux GOARCH=arm go build -o build/bin/linux-arm/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-i386:
	@GOOS=linux GOARCH=386 go build -o build/bin/linux-i386/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/node.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node

release-node: ver build-node-linux build-node-linux-arm build-node-linux-i386 build-node-darwin build-node-windows

dist-node: release-node build-node-mainnet-deb-package build-node-testnet-deb-package build-node-stagenet-deb-package
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/node_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/node*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/node_$(VERSION)_Linux-64bit.tar.gz ./node*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/node_$(VERSION)_macOS-64bit.tar.gz ./node*

build-custom-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/custom -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/custom
build-custom-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/custom -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/custom
build-custom-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/custom.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/custom

build-custom: ver build-custom-linux build-custom-darwin build-custom-windows

build-importer-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/importer -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer
build-importer-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/importer -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer
build-importer-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/importer.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer

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

dist-wallet: release-wallet
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/wallet_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/wallet*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_Linux-64bit.tar.gz ./wallet*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_macOS-64bit.tar.gz ./wallet*

build-rollback-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/rollback -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback
build-rollback-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/rollback -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback
build-rollback-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/rollback.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback

release-rollback: ver build-rollback-linux build-rollback-darwin build-rollback-windows

dist: clean dist-chaincmp dist-wmd dist-importer dist-node dist-wallet


build-genconfig:
	go build -o build/bin/darwin-amd64/genconfig ./cmd/genconfig

mock:
	mockgen -source pkg/miner/utxpool/cleaner.go -destination pkg/miner/utxpool/mock.go -package utxpool stateWrapper
	mockgen -source pkg/node/peer_manager/peer_manager.go -destination pkg/mock/peer_manager.go -package mock PeerManager
	mockgen -source pkg/node/peer_manager/peer_storage.go -destination pkg/mock/peer_storage.go -package mock PeerStorage
	mockgen -source pkg/p2p/peer/peer.go -destination pkg/mock/peer.go -package mock Peer
	mockgen -source pkg/state/api.go -destination pkg/mock/state.go -package mock State
	mockgen -source pkg/node/state_fsm/default.go -destination pkg/node/state_fsm/default_mock.go -package state_fsm Default
	mockgen -source pkg/grpc/server/api.go -destination pkg/mock/grpc.go -package mock GrpcHandlers

proto:
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=./ --go_opt=module=$(MODULE) --go-vtproto_out=./ --go-vtproto_opt=features=marshal_strict+unmarshal+size --go-vtproto_opt=module=$(MODULE) pkg/grpc/protobuf-schemas/proto/waves/*.proto
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=./ --go_opt=module=$(MODULE) --go-grpc_out=./ --go-grpc_opt=require_unimplemented_servers=false --go-grpc_opt=module=$(MODULE) pkg/grpc/protobuf-schemas/proto/waves/node/grpc/*.proto
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=./ --go_opt=module=$(MODULE) pkg/grpc/protobuf-schemas/proto/waves/lang/*.proto
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=./ --go_opt=module=$(MODULE) pkg/grpc/protobuf-schemas/proto/waves/events/*.proto
	@protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=./ --go_opt=module=$(MODULE) --go-grpc_out=./ --go-grpc_opt=require_unimplemented_servers=false --go-grpc_opt=module=$(MODULE) pkg/grpc/protobuf-schemas/proto/waves/events/grpc/*.proto

build-wmd-deb-package: release-wmd
	@mkdir -p build/dist

	@mkdir -p ./build/wmd/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Waves Market Data System Service/g; s/PACKAGE/wmd/g" ./dpkg/control > ./build/wmd/DEBIAN/control
	@sed "s/PACKAGE/wmd/g; s/NAME/wmd/g;" ./dpkg/postinst > ./build/wmd/DEBIAN/postinst
	@sed "s/PACKAGE/wmd/g" ./dpkg/postrm > ./build/wmd/DEBIAN/postrm
	@sed "s/PACKAGE/wmd/g" ./dpkg/prerm > ./build/wmd/DEBIAN/prerm
	@chmod 0644 ./build/wmd/DEBIAN/control
	@chmod 0775 ./build/wmd/DEBIAN/postinst
	@chmod 0775 ./build/wmd/DEBIAN/postrm
	@chmod 0775 ./build/wmd/DEBIAN/prerm

	@mkdir -p ./build/wmd/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Waves Market Data System Service|g; s|PACKAGE|wmd|g; s|EXECUTABLE|wmd|g; s|PARAMS|-db /var/lib/wmd/ -address 0.0.0.0:6990 -node grpc.wavesnodes.com:6870 -symbols /usr/share/wmd/symbols.txt -sync-interval 10|g; s|NAME|wmd|g" ./dpkg/service.service > ./build/wmd/lib/systemd/system/wmd.service

	@mkdir -p ./build/wmd/usr/share/wmd
	@cp ./cmd/wmd/symbols.txt ./build/wmd/usr/share/wmd/
	@cp ./build/bin/linux-amd64/wmd ./build/wmd/usr/share/wmd/

	@mkdir -p ./build/wmd/var/lib/wmd/
	@mkdir -p ./build/wmd/var/log/wmd/

	@dpkg-deb --build ./build/wmd
	@mv ./build/wmd.deb ./build/dist/wmd_${VERSION}.deb
	@rm -rf ./build/wmd

build-node-mainnet-deb-package: release-node
	@mkdir -p build/dist

	@mkdir -p ./build/gowaves-mainnet/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for MainNet System Service/g; s/PACKAGE/gowaves-mainnet/g" ./dpkg/control > ./build/gowaves-mainnet/DEBIAN/control
	@sed "s/PACKAGE/gowaves-mainnet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-mainnet/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/postrm > ./build/gowaves-mainnet/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/prerm > ./build/gowaves-mainnet/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-mainnet/DEBIAN/control
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-mainnet/DEBIAN/prerm

	@mkdir -p ./build/gowaves-mainnet/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for MainNet System Service|g; s|PACKAGE|gowaves-mainnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-mainnet/ -api-address 0.0.0.0:8080|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-mainnet/lib/systemd/system/gowaves-mainnet.service

	@mkdir -p ./build/gowaves-mainnet/usr/share/gowaves-mainnet
	@cp ./build/bin/linux-amd64/node ./build/gowaves-mainnet/usr/share/gowaves-mainnet

	@mkdir -p ./build/gowaves-mainnet/var/lib/gowaves-mainnet/
	@mkdir -p ./build/gowaves-mainnet/var/log/gowaves-mainnet/

	@dpkg-deb --build ./build/gowaves-mainnet
	@mv ./build/gowaves-mainnet.deb ./build/dist/gowaves-mainnet_${VERSION}.deb
	@rm -rf ./build/gowaves-mainnet

build-node-testnet-deb-package: release-node
	@mkdir -p build/dist

	@mkdir -p ./build/gowaves-testnet/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for TestNet System Service/g; s/PACKAGE/gowaves-testnet/g" ./dpkg/control > ./build/gowaves-testnet/DEBIAN/control
	@sed "s/PACKAGE/gowaves-testnet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-testnet/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-testnet/g" ./dpkg/postrm > ./build/gowaves-testnet/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-testnet/g" ./dpkg/prerm > ./build/gowaves-testnet/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-testnet/DEBIAN/control
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-testnet/DEBIAN/prerm

	@mkdir -p ./build/gowaves-testnet/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for TestNet System Service|g; s|PACKAGE|gowaves-testnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-testnet/ -api-address 0.0.0.0:8090 -blockchain-type testnet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-testnet/lib/systemd/system/gowaves-testnet.service

	@mkdir -p ./build/gowaves-testnet/usr/share/gowaves-testnet
	@cp ./build/bin/linux-amd64/node ./build/gowaves-testnet/usr/share/gowaves-testnet

	@mkdir -p ./build/gowaves-testnet/var/lib/gowaves-testnet/
	@mkdir -p ./build/gowaves-testnet/var/log/gowaves-testnet/

	@dpkg-deb --build ./build/gowaves-testnet
	@mv ./build/gowaves-testnet.deb ./build/dist/gowaves-testnet_${VERSION}.deb
	@rm -rf ./build/gowaves-testnet

build-node-stagenet-deb-package: release-node
	@mkdir -p build/dist

	@mkdir -p ./build/gowaves-stagenet/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for StageNet System Service/g; s/PACKAGE/gowaves-stagenet/g" ./dpkg/control > ./build/gowaves-stagenet/DEBIAN/control
	@sed "s/PACKAGE/gowaves-stagenet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-stagenet/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/postrm > ./build/gowaves-stagenet/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/prerm > ./build/gowaves-stagenet/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-stagenet/DEBIAN/control
	@chmod 0775 ./build/gowaves-stagenet/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-stagenet/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-stagenet/DEBIAN/prerm

	@mkdir -p ./build/gowaves-stagenet/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for StageNet System Service|g; s|PACKAGE|gowaves-stagenet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-stagenet/ -api-address 0.0.0.0:8100 -blockchain-type stagenet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-stagenet/lib/systemd/system/gowaves-stagenet.service

	@mkdir -p ./build/gowaves-stagenet/usr/share/gowaves-stagenet
	@cp ./build/bin/linux-amd64/node ./build/gowaves-stagenet/usr/share/gowaves-stagenet

	@mkdir -p ./build/gowaves-stagenet/var/lib/gowaves-stagenet/
	@mkdir -p ./build/gowaves-stagenet/var/log/gowaves-stagenet/

	@dpkg-deb --build ./build/gowaves-stagenet
	@mv ./build/gowaves-stagenet.deb ./build/dist/gowaves-stagenet_${VERSION}.deb
	@rm -rf ./build/gowaves-stagenet
