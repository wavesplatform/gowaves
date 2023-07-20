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

all: vendor vetcheck fmtcheck gotest mod-clean build-node-native

ci: vendor vetcheck fmtcheck release-node gotest-race-coverage mod-clean

ver:
	@echo Building version: $(VERSION)

gotest:
	go test -cover $$(go list ./... | grep -v "/itests")

gotest-race-coverage:
	go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./... | grep -v "/itests")

gotest-real-node:
	@REAL_NODE=true go test -cover $$(go list ./... | grep -v "/itests")

itest:
	mkdir -p build/config
	mkdir -p build/logs
	go test -timeout 30m -parallel 3 $$(go list ./... | grep "/itests")

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

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
	go vet ./...
	golangci-lint run -c .golangci.yml
	golangci-lint run -c .golangci-strict.yml --new-from-rev=origin/master

strict-vet-check:
	golangci-lint run -c .golangci-strict.yml

build-chaincmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/chaincmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp

release-chaincmp: ver build-chaincmp-linux build-chaincmp-darwin build-chaincmp-windows

dist-chaincmp: release-chaincmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/chaincmp_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/chaincmp*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_Linux-amd64.tar.gz ./chaincmp*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_macOS-amd64.tar.gz ./chaincmp*

build-blockcmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/blockcmp -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp
build-blockcmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/blockcmp -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp
build-blockcmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/blockcmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/blockcmp

release-blockcmp: ver build-blockcmp-linux build-blockcmp-darwin build-blockcmp-windows

dist-blockcmp: release-blockcmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/blockcmp_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/blockcmp*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/blockcmp_$(VERSION)_Linux-amd64.tar.gz ./blockcmp*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/blockcmp_$(VERSION)_macOS-amd64.tar.gz ./blockcmp*

build-node-native:
	@CGO_ENABLE=0 go build -o build/bin/native/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-amd64:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-i386:
	@GOOS=linux GOARCH=386 go build -o build/bin/linux-i386/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-arm:
	@GOOS=linux GOARCH=arm go build -o build/bin/linux-arm/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-linux-arm64:
	@GOOS=linux GOARCH=arm64 go build -o build/bin/linux-arm64/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-darwin-amd64:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/node -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node
build-node-windows-amd64:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/node.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/node

release-node: ver build-node-linux-amd64 build-node-linux-i386 build-node-linux-arm64 build-node-linux-arm build-node-darwin-amd64 build-node-windows-amd64

dist-node: release-node build-node-mainnet-amd64-deb-package build-node-testnet-amd64-deb-package build-node-testnet-arm64-deb-package build-node-stagenet-amd64-deb-package build-node-stagenet-arm64-deb-package
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/node_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/node*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/node_$(VERSION)_Linux-amd64.tar.gz ./node*
	@cd ./build/bin/linux-arm64/; tar pzcvf ../../dist/node_$(VERSION)_Linux-arm64.tar.gz ./node*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/node_$(VERSION)_macOS-amd64.tar.gz ./node*

build-importer-native:
	@CGO_ENABLE=0 go build -o build/bin/native/importer -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer
build-importer-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/importer -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer
build-importer-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/importer -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer
build-importer-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/importer.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/importer

release-importer: ver build-importer-linux build-importer-darwin build-importer-windows

dist-importer: release-importer
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/importer_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/importer*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_Linux-amd64.tar.gz ./importer*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_macOS-amd64.tar.gz ./importer*

build-wallet-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/wallet ./cmd/wallet
build-wallet-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/wallet ./cmd/wallet
build-wallet-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/wallet.exe ./cmd/wallet

release-wallet: ver build-wallet-linux build-wallet-darwin build-wallet-windows

dist-wallet: release-wallet
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/wallet_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/wallet*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_Linux-amd64.tar.gz ./wallet*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/wallet_$(VERSION)_macOS-amd64.tar.gz ./wallet*

build-rollback-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/rollback -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback
build-rollback-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/rollback -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback
build-rollback-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/rollback.exe -ldflags="-X 'github.com/wavesplatform/gowaves/pkg/versioning.Version=$(VERSION)'" ./cmd/rollback

release-rollback: ver build-rollback-linux build-rollback-darwin build-rollback-windows

build-compiler-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/compiler ./cmd/compiler
build-compiler-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/compiler ./cmd/compiler
build-compiler-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/compiler.exe ./cmd/compiler

release-compiler: ver build-compiler-linux build-compiler-darwin build-compiler-windows

build-statehash-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/statehash ./cmd/statehash
build-statehash-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/statehash ./cmd/statehash
build-statehash-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/statehash.exe ./cmd/statehash

release-statehash: ver build-statehash-linux build-statehash-darwin build-statehash-windows

dist-compiler: release-compiler
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/compiler_$(VERSION)_Windows-amd64.zip ./bin/windows-amd64/compiler*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/compiler_$(VERSION)_Linux-amd64.tar.gz ./compiler*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/compiler_$(VERSION)_macOS-amd64.tar.gz ./compiler*

