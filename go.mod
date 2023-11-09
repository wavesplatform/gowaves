module github.com/wavesplatform/gowaves

go 1.20

require (
	filippo.io/edwards25519 v1.0.0
	github.com/beevik/ntp v1.3.0
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/consensys/gnark v0.9.1
	github.com/consensys/gnark-crypto v0.12.2-0.20231013160410-1f65e75b6dfb
	github.com/coocood/freecache v1.2.4
	github.com/elliotchance/orderedmap/v2 v2.2.0
	github.com/ericlagergren/decimal v0.0.0-20210307182354-5f8425a47c58
	github.com/fxamacker/cbor/v2 v2.5.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-test/deep v1.1.0
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/howeyc/gopass v0.0.0-20210920133722-c8aef6fb66ef
	github.com/influxdata/influxdb1-client v0.0.0-20200827194710-b269163b24ab
	github.com/jinzhu/copier v0.4.0
	github.com/mr-tron/base58 v1.2.0
	github.com/ory/dockertest/v3 v3.10.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.17.0
	github.com/qmuntal/stateless v1.6.8
	github.com/seiflotfy/cuckoofilter v0.0.0-20201222105146-bc6005554a0c
	github.com/semrush/zenrpc/v2 v2.1.1
	github.com/spf13/afero v1.10.0
	github.com/spf13/pflag v1.0.5
	github.com/starius/emsort v0.0.0-20191221202443-6f2fbdee4781
	github.com/steakknife/bloomfilter v0.0.0-20180922174646-6819c0d2a570
	github.com/stoewer/go-strcase v1.3.0
	github.com/stretchr/testify v1.8.4
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/throttled/throttled/v2 v2.12.0
	github.com/tidwall/sjson v1.2.5
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/umbracle/fastrlp v0.1.0
	github.com/valyala/bytebufferpool v1.0.0
	github.com/xenolf/lego v2.7.2+incompatible
	go.uber.org/atomic v1.11.0
	go.uber.org/zap v1.26.0
	golang.org/x/crypto v0.15.0
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63
	golang.org/x/sync v0.5.0
	golang.org/x/sys v0.14.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	moul.io/zapfilter v1.7.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.8.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/dgryski/go-metro v0.0.0-20200812162917-85c65e2d0165 // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20230817174616-7a8ec2ada47b // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.1.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.4.1-0.20230718164431-9a2bf3000d16 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/rs/zerolog v1.30.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3 // indirect
	github.com/tidwall/gjson v1.14.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/term v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.12.1-0.20230815132531-74c255bcf846 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
