package docker

import (
	"context"
	stderrs "errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/client"
)

const (
	DefaultTimeout       = 16 * time.Second
	DefaultAPIKey        = "itest-api-key"
	networkName          = "waves-it-network"
	goNodeLogFileName    = "go-node.log"
	goNodeErrFileName    = "go-node.err"
	scalaNodeLogFileName = "scala-node.log"
	scalaNodeErrFileName = "scala-node.err"
	logsDir              = "../build/logs"
)

type PortConfig struct {
	RESTAPIPort string
	GRPCPort    string
	BindPort    string
}

type NodeContainer struct {
	container *dockertest.Resource
	logs      *os.File
	errors    *os.File
	ports     *PortConfig
	network   *dockertest.Network
}

func (c *NodeContainer) RestAPIURL() string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(config.DefaultIP, c.ports.RESTAPIPort))
}

func (c *NodeContainer) Ports() *PortConfig {
	return c.ports
}

func (c *NodeContainer) ContainerNetworkIP() string {
	return c.container.GetIPInNetwork(c.network)
}

func (c *NodeContainer) closeFiles() error {
	var err error
	if c.logs != nil {
		if clErr := c.logs.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrapf(clErr, "failed to close logs file %q", c.logs.Name()))
		}
	}
	if c.errors != nil {
		if clErr := c.errors.Close(); clErr != nil {
			err = stderrs.Join(err, errors.Wrapf(clErr, "failed to close errors file %q", c.errors.Name()))
		}
	}
	return err
}

// Close purges container and closes log files.
func (c *NodeContainer) Close() error {
	if c.container == nil {
		return nil
	}
	if dcErr := c.container.DisconnectFromNetwork(c.network); dcErr != nil {
		return errors.Wrapf(dcErr, "failed to disconnect container %q from network %q",
			c.container.Container.ID, c.network.Network.Name)
	}
	if clErr := c.container.Close(); clErr != nil {
		return errors.Wrapf(clErr, "failed to close container %q", c.container.Container.ID)
	}
	if err := c.closeFiles(); err != nil {
		return err
	}
	return nil
}

type Docker struct {
	suite string

	pool    *dockertest.Pool
	network *dockertest.Network

	goNode    *NodeContainer
	scalaNode *NodeContainer

	logs string
}

// NewDocker creates a new Docker handler for a given suite name.
// It removes any existing containers or networks for the suite and creates a new suite network.
func NewDocker(suiteName string) (*Docker, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}
	docker := &Docker{suite: suiteName, pool: pool}
	if rmErr := docker.removeContainers(); rmErr != nil {
		return nil, rmErr
	}
	if rmErr := docker.removeNetworks(); rmErr != nil {
		return nil, rmErr
	}
	if netErr := docker.createNetwork(); netErr != nil {
		return nil, netErr
	}
	if mkErr := docker.mkLogsDir(); mkErr != nil {
		return nil, mkErr
	}
	return docker, nil
}

func (d *Docker) GoNode() *NodeContainer {
	return d.goNode
}

func (d *Docker) ScalaNode() *NodeContainer {
	return d.scalaNode
}

// StartGoNode starts a Go node container with the given configuration.
func (d *Docker) StartGoNode(ctx context.Context, cfg config.DockerConfigurator) error {
	var err error
	d.goNode, err = d.startNode(ctx, cfg, goNodeLogFileName, goNodeErrFileName)
	if err != nil {
		return errors.Wrap(err, "failed to start Go node")
	}
	return nil
}

// StartScalaNode starts a Scala node container with the given configuration.
func (d *Docker) StartScalaNode(ctx context.Context, cfg config.DockerConfigurator) error {
	var err error
	d.scalaNode, err = d.startNode(ctx, cfg, scalaNodeLogFileName, scalaNodeErrFileName)
	if err != nil {
		return errors.Wrap(err, "failed to start Scala node")
	}
	return nil
}

func (d *Docker) Finish(cancel context.CancelFunc) {
	if d.scalaNode != nil {
		if stErr := d.stopContainer(d.scalaNode.container.Container.ID); stErr != nil {
			log.Printf("Failed to stop Scala node container: %v", stErr)
		}
	}
	if d.goNode != nil {
		if stErr := d.stopContainer(d.goNode.container.Container.ID); stErr != nil {
			log.Printf("Failed to stop Go node container: %v", stErr)
		}
	}
	if d.scalaNode != nil {
		if clErr := d.scalaNode.Close(); clErr != nil {
			log.Printf("Failed to close scala-node: %s", clErr)
		}
	}
	if d.goNode != nil {
		if clErr := d.goNode.Close(); clErr != nil {
			log.Printf("Failed to close go-node: %s", clErr)
		}
	}
	if err := d.pool.RemoveNetwork(d.network); err != nil {
		log.Printf("Failed to remove docker network: %s", err)
	}
	cancel()
}