dist: clean dist-chaincmp dist-importer dist-node dist-wallet dist-compiler

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

build-node-mainnet-amd64-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-mainnet-amd64/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for MainNet System Service/g; s/PACKAGE/gowaves-mainnet/g; s/ARCH/amd64/g" ./dpkg/control > ./build/gowaves-mainnet-amd64/DEBIAN/control
	@sed "s/PACKAGE/gowaves-mainnet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-mainnet-amd64/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/postrm > ./build/gowaves-mainnet-amd64/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/prerm > ./build/gowaves-mainnet-amd64/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-mainnet-amd64/DEBIAN/control
	@chmod 0775 ./build/gowaves-mainnet-amd64/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-mainnet-amd64/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-mainnet-amd64/DEBIAN/prerm

	@mkdir -p ./build/gowaves-mainnet-amd64/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for MainNet System Service|g; s|PACKAGE|gowaves-mainnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-mainnet/ -api-address 0.0.0.0:8080|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-mainnet-amd64/lib/systemd/system/gowaves-mainnet.service

	@mkdir -p ./build/gowaves-mainnet-amd64/usr/share/gowaves-mainnet
	@cp ./build/bin/linux-amd64/node ./build/gowaves-mainnet-amd64/usr/share/gowaves-mainnet

	@mkdir -p ./build/gowaves-mainnet-amd64/var/lib/gowaves-mainnet/
	@mkdir -p ./build/gowaves-mainnet-amd64/var/log/gowaves-mainnet/

	@dpkg-deb --build ./build/gowaves-mainnet-amd64
	@mv ./build/gowaves-mainnet-amd64.deb ./build/dist/gowaves-mainnet-amd64_${VERSION}.deb
	@rm -rf ./build/gowaves-mainnet-amd64

build-node-mainnet-arm64-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-mainnet-arm64/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for MainNet System Service/g; s/PACKAGE/gowaves-mainnet/g; s/ARCH/arm64/g" ./dpkg/control > ./build/gowaves-mainnet-arm64/DEBIAN/control
	@sed "s/PACKAGE/gowaves-mainnet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-mainnet-arm64/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/postrm > ./build/gowaves-mainnet-arm64/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-mainnet/g" ./dpkg/prerm > ./build/gowaves-mainnet-arm64/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-mainnet-arm64/DEBIAN/control
	@chmod 0775 ./build/gowaves-mainnet-arm64/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-mainnet-arm64/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-mainnet-arm64/DEBIAN/prerm

	@mkdir -p ./build/gowaves-mainnet-arm64/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for MainNet System Service|g; s|PACKAGE|gowaves-mainnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-mainnet/ -api-address 0.0.0.0:8080|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-mainnet-arm64/lib/systemd/system/gowaves-mainnet.service

	@mkdir -p ./build/gowaves-mainnet-arm64/usr/share/gowaves-mainnet
	@cp ./build/bin/linux-arm64/node ./build/gowaves-mainnet-arm64/usr/share/gowaves-mainnet

	@mkdir -p ./build/gowaves-mainnet-arm64/var/lib/gowaves-mainnet/
	@mkdir -p ./build/gowaves-mainnet-arm64/var/log/gowaves-mainnet/

	@dpkg-deb --build ./build/gowaves-mainnet-arm64
	@mv ./build/gowaves-mainnet-arm64.deb ./build/dist/gowaves-mainnet-arm64_${VERSION}.deb
	@rm -rf ./build/gowaves-mainnet-arm64

build-node-testnet-amd64-deb-package: release-node
	@mkdir -p build/dist

	@mkdir -p ./build/gowaves-testnet/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for TestNet System Service/g; s/PACKAGE/gowaves-testnet/g; s/ARCH/amd64/g" ./dpkg/control > ./build/gowaves-testnet/DEBIAN/control
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

build-node-testnet-arm64-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-testnet-arm64/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for TestNet System Service/g; s/PACKAGE/gowaves-testnet/g; s/ARCH/arm64/g" ./dpkg/control > ./build/gowaves-testnet-arm64/DEBIAN/control
	@sed "s/PACKAGE/gowaves-testnet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-testnet-arm64/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-testnet/g" ./dpkg/postrm > ./build/gowaves-testnet-arm64/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-testnet/g" ./dpkg/prerm > ./build/gowaves-testnet-arm64/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-testnet-arm64/DEBIAN/control
	@chmod 0775 ./build/gowaves-testnet-arm64/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-testnet-arm64/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-testnet-arm64/DEBIAN/prerm

	@mkdir -p ./build/gowaves-testnet-arm64/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for TestNet System Service|g; s|PACKAGE|gowaves-testnet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-testnet/ -api-address 0.0.0.0:8090 -blockchain-type testnet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-testnet-arm64/lib/systemd/system/gowaves-testnet.service

	@mkdir -p ./build/gowaves-testnet-arm64/usr/share/gowaves-testnet
	@cp ./build/bin/linux-arm64/node ./build/gowaves-testnet-arm64/usr/share/gowaves-testnet

	@mkdir -p ./build/gowaves-testnet-arm64/var/lib/gowaves-testnet/
	@mkdir -p ./build/gowaves-testnet-arm64/var/log/gowaves-testnet/

	@dpkg-deb --build ./build/gowaves-testnet-arm64
	@mv ./build/gowaves-testnet-arm64.deb ./build/dist/gowaves-testnet-arm64_${VERSION}.deb
	@rm -rf ./build/gowaves-testnet-arm64

