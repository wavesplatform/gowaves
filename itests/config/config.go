package config

import (
	"encoding/binary"
	"encoding/json"
	stderrs "errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/ory/dockertest/v4"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

	ScalaImageRepository = "wavesplatform/wavesnode"
	ScalaImageTag        = "1.6.5"
	goImageRepository    = "go-node"
)

const (
	RESTAPIPort = "6869"
	GRPCAPIPort = "6870"
	BindPort    = "6868"
	DefaultIP   = "0.0.0.0"
	NetTCP      = "/tcp"
)

type TestConfig struct {
	Accounts           []AccountInfo
	BlockchainSettings *settings.BlockchainSettings
}

func (c *TestConfig) GetRichestAccount() AccountInfo {
	r := c.Accounts[0]
	for _, a := range c.Accounts {
		if a.Amount > r.Amount {
			r = a
		}
	}
	return r
}

func (c *TestConfig) GetAccount(address string) (AccountInfo, error) {
	for _, a := range c.Accounts {
		if a.Address.String() == address {
			return a, nil
		}
	}
	return AccountInfo{}, fmt.Errorf("account with address '%s' is not found", address)
}

func (c *TestConfig) GenesisSH() crypto.Digest {
	const uint64Size = 8

	hash, err := crypto.NewFastHash()
	if err != nil {
		panic(err)
	}
	var emptyDigest crypto.Digest
	hash.Sum(emptyDigest[:0])

	// Create binary entries for all genesis transactions.
	prevSH := emptyDigest
	for _, a := range c.Accounts {
		hash.Reset()
		buf := make([]byte, proto.WavesAddressSize+uint64Size)
		copy(buf, a.Address[:])
		binary.BigEndian.PutUint64(buf[proto.WavesAddressSize:], a.Amount)
		hash.Write(buf)
		var txSH crypto.Digest
		hash.Sum(txSH[:0])

		hash.Reset()
		hash.Write(prevSH[:])
		hash.Write(txSH[:])

		var newSH crypto.Digest
		hash.Sum(newSH[:0])
		prevSH = newSH
	}
	return prevSH
}

type DockerConfigurator interface {
	// DockerRunOptions returns the image repository and the options to run a container from it.
	DockerRunOptions() (string, []dockertest.RunOption)
	WithImageRepository(repository string) DockerConfigurator
	WithImageTag(tag string) DockerConfigurator
}

type ScalaConfigurator struct {
	suite           string
	cfg             *BlockchainConfig
	configFolder    string
	knownPeers      []string
	imageRepository string
	imageTag        string
}

func NewScalaConfigurator(suite string, cfg *BlockchainConfig) (*ScalaConfigurator, error) {
	c := &ScalaConfigurator{suite: suite, cfg: cfg}
	if err := c.createNodeConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to create scala node configuration")
	}
	return c, nil
}

func (c *ScalaConfigurator) WithImageRepository(repository string) DockerConfigurator {
	c.imageRepository = repository
	return c
}

func (c *ScalaConfigurator) WithImageTag(tag string) DockerConfigurator {
	c.imageTag = tag
	return c
}

func (c *ScalaConfigurator) WithGoNode(goNodeIP string) *ScalaConfigurator {
	c.knownPeers = append(c.knownPeers, goNodeIP)
	return c
}

func (c *ScalaConfigurator) DockerRunOptions() (string, []dockertest.RunOption) {
	if c.imageRepository == "" {
		c.imageRepository = ScalaImageRepository
	}

	if c.imageTag == "" {
		c.imageTag = ScalaImageTag
	}

	kps := new(strings.Builder)
	for i, kp := range c.knownPeers {
		if _, err := fmt.Fprintf(kps, "-Dwaves.network.known-peers.%d=%s:%s ", i, kp, BindPort); err != nil {
			panic(err)
		}
	}
	opts := []dockertest.RunOption{
		dockertest.WithName(c.suite + "-" + scalaContainerName),
		dockertest.WithTag(c.imageTag),
		dockertest.WithHostname("scala-node"),
		dockertest.WithMounts([]string{
			c.configFolder + ":/etc/waves",
		}),
		dockertest.WithEnv([]string{
			"WAVES_LOG_LEVEL=TRACE",
			"WAVES_NETWORK=custom",
			"JAVA_OPTS=" +
				kps.String() +
				"-Dwaves.network.declared-address=scala-node:" + BindPort + " " +
				"-Dwaves.network.port=" + BindPort + " " +
				"-Dwaves.rest-api.port=" + RESTAPIPort + " " +
				"-Dwaves.grpc.port=" + GRPCAPIPort + " " +
				"-Dwaves.network.enable-blacklisting=no",
		}),
		withExposedPorts(),
		dockertest.WithoutReuse(),
	}
	return c.imageRepository, opts
}

