package config

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/xenolf/lego/log"
)

const (
	scalaConfigFilename      = "waves.conf"
	goConfigFilename         = "gowaves-it.json"
	templateScalaCfgFilename = "template.conf"

	tmpDir = "../build/config"
)

type TestConfig struct {
	Accounts           []AccountInfo
	BlockchainSettings *settings.BlockchainSettings
	Env                *goEnvOptions
}

func createConfigDir(suiteName string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(pwd, tmpDir, suiteName)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return "", err
	}
	return configDir, nil
}

func createScalaNodeConfig(cfg *config, configDir string) error {
	configPath := filepath.Join(configDir, scalaConfigFilename)
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Sync(); err != nil {
			log.Warnf("Failed to sync file '%s' to disk: %v", f.Name(), err)
			return
		}
		if err := f.Close(); err != nil {
			log.Warnf("Failed to close file '%s': %v", f.Name(), err)
		}
	}()
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	templatePath := filepath.Join(pwd, configFolder, templateScalaCfgFilename)
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return t.Execute(f, cfg)
}

func createGoNodeConfig(cfg *config, configDir string) error {
	configPath := filepath.Join(configDir, goConfigFilename)
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Sync(); err != nil {
			log.Warnf("Failed to sync file '%s' to disk: %v", f.Name(), err)
			return
		}
		if err := f.Close(); err != nil {
			log.Warnf("Failed to close file '%s': %v", f.Name(), err)
		}
	}()
	jsonWriter := json.NewEncoder(f)
	jsonWriter.SetIndent("", "\t")
	if err := jsonWriter.Encode(cfg.BlockchainSettings); err != nil {
		return errors.Wrap(err, "failed to encode genesis settings")
	}
	return nil
}

type ConfigPaths struct {
	GoConfigPath    string
	ScalaConfigPath string
}

func CreateFileConfigs(suiteName string, enableScalaMining bool,
	additionalArgsPath ...string) (ConfigPaths, TestConfig, error) {
	cfg, acc, err := newBlockchainConfig(additionalArgsPath...)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create blockchain config")
	}
	cfg.ScalaOpts.EnableMining = enableScalaMining
	configDir, err := createConfigDir(suiteName)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create scala-node config")
	}
	if err := createScalaNodeConfig(cfg, configDir); err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create scala-node config")
	}
	if err := createGoNodeConfig(cfg, configDir); err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create go-node config")
	}
	return ConfigPaths{ScalaConfigPath: configDir, GoConfigPath: configDir},
		TestConfig{Accounts: acc, BlockchainSettings: cfg.BlockchainSettings,
			Env: &goEnvOptions{DesiredBlockReward: cfg.GoOpts.DesiredBlockReward,
				SupportedFeatures: cfg.GoOpts.SupportedFeatures}}, nil
}
