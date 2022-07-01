package config

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	scalaConfigFilename      = "waves.conf"
	goConfigFilename         = "gowaves-it.json"
	templateScalaCfgFilename = "template.conf"

	tmpDir = "../build"
)

type TestConfig struct {
	Accounts []AccountInfo
}

func CreateScalaNodeConfig(cfg *settings.BlockchainSettings) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configPath := filepath.Clean(filepath.Join(pwd, tmpDir, scalaConfigFilename))
	f, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
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

func CreateGoNodeConfig(cfg *settings.BlockchainSettings) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configPath := filepath.Clean(filepath.Join(pwd, tmpDir, goConfigFilename))
	f, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	jsonWriter := json.NewEncoder(f)
	jsonWriter.SetIndent("", "\t")
	if err := jsonWriter.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode genesis settings: %s", err)
	}
	return filepath.Clean(filepath.Join(pwd, tmpDir)), nil
}

type ConfigPaths struct {
	GoConfigPath    string
	ScalaConfigPath string
}

func CreateFileConfigs() (ConfigPaths, TestConfig, error) {
	cfg, acc, err := NewBlockchainConfig()
	if err != nil {
		return ConfigPaths{}, TestConfig{}, fmt.Errorf("failed to create blockchain config: %s", err)
	}
	scalaPath, err := CreateScalaNodeConfig(cfg)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, err
	}
	goPath, err := CreateGoNodeConfig(cfg)
	if err != nil {
		return ConfigPaths{}, TestConfig{}, err
	}
	return ConfigPaths{ScalaConfigPath: scalaPath, GoConfigPath: goPath}, TestConfig{Accounts: acc}, nil
}