build-node-stagenet-amd64-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-stagenet-amd64/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for StageNet System Service/g; s/PACKAGE/gowaves-stagenet/g; s/ARCH/amd64/g" ./dpkg/control > ./build/gowaves-stagenet-amd64/DEBIAN/control
	@sed "s/PACKAGE/gowaves-stagenet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-stagenet-amd64/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/postrm > ./build/gowaves-stagenet-amd64/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/prerm > ./build/gowaves-stagenet-amd64/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-stagenet-amd64/DEBIAN/control
	@chmod 0775 ./build/gowaves-stagenet-amd64/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-stagenet-amd64/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-stagenet-amd64/DEBIAN/prerm

	@mkdir -p ./build/gowaves-stagenet-amd64/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for StageNet System Service|g; s|PACKAGE|gowaves-stagenet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-stagenet/ -api-address 0.0.0.0:8100 -blockchain-type stagenet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-stagenet-amd64/lib/systemd/system/gowaves-stagenet.service

	@mkdir -p ./build/gowaves-stagenet-amd64/usr/share/gowaves-stagenet
	@cp ./build/bin/linux-amd64/node ./build/gowaves-stagenet-amd64/usr/share/gowaves-stagenet-amd64

	@mkdir -p ./build/gowaves-stagenet-amd64/var/lib/gowaves-stagenet/
	@mkdir -p ./build/gowaves-stagenet-amd64/var/log/gowaves-stagenet/

	@dpkg-deb --build ./build/gowaves-stagenet-amd64
	@mv ./build/gowaves-stagenet-amd64.deb ./build/dist/gowaves-stagenet-amd64_${VERSION}.deb
	@rm -rf ./build/gowaves-stagenet-amd64

build-node-stagenet-arm64-deb-package: release-node
	@mkdir -p build/dist
	@mkdir -p ./build/gowaves-stagenet-arm64/DEBIAN
	@sed "s/DEB_VER/$(DEB_VER)/g; s/VERSION/$(VERSION)/g; s/DESCRIPTION/Gowaves Node for StageNet System Service/g; s/PACKAGE/gowaves-stagenet/g; s/ARCH/arm64/g" ./dpkg/control > ./build/gowaves-stagenet-arm64/DEBIAN/control
	@sed "s/PACKAGE/gowaves-stagenet/g; s/NAME/gowaves/g;" ./dpkg/postinst > ./build/gowaves-stagenet-arm64/DEBIAN/postinst
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/postrm > ./build/gowaves-stagenet-arm64/DEBIAN/postrm
	@sed "s/PACKAGE/gowaves-stagenet/g" ./dpkg/prerm > ./build/gowaves-stagenet-arm64/DEBIAN/prerm
	@chmod 0644 ./build/gowaves-stagenet-arm64/DEBIAN/control
	@chmod 0775 ./build/gowaves-stagenet-arm64/DEBIAN/postinst
	@chmod 0775 ./build/gowaves-stagenet-arm64/DEBIAN/postrm
	@chmod 0775 ./build/gowaves-stagenet-arm64/DEBIAN/prerm

	@mkdir -p ./build/gowaves-stagenet-arm64/lib/systemd/system
	@sed "s|VERSION|$(VERSION)|g; s|DESCRIPTION|Gowaves Node for StageNet System Service|g; s|PACKAGE|gowaves-stagenet|g; s|EXECUTABLE|node|g; s|PARAMS|-state-path /var/lib/gowaves-stagenet/ -api-address 0.0.0.0:8100 -blockchain-type stagenet|g; s|NAME|gowaves|g" ./dpkg/service.service > ./build/gowaves-stagenet-arm64/lib/systemd/system/gowaves-stagenet.service

	@mkdir -p ./build/gowaves-stagenet-arm64/usr/share/gowaves-stagenet
	@cp ./build/bin/linux-arm64/node ./build/gowaves-stagenet-arm64/usr/share/gowaves-stagenet

	@mkdir -p ./build/gowaves-stagenet-arm64/var/lib/gowaves-stagenet/
	@mkdir -p ./build/gowaves-stagenet-arm64/var/log/gowaves-stagenet/

	@dpkg-deb --build ./build/gowaves-stagenet-arm64
	@mv ./build/gowaves-stagenet-arm64.deb ./build/dist/gowaves-stagenet-arm64_${VERSION}.deb
	@rm -rf ./build/gowaves-stagenet-arm64
