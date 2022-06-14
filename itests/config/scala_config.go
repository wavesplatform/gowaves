package config

import (
	"os"
	"path/filepath"
	"text/template"
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
      address-scheme-character = L
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
        average-block-delay = {{.AverageBlockDelay}}s
        initial-base-target = {{.GenesisBaseTarget}}
        initial-balance = 6400000000000000
        timestamp = {{.GenesisTimestamp}}
        block-timestamp = {{.GenesisTimestamp}}
        signature = "{{.GenesisSignature}}"
        transactions = [{{range .Transaction}}
          { recipient = {{.Address}}, amount = {{.Amount}} }{{end}}
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

const (
	scalaConfigFilename = "waves.conf"
)

func CreateNewScalaConfig() error {
	t, err := template.New("scala_conf").Parse(scalaTemplateConfig)
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Clean(filepath.Join(pwd, configFolder, scalaConfigFilename))
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = f.Close()
	}()
	params, err := NewGenesisConfig()
	if err != nil {
		return err
	}

	return t.Execute(f, params)
}

func DeleteScalaConfig() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Clean(filepath.Join(pwd, configFolder, scalaConfigFilename))
	return os.Remove(configPath)
}
