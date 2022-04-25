module github.com/wavesplatform/gowaves

go 1.18

// exclude vulnerable dependency: github.com/prometheus/client_golang -> github.com/prometheus/common@v0.4.1 -> vulnerable
exclude github.com/gogo/protobuf v1.1.1

require (
	filippo.io/edwards25519 v1.0.0-rc.1
	github.com/beevik/ntp v0.3.0
	github.com/btcsuite/btcd v0.22.0-beta
	github.com/cespare/xxhash/v2 v2.1.2
	github.com/coocood/freecache v1.2.1
	github.com/ericlagergren/decimal v0.0.0-20210307182354-5f8425a47c58
	github.com/fxamacker/cbor/v2 v2.4.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/howeyc/gopass v0.0.0-20210920133722-c8aef6fb66ef
	github.com/influxdata/influxdb1-client v0.0.0-20200827194710-b269163b24ab
	github.com/jinzhu/copier v0.3.5
	github.com/kilic/bls12-381 v0.0.0-20200820230200-6b2c19996391
	github.com/kilic/bn254 v0.0.0-20201116081810-790649bc68fe
	github.com/mr-tron/base58 v1.2.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/seiflotfy/cuckoofilter v0.0.0-20201222105146-bc6005554a0c
	github.com/semrush/zenrpc/v2 v2.1.1
	github.com/spf13/afero v1.8.2
	github.com/spf13/pflag v1.0.5
	github.com/starius/emsort v0.0.0-20191221202443-6f2fbdee4781
	github.com/steakknife/bloomfilter v0.0.0-20180922174646-6819c0d2a570
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/throttled/throttled/v2 v2.9.1
	github.com/umbracle/fastrlp v0.0.0-20210128110402-41364ca56ca8
	github.com/valyala/bytebufferpool v1.0.0
	github.com/xenolf/lego v2.7.2+incompatible
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20211108221036-ceb1ce70b4fa
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-metro v0.0.0-20200812162917-85c65e2d0165 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210226172003-ab064af71705 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
