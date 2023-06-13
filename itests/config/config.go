package config

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/xenolf/lego/log"
)

const (
	scalaConfigFilename      = "waves.conf"
	goConfigFilename         = "gowaves-it.json"
	templateScalaCfgFilename = "template.conf"

	tmpDir = "../build/config"
)

type TestConfig struct {
	Accounts []AccountInfo
}

func createConfigDir(suiteName string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configDir := filepath.Clean(filepath.Join(pwd, tmpDir, suiteName))
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0744); err != nil {
			return "", err
		}
	}
	return configDir, nil
}

func createScalaNodeConfig(cfg *Config, configDir string) error {
	configPath := filepath.Clean(filepath.Join(configDir, scalaConfigFilename))
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
	templatePath := filepath.Clean(filepath.Join(pwd, configFolder, templateScalaCfgFilename))
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	err = t.Execute(f, cfg)
	if err != nil {
		return err
	}
	return nil
}

func createGoNodeConfig(cfg *Config, configDir string) error {
	configPath := filepath.Clean(filepath.Join(configDir, goConfigFilename))
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

func CreateFileConfigs(suiteName string, enableScalaMining bool) (ConfigPaths, TestConfig, error) {
	cfg, acc, err := NewBlockchainConfig()
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
	return ConfigPaths{ScalaConfigPath: configDir, GoConfigPath: configDir}, TestConfig{Accounts: acc}, nil
}
