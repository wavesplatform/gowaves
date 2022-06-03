package docker

import (
	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

type Docker struct {
	pool      *dockertest.Pool
	goNode    *dockertest.Resource
	scalaNode *dockertest.Resource
}

func NewDocker() (Docker, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return Docker{}, err
	}
	return Docker{pool: pool}, nil
}

func (d *Docker) RunContainers() error {
	goNodeRes, err := d.runGoNode()
	if err != nil {
		return err
	}
	scalaNodeRes, err := d.runScalaNode()
	if err != nil {
		return err
	}
	d.goNode = goNodeRes
	d.scalaNode = scalaNodeRes
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

func (d *Docker) runGoNode() (*dockertest.Resource, error) {
	opt := &dockertest.RunOptions{
		Name: "gowaves-node",
		User: "gowaves",
		Env:  []string{"WAVES_NETWORK=stagenet"},
	}
	return d.pool.BuildAndRunWithOptions("/Users/ailin/Projects/gowaves/Dockerfile.gowaves-it", opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
}

func (d *Docker) runScalaNode() (*dockertest.Resource, error) {
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       "scala_node",
		Tag:        "latest",
		Env: []string{
			"WAVES_NETWORK=stagenet",
			"WAVES_WALLET_PASSWORD=myWalletSuperPassword",
		},
		Mounts: []string{
			"/Users/ailin/Projects/scala_node/data:/var/lib/waves",
			"/Users/ailin/Projects/scala_node/config:/etc/waves",
		},
	}
	return d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
}
