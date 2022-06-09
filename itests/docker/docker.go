package docker

import (
	"os"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"

	"github.com/wavesplatform/gowaves/itests/config"
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
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	opt := &dockertest.RunOptions{
		Name: "go-node",
		User: "gowaves",
		Env:  []string{"WAVES_NETWORK=stagenet"},
	}
	return d.pool.BuildAndRunWithOptions(pwd+"/../Dockerfile.gowaves-it", opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
}

func (d *Docker) runScalaNode() (*dockertest.Resource, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       "scala_node",
		Tag:        "latest",
		PortBindings: map[dc.Port][]dc.PortBinding{
			"6970/tcp": {{HostPort: "6970"}},
			"6869/tcp": {{HostPort: "6869"}},
		},
		Mounts: []string{
			pwd + "/config:/etc/waves",
		},
	}

	err = config.CreateNewScalaConfig()
	if err != nil {
		return nil, err
	}
	return d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
	})
}
