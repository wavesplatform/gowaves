package config

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ory/dockertest/v3"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	scalaConfigFilename      = "waves.conf"
	goConfigFilename         = "gowaves-it.json"
	templateScalaCfgFilename = "template.conf"

	tmpDir = "../build/config"

	walletPath = "wallet"

	scalaContainerName = "scala-node"
	goContainerName    = "go-node"
)

const (
	RESTAPIPort = "6869"
	GRPCAPIPort = "6870"
	BindPort    = "6868"
	Localhost   = "0.0.0.0"
	NetTCP      = "/tcp"
)

type TestConfig struct {
	Accounts           []AccountInfo
	BlockchainSettings *settings.BlockchainSettings
}

type DockerConfigurator interface {
	DockerRunOptions() *dockertest.RunOptions
}

type ScalaConfigurator struct {
	suite        string
	cfg          *BlockchainConfig
	configFolder string
	knownPeers   []string
}

func NewScalaConfigurator(suite string, cfg *BlockchainConfig) (*ScalaConfigurator, error) {
	c := &ScalaConfigurator{suite: suite, cfg: cfg}
	if err := c.createNodeConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to create scala node configuration")
	}
	return c, nil
}

func (c *ScalaConfigurator) WithGoNode(goNodeIP string) *ScalaConfigurator {
	c.knownPeers = append(c.knownPeers, goNodeIP)
	return c
}

func (c *ScalaConfigurator) DockerRunOptions() *dockertest.RunOptions {
	kpb := new(strings.Builder)
	for i, kp := range c.knownPeers {
		kpb.WriteString("-Dwaves.network.known-peers.")
		kpb.WriteString(strconv.Itoa(i))
		kpb.WriteString("=")
		kpb.WriteString(kp)
		kpb.WriteString(":")
		kpb.WriteString(BindPort)
		kpb.WriteString(" ")
	}
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       c.suite + "-" + scalaContainerName,
		Tag:        "latest",
		Hostname:   "scala-node",
		Mounts: []string{
			c.configFolder + ":/etc/waves",
		},
		Env: []string{
			"WAVES_LOG_LEVEL=TRACE",
			"WAVES_NETWORK=custom",
			"JAVA_OPTS=" +
				kpb.String() +
				"-Dwaves.network.declared-address=scala-node:" + BindPort + " " +
				"-Dwaves.network.port=" + BindPort + " " +
				"-Dwaves.rest-api.port=" + RESTAPIPort + " " +
				"-Dwaves.grpc.port=" + GRPCAPIPort + " " +
				"-Dwaves.network.enable-blacklisting=no",
		},
		ExposedPorts: []string{
			GRPCAPIPort + NetTCP,
			RESTAPIPort + NetTCP,
			BindPort + NetTCP,
		},
	}
	return opt
}

func (c *ScalaConfigurator) createNodeConfig() error {
	configDir, err := createConfigDir(c.suite)
	if err != nil {
		return errors.Wrap(err, "failed to create scala node configuration")
	}
	configPath := filepath.Join(configDir, scalaConfigFilename)
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if synErr := f.Sync(); synErr != nil {
			log.Printf("Failed to sync file %q to disk: %v", f.Name(), err)
			return
		}
		if clErr := f.Close(); clErr != nil {
			log.Printf("Failed to close file %q: %v", f.Name(), clErr)
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
	if exErr := t.Execute(f, c.cfg); exErr != nil {
		return errors.Wrap(exErr, "failed to create scala node configuration")
	}
	c.configFolder = configDir
	return nil
}

type GoConfigurator struct {
	suite        string
	cfg          *BlockchainConfig
	configFolder string
	walletFolder string
}

func NewGoConfigurator(suite string, cfg *BlockchainConfig) (*GoConfigurator, error) {
	c := &GoConfigurator{suite: suite, cfg: cfg}
	if err := c.createNodeConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to create go node configuration")
	}
	if err := c.setAndVerifyWalletFolder(); err != nil {
		return nil, errors.Wrap(err, "failed to create go node configuration")
	}
	return c, nil
}

func (c *GoConfigurator) DockerRunOptions() *dockertest.RunOptions {
	opt := &dockertest.RunOptions{
		Repository: "go-node",
		Name:       c.suite + "-" + goContainerName,
		User:       "gowaves",
		Hostname:   "go-node",
		Env: []string{
			"GRPC_ADDR=" + Localhost + ":" + GRPCAPIPort,
			"API_ADDR=" + Localhost + ":" + RESTAPIPort,
			"BIND_ADDR=" + Localhost + ":" + BindPort,
			"DECLARED_ADDR=" + "go-node:" + BindPort,
			"PEERS=",
			"WALLET_PASSWORD=itest",
			"DESIRED_REWARD=" + c.cfg.DesiredBlockRewardString(),
			"SUPPORTED_FEATURES=" + c.cfg.SupportedFeaturesString(),
		},
		ExposedPorts: []string{
			GRPCAPIPort + NetTCP,
			RESTAPIPort + NetTCP,
			BindPort + NetTCP,
		},
		Mounts: []string{
			c.configFolder + ":/home/gowaves/config",
			c.walletFolder + ":/home/gowaves/wallet",
		},
	}
	return opt
}

func (c *GoConfigurator) setAndVerifyWalletFolder() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	c.walletFolder = filepath.Clean(filepath.Join(pwd, walletPath))
	if _, flErr := os.Stat(c.walletFolder); os.IsNotExist(flErr) {
		return errors.New("wallet folder does not exist")
	}
	return nil
}

func (c *GoConfigurator) createNodeConfig() error {
	configDir, err := createConfigDir(c.suite)
	if err != nil {
		return errors.Wrap(err, "failed to create go node configuration")
	}
	configPath := filepath.Join(configDir, goConfigFilename)
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Sync(); err != nil {
			log.Printf("Failed to sync file '%s' to disk: %v", f.Name(), err)
			return
		}
		if err := f.Close(); err != nil {
			log.Printf("Failed to close file '%s': %v", f.Name(), err)
		}
	}()
	jsonWriter := json.NewEncoder(f)
	jsonWriter.SetIndent("", "\t")
	if jsErr := jsonWriter.Encode(c.cfg.Settings); jsErr != nil {
		return errors.Wrap(jsErr, "failed to encode genesis settings")
	}
	c.configFolder = configDir
	return nil
}

func createConfigDir(suiteName string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(pwd, tmpDir, suiteName)
	if mkErr := os.MkdirAll(configDir, os.ModePerm); mkErr != nil {
		return "", mkErr
	}
	return configDir, nil
}