func (d *Docker) stopContainer(containerID string) error {
	const shutdownTimeout = 5 // In seconds.
	if stErr := d.pool.Client.StopContainer(containerID, shutdownTimeout); stErr != nil {
		if klErr := d.pool.Client.KillContainer(dc.KillContainerOptions{
			ID:     containerID,
			Signal: dc.SIGINT,
		}); klErr != nil {
			return errors.Wrapf(stderrs.Join(stErr, klErr), "failed to stop container %q", containerID)
		}
	}
	return nil
}

func (d *Docker) startNode(
	ctx context.Context, cfg config.DockerConfigurator, logFilename, errFilename string,
) (*NodeContainer, error) {
	opts := cfg.DockerRunOptions()
	opts.Networks = []*dockertest.Network{d.network}

	res, err := d.pool.RunWithOptions(opts, func(hc *dc.HostConfig) {
		hc.AutoRemove = false
		hc.PublishAllPorts = true
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start container for suite %q", d.suite)
	}

	logFile, err := os.Create(filepath.Clean(filepath.Join(d.logs, logFilename)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start container for suite %q", d.suite)
	}
	errFile, err := os.Create(filepath.Clean(filepath.Join(d.logs, errFilename)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start container for suite %q", d.suite)
	}

	go func() {
		err = d.pool.Client.Logs(dc.LogsOptions{
			Context:     ctx,
			Stderr:      true,
			Stdout:      true,
			Follow:      true,
			Timestamps:  false,
			RawTerminal: false,

			Container: res.Container.ID,

			OutputStream: logFile,
			ErrorStream:  errFile,
		})
		if err != nil && !errors.Is(err, context.Canceled) { // Do not log context.Canceled error.
			log.Printf("Failed to get logs from container %q: %v", res.Container.ID, err)
		}
	}()
	nc := &NodeContainer{
		container: res,
		logs:      logFile,
		errors:    errFile,
		ports: &PortConfig{
			RESTAPIPort: res.GetPort(config.RESTAPIPort + config.NetTCP),
			GRPCPort:    res.GetPort(config.GRPCAPIPort + config.NetTCP),
			BindPort:    res.GetPort(config.BindPort + config.NetTCP),
		},
		network: d.network,
	}

	err = d.pool.Retry(func() error {
		nodeClient, fErr := client.NewClient(client.Options{
			BaseUrl: nc.RestAPIURL(),
			Client:  &http.Client{Timeout: DefaultTimeout},
			ApiKey:  DefaultAPIKey,
		})
		if fErr != nil {
			return fErr
		}
		_, _, fErr = nodeClient.Blocks.Height(ctx)
		return fErr
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start container for suite %q", d.suite)
	}
	return nc, nil
}

func (d *Docker) removeContainers() error {
	err := d.pool.RemoveContainerByName(d.suite)
	if err != nil {
		return errors.Wrapf(err, "failed to remove existing containers for suite %s", d.suite)
	}
	return nil
}

func (d *Docker) removeNetworks() error {
	nets, err := d.pool.NetworksByName(d.suite + "-" + networkName)
	if err != nil {
		return errors.Wrapf(err, "failed to remove existing networks for suite %s", d.suite)
	}
	for i := 0; i < len(nets); i++ {
		err = d.pool.RemoveNetwork(&nets[i])
		if err != nil {
			return errors.Wrapf(err, "failed to remove existing networks for suite %s", d.suite)
		}
	}
	return nil
}

func (d *Docker) createNetwork() error {
	n, err := d.pool.CreateNetwork(d.suite + "-" + networkName)
	if err != nil {
		return errors.Wrapf(err, "failed to create network for suite %s", d.suite)
	}
	d.network = n
	return nil
}

func (d *Docker) mkLogsDir() error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "failed to create logs dir for suite %s", d.suite)
	}
	d.logs = filepath.Join(pwd, logsDir, d.suite)
	if mkErr := os.MkdirAll(d.logs, os.ModePerm); mkErr != nil {
		return errors.Wrapf(mkErr, "failed to create logs dir for suite %s", d.suite)
	}
	return nil
}
