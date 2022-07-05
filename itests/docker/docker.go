package docker

import (
	"os"
	"path/filepath"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"

	"github.com/wavesplatform/gowaves/itests/api"
	"github.com/wavesplatform/gowaves/itests/config"
)

var (
	GoNodeClient    = api.NewNodeClient("http://"+Localhost+":"+GoNodeRESTApiPort+"/", api.DefaultTimeout)
	ScalaNodeClient = api.NewNodeClient("http://"+Localhost+":"+ScalaNodeRESTApiPort+"/", api.DefaultTimeout)
)

const (
	Localhost = "0.0.0.0"

	GoNodeRESTApiPort = "6872"
	GoNodeGrpsApiPort = "6871"
	GoNodeBindPort    = "6873"

	ScalaNodeRESTApiPort = "6869"
	ScalaNodeGrpsApiPort = "6870"
	ScalaNodeBindPort    = "6868"

	tcp = "/tcp"
)

const (
	dockerfilePath = "/../Dockerfile.gowaves-it"

	goNodeLogFileName    = "go-node.log"
	scalaNodeLogFileName = "scala-node.log"
	logDir               = "../build/logs"

	walletPath = "wallet"
)

type Docker struct {
	pool         *dockertest.Pool
	network      *dockertest.Network
	goNode       *dockertest.Resource
	goLogFile    *os.File
	scalaNode    *dockertest.Resource
	scalaLogFile *os.File
}

func NewDocker() (Docker, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return Docker{}, err
	}
	net, err := pool.CreateNetwork("waves_it_network")
	if err != nil {
		return Docker{}, err
	}
	return Docker{pool: pool, network: net}, nil
}

func (d *Docker) RunContainers(paths config.ConfigPaths) error {
	goNodeRes, err := d.runGoNode(paths.GoConfigPath)
	if err != nil {
		return err
	}
	d.goNode = goNodeRes
	scalaNodeRes, err := d.runScalaNode(paths.ScalaConfigPath)
	if err != nil {
		return err
	}
	d.scalaNode = scalaNodeRes
	return nil
}

func (d *Docker) Purge() error {
	defer func() {
		_ = d.goLogFile.Close()
		_ = d.scalaLogFile.Close()
	}()
	if err := d.pool.Purge(d.scalaNode); err != nil {
		return err
	}
	if err := d.pool.Purge(d.goNode); err != nil {
		return err
	}
	if err := d.pool.RemoveNetwork(d.network); err != nil {
		return err
	}
	return nil
}

func (d *Docker) runGoNode(cfgPath string) (*dockertest.Resource, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	opt := &dockertest.RunOptions{
		Name:     "go-node",
		User:     "gowaves",
		Hostname: "go-node",
		Env: []string{
			"WALLET_PASSWORD=test",
			"GRPC_ADDR=" + Localhost + ":" + GoNodeGrpsApiPort,
			"API_ADDR=" + Localhost + ":" + GoNodeRESTApiPort,
			"PEERS=",
			"WALLET_PASSWORD=itest",
		},
		PortBindings: map[dc.Port][]dc.PortBinding{
			GoNodeGrpsApiPort + tcp: {{HostIP: "localhost", HostPort: GoNodeGrpsApiPort}},
			GoNodeRESTApiPort + tcp: {{HostIP: "localhost", HostPort: GoNodeRESTApiPort}},
			GoNodeBindPort + tcp:    {{HostIP: "localhost", HostPort: GoNodeBindPort}},
		},
		Mounts: []string{
			cfgPath + ":/home/gowaves/config",
			filepath.Clean(filepath.Join(pwd, walletPath)) + ":/home/gowaves/wallet",
		},
		Networks: []*dockertest.Network{d.network},
	}
	res, err := d.pool.BuildAndRunWithOptions(pwd+dockerfilePath, opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return nil, err
	}

	logfile, err := os.Create(filepath.Clean(filepath.Join(pwd, logDir, goNodeLogFileName)))
	if err != nil {
		return nil, err
	}

	go func() {
		_ = d.pool.Client.Logs(dc.LogsOptions{
			Stderr:      true,
			Stdout:      true,
			Follow:      true,
			Timestamps:  false,
			RawTerminal: false,

			Container: res.Container.ID,

			OutputStream: logfile,
		})
	}()
	d.goLogFile = logfile
	err = d.pool.Retry(func() error {
		_, err := GoNodeClient.GetBlocksHeight()
		return err
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *Docker) runScalaNode(cfgPath string) (*dockertest.Resource, error) {
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       "scala-node",
		Tag:        "latest",
		Hostname:   "scala-node",
		PortBindings: map[dc.Port][]dc.PortBinding{
			ScalaNodeGrpsApiPort + tcp: {{HostIP: "localhost", HostPort: ScalaNodeGrpsApiPort}},
			ScalaNodeRESTApiPort + tcp: {{HostIP: "localhost", HostPort: ScalaNodeRESTApiPort}},
			ScalaNodeBindPort + tcp:    {{HostIP: "localhost", HostPort: ScalaNodeBindPort}},
		},
		Mounts: []string{
			cfgPath + ":/etc/waves",
		},
		Env: []string{
			"WAVES_LOG_LEVEL=TRACE",
			"WAVES_NETWORK=custom",
			"JAVA_OPTS=" +
				"-Dwaves.network.known-peers.0=" + d.goNode.GetIPInNetwork(d.network) + ":" + GoNodeBindPort,
		},
		Networks: []*dockertest.Network{d.network},
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	res, err := d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return nil, err
	}

	logfile, err := os.Create(filepath.Clean(filepath.Join(pwd, logDir, scalaNodeLogFileName)))
	if err != nil {
		return nil, err
	}

	go func() {
		_ = d.pool.Client.Logs(dc.LogsOptions{
			Stderr:      true,
			Stdout:      true,
			Follow:      true,
			Timestamps:  false,
			RawTerminal: false,

			Container: res.Container.ID,

			OutputStream: logfile,
		})
	}()
	d.scalaLogFile = logfile

	err = d.pool.Retry(func() error {
		_, err := ScalaNodeClient.GetBlocksHeight()
		return err
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
