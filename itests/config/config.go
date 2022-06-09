package config

import (
	"html/template"
	"os"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
)

const scalaTemplateConfig = `waves {
  blockchain.type = CUSTOM
  directory = /tmp/waves
  ntp-server = "0.ru.pool.ntp.org"
  network {
    known-peers = []
    black-list-residence-time = 30s
    peers-broadcast-interval = 2s
    connection-timeout = 30s
    suspension-residence-time = 5s

    traffic-logger {
      ignore-tx-messages = [1, 2]
      ignore-rx-messages = [1, 2]
    }
  }
  synchronization {
    utx-synchronizer.max-queue-size = 20000
    invalid-blocks-storage.timeout = 100ms
  }
  blockchain {
    type = CUSTOM
    custom {
      address-scheme-character = I
      functionality {
        feature-check-blocks-period = 1
        blocks-for-feature-activation = 1
        generation-balance-depth-from-50-to-1000-after-height = 0
        reset-effective-balances-at-height = 0
        block-version-3-after-height = 0
        last-time-based-fork-parameter = 0
        double-features-periods-after-height = 100000000
        max-transaction-time-back-offset = 120m
        max-transaction-time-forward-offset = 90m
        pre-activated-features = {
          2 = 0
          3 = 0
          4 = 0
          5 = 0
          6 = 0
          7 = -${waves.blockchain.custom.functionality.feature-check-blocks-period}
          9 = 0
          10 = 0
          11 = 0
          12 = 0
          13 = 0
          14 = 2
          15 = 0
        }
        min-block-time = 5s
        delay-delta = 0
      }
      rewards {
        term = 100000
        initial = 600000000
        min-increment = 50000000
        voting-interval = 10000
      }
      # These fields are ignored: timestamp, block-timestamp, signature. They are generated in integration tests.
      genesis {
        average-block-delay = 10s
        initial-base-target = {{.GenesisBaseTarget}}
        initial-balance = 6400000000000000
        timestamp = {{.GenesisTimestamp}}
        block-timestamp = {{.GenesisTimestamp}}
        signature = "{{.GenesisSignature}}"
        transactions = [
          # Initial balances are balanced (pun intended) in such way that initial block
          # generation delay doesn't vary much, no matter which node is chosen as a miner.
          { recipient = 3Hm3LGoNPmw1VTZ3eRA2pAfeQPhnaBm6YFC, amount =   10000000000000 }
          { recipient = 3HPG313x548Z9kJa5XY4LVMLnUuF77chcnG, amount =   15000000000000 }
          { recipient = 3HZxhQhpSU4yEGJdGetncnHaiMnGmUusr9s, amount =   25000000000000 }
          { recipient = 3HVW7RDYVkcN5xFGBNAUnGirb5KaBSnbUyB, amount =   25000000000000 }
          { recipient = 3Hi5pLwXXo3WeGEg2HgeDcy4MjQRTgz7WRx, amount =   40000000000000 }
          { recipient = 3HhtyiszMEhXdWzGgvxcfgfJdzrgfgyWcQq, amount =   45000000000000 }
          { recipient = 3HRVTkn9BdxxQJ6PDr2qibXVdTtK2D5uzRF, amount =   60000000000000 }
          { recipient = 3HQvEJwjxskvcKLC79XpQk6PQeNxGibozrq, amount =   80000000000000 }
          { recipient = 3HnGfdhUuA948dYjQHnrz2ZHxfT4P72oBBy, amount =  100000000000000 }
          { recipient = 3HmFkAoQRs4Y3PE2uR6ohN7wS4VqPBGKv7k, amount = 6000000000000000 }
        ]
      }
    }
  }
  features.auto-shutdown-on-unsupported-feature = no
  matcher.enable = no
  miner {
    enable = yes
    quorum = 1
    interval-after-last-block-then-generation-is-allowed = 1h
    micro-block-interval = 5s
    min-micro-block-age = 0s
  }
  rest-api {
    enable = yes
    bind-address = 0.0.0.0
    api-key-hash = 7L6GpLHhA5KyJTAVc8WFHwEcyTY8fC8rRbyMCiFnM4i
    api-key-different-host = yes
    minimum-peers = 0
  }
  wallet {
    file = "wallet"
    password = test
  }
  utx {
    max-scripted-size = 100000
    allow-skip-checks = false
  }
  extensions = [
    com.wavesplatform.api.grpc.GRPCServerExtension
  ]
  grpc {
    host = 0.0.0.0
    port = 6870
  }
}

akka.actor.debug {
  lifecycle = on
  unhandled = on
}`

type GenesisParams struct {
	GenesisTimestamp  int64
	GenesisSignature  crypto.Signature
	GenesisBaseTarget uint64
}

func CreateNewScalaConfig() error {
	t, err := template.New("scala_conf").Parse(scalaTemplateConfig)
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	f, err := os.Create(pwd + "/config/waves.conf")
	defer f.Close()
	if err != nil {
		return err
	}
	params, err := NewGenesisParams()
	if err != nil {
		return err
	}

	return t.Execute(f, params)
}

type GenesisTransactionInfoRaw struct {
	Address string
	Amount  uint64
}

var (
	transactionInfo = []GenesisTransactionInfoRaw{
		{Address: "3Hm3LGoNPmw1VTZ3eRA2pAfeQPhnaBm6YFC", Amount: 10000000000000},
		{Address: "3HPG313x548Z9kJa5XY4LVMLnUuF77chcnG", Amount: 15000000000000},
		{Address: "3HZxhQhpSU4yEGJdGetncnHaiMnGmUusr9s", Amount: 25000000000000},
		{Address: "3HVW7RDYVkcN5xFGBNAUnGirb5KaBSnbUyB", Amount: 25000000000000},
		{Address: "3Hi5pLwXXo3WeGEg2HgeDcy4MjQRTgz7WRx", Amount: 40000000000000},
		{Address: "3HhtyiszMEhXdWzGgvxcfgfJdzrgfgyWcQq", Amount: 45000000000000},
		{Address: "3HRVTkn9BdxxQJ6PDr2qibXVdTtK2D5uzRF", Amount: 60000000000000},
		{Address: "3HQvEJwjxskvcKLC79XpQk6PQeNxGibozrq", Amount: 80000000000000},
		{Address: "3HnGfdhUuA948dYjQHnrz2ZHxfT4P72oBBy", Amount: 100000000000000},
		{Address: "3HmFkAoQRs4Y3PE2uR6ohN7wS4VqPBGKv7k", Amount: 6000000000000000},
	}
)

const scheme = 'L'

func CreateTransactionInfo(ts uint64) ([]genesis_generator.GenesisTransactionInfo, error) {
	var res []genesis_generator.GenesisTransactionInfo
	for _, info := range transactionInfo {
		addr, err := proto.NewAddressFromString(info.Address)
		if err != nil {
			return nil, err
		}
		res = append(res, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: info.Amount, Timestamp: ts})
	}
	return res, nil
}

func NewGenesisParams() (GenesisParams, error) {
	ts := time.Now().UnixMilli()
	var bs uint64 = 124412412
	txs, err := CreateTransactionInfo(uint64(ts))
	if err != nil {
		return GenesisParams{}, err
	}
	b, err := genesis_generator.GenerateGenesisBlock(scheme, txs, bs, uint64(ts))
	if err != nil {
		return GenesisParams{}, err
	}
	return GenesisParams{
		GenesisTimestamp:  ts,
		GenesisSignature:  b.BlockSignature,
		GenesisBaseTarget: bs,
	}, nil
}

func DeleteConfig() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return os.Remove(pwd + "/config/waves.conf")
}
