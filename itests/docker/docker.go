package docker

import (
	"os"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"

	"github.com/wavesplatform/gowaves/itests/config"
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
	if err := d.pool.Purge(d.goNode); err != nil {
		return err
	}
	if err := d.pool.Purge(d.scalaNode); err != nil {
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
		Name: "go-node",
		User: "gowaves",
		Env: []string{
			"WALLET_PASSWORD=test",
			"GRPS_ADDR=0.0.0.0:6871",
			"API_ADDR=0.0.0.0:6872",
			"PEERS=" + d.scalaNode.GetIPInNetwork(d.network) + ":6868",
		},
		PortBindings: map[dc.Port][]dc.PortBinding{
			"6871/tcp": {{HostPort: "6871"}},
			"6872/tcp": {{HostPort: "6872"}},
			"6873/tcp": {{HostPort: "6873"}},
		},
		Mounts: []string{
			cfgPath + ":/home/gowaves/config",
		},
	}
	res, err := d.pool.BuildAndRunWithOptions(pwd+"/../Dockerfile.gowaves-it", opt, func(hc *dc.HostConfig) {
		//hc.AutoRemove = true
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
			"6870/tcp": {{HostPort: "6870"}},
			"6869/tcp": {{HostPort: "6869"}},
			"6868/tcp": {{HostPort: "6868"}},
			"6873/tcp": {{HostPort: "6873"}},
		},
		Mounts: []string{
			cfgPath + ":/etc/waves",
		},
	}
	res, err := d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		//hc.AutoRemove = true
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
