package docker

import (
	"os"

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
)

type Docker struct {
	pool      *dockertest.Pool
	network   *dockertest.Network
	goNode    *dockertest.Resource
	scalaNode *dockertest.Resource
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
	scalaNodeRes, err := d.runScalaNode(paths.ScalaConfigPath)
	if err != nil {
		return err
	}
	d.scalaNode = scalaNodeRes
	goNodeRes, err := d.runGoNode(paths.GoConfigPath)
	if err != nil {
		return err
	}
	d.goNode = goNodeRes
	return nil
}

func (d *Docker) Purge() error {
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
	opt := &dockertest.RunOptions{
		Name: "go-node",
		User: "gowaves",
		Env: []string{
			"WALLET_PASSWORD=test",
			"GRPC_ADDR=" + Localhost + ":" + GoNodeGrpsApiPort,
			"API_ADDR=" + Localhost + ":" + GoNodeRESTApiPort,
			"PEERS=" + d.scalaNode.GetIPInNetwork(d.network) + ":" + ScalaNodeBindPort,
		},
		PortBindings: map[dc.Port][]dc.PortBinding{
			GoNodeGrpsApiPort + tcp: {{HostPort: GoNodeGrpsApiPort}},
			GoNodeRESTApiPort + tcp: {{HostPort: GoNodeRESTApiPort}},
			GoNodeBindPort + tcp:    {{HostPort: GoNodeBindPort}},
		},
		Mounts: []string{
			cfgPath + ":/home/gowaves/config",
		},
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	res, err := d.pool.BuildAndRunWithOptions(pwd+dockerfilePath, opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return nil, err
	}

	err = d.pool.Retry(func() error {
		_, err := GoNodeClient.GetHeight()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = res.ConnectToNetwork(d.network)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *Docker) runScalaNode(cfgPath string) (*dockertest.Resource, error) {
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       "scala_node",
		Tag:        "latest",
		PortBindings: map[dc.Port][]dc.PortBinding{
			ScalaNodeGrpsApiPort + tcp: {{HostPort: ScalaNodeGrpsApiPort}},
			ScalaNodeRESTApiPort + tcp: {{HostPort: ScalaNodeRESTApiPort}},
			ScalaNodeBindPort + tcp:    {{HostPort: ScalaNodeBindPort}},
		},
		Mounts: []string{
			cfgPath + ":/etc/waves",
		},
	}
	res, err := d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return nil, err
	}

	err = d.pool.Retry(func() error {
		_, err := ScalaNodeClient.GetHeight()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = res.ConnectToNetwork(d.network)
	if err != nil {
		return nil, err
	}
	return res, nil
}