func (c *ScalaConfigurator) createNodeConfig() (err error) {
	var configDir string
	configDir, err = createConfigDir(c.suite)
	if err != nil {
		return errors.Wrap(err, "failed to create scala node configuration")
	}
	configPath := filepath.Join(configDir, scalaConfigFilename)
	var f *os.File
	f, err = os.Create(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to create scala node configuration")
	}
	defer func() {
		if synErr := f.Sync(); synErr != nil {
			err = stderrs.Join(err, errors.Wrapf(synErr, "failed to sync file %q to disk", f.Name()))
		}
		if clErr := f.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrapf(clErr, "failed to close file %q", f.Name()))
		}
	}()
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "failed to create scala node configuration")
	}
	templatePath := filepath.Join(pwd, configFolder, templateScalaCfgFilename)
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return errors.Wrap(err, "failed to create scala node configuration")
	}
	if exErr := t.Execute(f, c.cfg); exErr != nil {
		return errors.Wrap(exErr, "failed to create scala node configuration")
	}
	c.configFolder = configDir
	return err
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

func (c *GoConfigurator) WithImageRepository(_ string) DockerConfigurator {
	return c
}

func (c *GoConfigurator) WithImageTag(_ string) DockerConfigurator {
	return c
}

func (c *GoConfigurator) DockerRunOptions() (string, []dockertest.RunOption) {
	opts := []dockertest.RunOption{
		dockertest.WithName(c.suite + "-" + goContainerName),
		dockertest.WithTag("latest"),
		dockertest.WithUser("gowaves"),
		dockertest.WithHostname("go-node"),
		dockertest.WithEnv([]string{
			"GRPC_ADDR=" + DefaultIP + ":" + GRPCAPIPort,
			"API_ADDR=" + DefaultIP + ":" + RESTAPIPort,
			"BIND_ADDR=" + DefaultIP + ":" + BindPort,
			"DECLARED_ADDR=" + "go-node:" + BindPort,
			"PEERS=",
			"WALLET_PASSWORD=itest",
			"DESIRED_REWARD=" + c.cfg.DesiredBlockRewardString(),
			"SUPPORTED_FEATURES=" + c.cfg.SupportedFeaturesString(),
			"QUORUM=" + c.cfg.QuorumString(),
			"DISABLE_MINER=" + c.cfg.DisableGoMiningString(),
		}),
		withExposedPorts(),
		dockertest.WithMounts([]string{
			c.configFolder + ":/home/gowaves/config",
			c.walletFolder + ":/home/gowaves/wallet",
		}),
		dockertest.WithoutReuse(),
	}
	return goImageRepository, opts
}

// withExposedPorts exposes node's gRPC, REST API and bind ports.
// Dockertest v4 has no dedicated option for exposed ports, so they are set directly in container config.
func withExposedPorts() dockertest.RunOption {
	return dockertest.WithContainerConfig(func(cfg *container.Config) {
		cfg.ExposedPorts = network.PortSet{
			network.MustParsePort(GRPCAPIPort + NetTCP): {},
			network.MustParsePort(RESTAPIPort + NetTCP): {},
			network.MustParsePort(BindPort + NetTCP):    {},
		}
	})
}

func (c *GoConfigurator) setAndVerifyWalletFolder() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	c.walletFolder = filepath.Clean(filepath.Join(pwd, walletPath))
	if _, flErr := os.Stat(c.walletFolder); flErr != nil {
		if errors.Is(flErr, os.ErrNotExist) {
			return errors.New("wallet folder does not exist")
		}
		return errors.Wrap(err, "unexpected error while verifying wallet folder")
	}
	return nil
}

func (c *GoConfigurator) createNodeConfig() (err error) {
	var configDir string
	configDir, err = createConfigDir(c.suite)
	if err != nil {
		return errors.Wrap(err, "failed to create go node configuration")
	}
	configPath := filepath.Join(configDir, goConfigFilename)
	var f *os.File
	f, err = os.Create(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to create go node configuration")
	}
	defer func() {
		if synErr := f.Sync(); synErr != nil {
			err = stderrs.Join(err, errors.Wrapf(synErr, "failed to sync file %q to disk", f.Name()))
		}
		if clErr := f.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrapf(clErr, "failed to close file %q", f.Name()))
		}
	}()
	jsonWriter := json.NewEncoder(f)
	jsonWriter.SetIndent("", "\t")
	if jsErr := jsonWriter.Encode(c.cfg.Settings); jsErr != nil {
		return errors.Wrap(jsErr, "failed to create go node configuration")
	}
	c.configFolder = configDir
	return err
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

func Platform() ocispec.Platform {
	return ocispec.Platform{OS: "linux", Architecture: runtime.GOARCH}
}
