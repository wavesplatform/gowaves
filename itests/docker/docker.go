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

	"github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
	"github.com/ory/dockertest/v4"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/client"
)

const (
	DefaultTimeout   = 16 * time.Second
	PoolRetryTimeout = 2 * time.Minute

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
	container dockertest.ClosableResource
	logs      *os.File
	errors    *os.File
	ports     *PortConfig
	network   dockertest.Network
	ip        string // IP address of the container in the network.
}

func (c *NodeContainer) RestAPIURL() string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(config.DefaultIP, c.ports.RESTAPIPort))
}

// IP returns the IP address of the container in the Docker network.
// This is the address that can be used to connect to the container from other containers in the same network.
// This address should be used only for PeerInfo creation, not for REST API or gRPC connections.
func (c *NodeContainer) IP() string {
	return c.ip
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
func (c *NodeContainer) Close(ctx context.Context) error {
	if c.container == nil {
		return nil
	}
	// Close removes the container. If it is not stopped, it will be killed with SIGKILL.
	// Removed container is disconnected from the network automatically.
	if clErr := c.container.Close(ctx); clErr != nil {
		return errors.Wrapf(clErr, "failed to close container %q", c.container.ID())
	}
	if err := c.closeFiles(); err != nil {
		return err
	}
	return nil
}

type Docker struct {
	suite string

	client  *mobyclient.Client
	pool    dockertest.ClosablePool
	network dockertest.ClosableNetwork

	goNode    *NodeContainer
	scalaNode *NodeContainer

	logs string
}

// NewDocker creates a new Docker handler for a given suite name.
// It removes any existing containers or networks for the suite and creates a new suite network.
func NewDocker(ctx context.Context, suiteName string) (*Docker, error) {
	cl, err := NewClient(ctx)
	if err != nil {
		return nil, err
	}
	// The pool doesn't close a client provided with WithMobyClient option, it's closed in Finish.
	pool, err := dockertest.NewPool(ctx, "", dockertest.WithMobyClient(cl), dockertest.WithMaxWait(PoolRetryTimeout))
	if err != nil {
		return nil, stderrs.Join(err, cl.Close())
	}
	docker := &Docker{suite: suiteName, client: cl, pool: pool}
	if rmErr := docker.removeContainers(ctx); rmErr != nil {
		return nil, rmErr
	}
	if rmErr := docker.removeNetworks(ctx); rmErr != nil {
		return nil, rmErr
	}
	if netErr := docker.createNetwork(ctx); netErr != nil {
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

// StartNodes start both Go and Scala nodes with the given configurations.
// Note that while starting nodes in parallel it is impossible to retrieve the IP address of the other node in prior.
// So this method is heavily dependent on Docker DNS resolution and Go-node's domain name should be passed to the
// configuration of Scala node before calling this method:
//
//	scalaConfigurator.WithGoNode("go-node")
func (d *Docker) StartNodes(ctx context.Context, goCfg, scalaCfg config.DockerConfigurator) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		return d.StartGoNode(ctx, goCfg)
	})
	eg.Go(func() error {
		return d.StartScalaNode(ctx, scalaCfg)
	})
	return eg.Wait()
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

// Finish stops and removes containers and the suite network, then cancels the suite context with the given function.
// Cleanup is not interrupted if the given context is already canceled, but the context's values are preserved.
func (d *Docker) Finish(ctx context.Context, cancel context.CancelFunc) {
	ctx = context.WithoutCancel(ctx)
	eg := errgroup.Group{}
	if d.scalaNode != nil {
		eg.Go(func() error {
			stErr := d.stopContainer(ctx, d.scalaNode.container.ID())
			clErr := d.scalaNode.Close(ctx)
			return stderrs.Join(stErr, clErr)
		})
	}
	if d.goNode != nil {
		eg.Go(func() error {
			stErr := d.stopContainer(ctx, d.goNode.container.ID())
			clErr := d.goNode.Close(ctx)
			return stderrs.Join(stErr, clErr)
		})
	}
	if err := eg.Wait(); err != nil {
		log.Printf("[ERR] Failed to shutdown docker containers: %v", err)
	}
	if err := d.network.Close(ctx); err != nil {
		log.Printf("[ERR] Failed to remove docker network: %s", err)
	}
	if err := d.pool.Close(ctx); err != nil {
		log.Printf("[ERR] Failed to close docker pool: %s", err)
	}
	if err := d.client.Close(); err != nil {
		log.Printf("[ERR] Failed to close docker client: %s", err)
	}
	cancel()
}

func (d *Docker) stopContainer(ctx context.Context, containerID string) error {
	const shutdownTimeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()
	stopOpts := mobyclient.ContainerKillOptions{Signal: "SIGINT"}
	if _, intErr := d.client.ContainerKill(ctx, containerID, stopOpts); intErr != nil {
		return errors.Wrapf(intErr, "failed to interrupt container %q", containerID)
	}
	wait := d.client.ContainerWait(ctx, containerID,
		mobyclient.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	select {
	case <-wait.Result:
		return nil
	case wErr := <-wait.Error:
		return errors.Wrapf(wErr, "failed to wait for container %q to stop", containerID)
	}
}

func (d *Docker) startNode(
	ctx context.Context, cfg config.DockerConfigurator, logFilename, errFilename string,
) (*NodeContainer, error) {
	repository, opts := cfg.DockerRunOptions()
	opts = append(opts, dockertest.WithHostConfig(func(hc *container.HostConfig) {
		hc.AutoRemove = false
		hc.PublishAllPorts = true
		// Connect the container to the suite network on creation, so it is resolvable by other containers from start.
		hc.NetworkMode = container.NetworkMode(d.network.Inspect().Name)
	}))

	res, err := d.pool.Run(ctx, repository, opts...)
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
		if lErr := res.FollowLogs(ctx, logFile, errFile); lErr != nil &&
			!errors.Is(lErr, context.Canceled) { // Do not log context.Canceled error.
			log.Printf("Failed to get logs from container %q: %v", res.ID(), lErr)
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
		ip:      res.GetIPInNetwork(d.network),
	}

	containerName := res.Container().Name
	err = d.pool.Retry(ctx, 0, func() error { // Zero timeout means pool's MaxWait.
		nodeClient, fErr := client.NewClient(client.Options{
			BaseUrl: nc.RestAPIURL(),
			Client:  &http.Client{Timeout: DefaultTimeout},
			ApiKey:  DefaultAPIKey,
		})
		if fErr != nil {
			log.Printf("Failed to create client for container %q: %v", containerName, fErr)
			return fErr
		}
		_, _, fErr = nodeClient.Blocks.Height(ctx)
		log.Printf("Result requesting height from container %q: %v", containerName, fErr)
		return fErr
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start container for suite %q", d.suite)
	}
	return nc, nil
}

func (d *Docker) removeContainers(ctx context.Context) error {
	containers, err := d.client.ContainerList(ctx, mobyclient.ContainerListOptions{
		All:     true,
		Filters: make(mobyclient.Filters).Add("name", d.suite),
	})
	if err != nil {
		return fmt.Errorf("failed to list suite %q containers: %w", d.suite, err)
	}
	for _, c := range containers.Items {
		_, err = d.client.ContainerRemove(ctx, c.ID, mobyclient.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
		if err != nil {
			return fmt.Errorf("failed to remove container %q of suite %q: %w", c.ID, d.suite, err)
		}
	}
	return nil
}

func (d *Docker) removeNetworks(ctx context.Context) error {
	name := d.suite + "-" + networkName
	nets, err := d.client.NetworkList(ctx, mobyclient.NetworkListOptions{
		Filters: make(mobyclient.Filters).Add("name", name),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to remove existing networks for suite %s", d.suite)
	}
	for _, n := range nets.Items {
		if n.Name != name { // Name filter matches substrings, so check for exact match.
			continue
		}
		_, err = d.client.NetworkRemove(ctx, n.ID, mobyclient.NetworkRemoveOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to remove existing networks for suite %s", d.suite)
		}
	}
	return nil
}

func (d *Docker) createNetwork(ctx context.Context) error {
	// IPv6 is disabled by default.
	n, err := d.pool.CreateNetwork(ctx, d.suite+"-"+networkName, nil)
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

// NewClient creates a Docker client configured from the environment variables (DOCKER_HOST, DOCKER_API_VERSION,
// DOCKER_CERT_PATH, DOCKER_TLS_VERIFY). If DOCKER_API_VERSION is not set, the API version is negotiated with
// the Docker daemon.
func NewClient(ctx context.Context) (*mobyclient.Client, error) {
	cl, err := mobyclient.New(mobyclient.FromEnv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}
	if _, pErr := cl.Ping(ctx, mobyclient.PingOptions{NegotiateAPIVersion: true}); pErr != nil {
		return nil, stderrs.Join(errors.Wrap(pErr, "failed to connect to docker"), cl.Close())
	}
	return cl, nil
}
