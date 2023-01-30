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

func CreateScalaNodeConfig(cfg *Config) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(filepath.Join(pwd, tmpDir), os.ModePerm)
	if err != nil {
		return "", err
	}
	configPath := filepath.Clean(filepath.Join(pwd, tmpDir, scalaConfigFilename))
	f, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warnf("Failed to close file %s", err)
		}
	}()
	templatePath := filepath.Clean(filepath.Join(pwd, configFolder, templateScalaCfgFilename))
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}
	err = t.Execute(f, cfg)
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(pwd, tmpDir)), nil
}

func CreateGoNodeConfig(cfg *Config) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configPath := filepath.Clean(filepath.Join(pwd, tmpDir, goConfigFilename))
	f, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warnf("Failed to close file %s", err)
		}
	}()
	jsonWriter := json.NewEncoder(f)
	jsonWriter.SetIndent("", "\t")
	if err := jsonWriter.Encode(cfg.BlockchainSettings); err != nil {
		return "", errors.Wrap(err, "failed to encode genesis settings")
	}
	return filepath.Clean(filepath.Join(pwd, tmpDir)), nil
}

type ConfigPaths struct {
	GoConfigPath    string
	ScalaConfigPath string
}

func CreateFileConfigs(enableScalaMining bool) (ConfigPaths, TestConfig, error) {
	cfg, acc, err := NewBlockchainConfig()
	if err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create blockchain config")
	}
	cfg.ScalaOpts.EnableMining = enableScalaMining
	scalaPath, err := CreateScalaNodeConfig(cfg)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create scala-node config")
	}
	goPath, err := CreateGoNodeConfig(cfg)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, errors.Wrap(err, "failed to create go-node config")
	}
	return ConfigPaths{ScalaConfigPath: scalaPath, GoConfigPath: goPath}, TestConfig{Accounts: acc}, nil
}
