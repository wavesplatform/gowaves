package docker

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"
	"github.com/xenolf/lego/log"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/client"
)

const (
	Localhost = "0.0.0.0"

	tcp = "/tcp"

	DefaultTimeout = 16 * time.Second

	scalaContainerName = "scala-node"
	goContainerName    = "go-node"
	networkName        = "waves-it-network"
)

const (
	RESTApiPort = "6869"
	GrpcApiPort = "6870"
	BindPort    = "6868"
)

const (
	dockerfilePath = "/../Dockerfile.gowaves-it"

	goNodeLogFileName    = "go-node.log"
	scalaNodeLogFileName = "scala-node.log"
	logDir               = "../build/logs"

	walletPath = "wallet"
)

type Ports struct {
	Go    *PortConfig
	Scala *PortConfig
}

type PortConfig struct {
	RestApiPort string
	GrpcPort    string
	BindPort    string
}

type Docker struct {
	pool         *dockertest.Pool
	network      *dockertest.Network
	goNode       *dockertest.Resource
	goLogFile    *os.File
	scalaNode    *dockertest.Resource
	scalaLogFile *os.File
}

func NewDocker(suiteName string) (*Docker, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}
	err = removeExistsContainers(pool, suiteName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to remove old containers")
	}
	net, err := pool.CreateNetwork(suiteName + "-" + networkName)
	if err != nil {
		return nil, err
	}
	return &Docker{pool: pool, network: net}, nil
}

func removeExistsContainers(pool *dockertest.Pool, suiteName string) error {
	res, exist := pool.ContainerByName(suiteName + "-" + goContainerName)
	if exist {
		err := pool.Purge(res)
		if err != nil {
			return err
		}
	}
	res, exist = pool.ContainerByName(suiteName + "-" + scalaContainerName)
	if exist {
		err := pool.Purge(res)
		if err != nil {
			return err
		}
	}
	net, err := pool.NetworksByName(suiteName + "-" + networkName)
	if err != nil {
		return err
	}
	for i := 0; i < len(net); i++ {
		err = pool.RemoveNetwork(&net[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Docker) RunContainers(ctx context.Context, paths config.ConfigPaths, suiteName string) (*Ports, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filepath.Join(pwd, logDir, suiteName), os.ModePerm)
	if err != nil {
		return nil, err
	}

	goNodeRes, goPorts, err := d.runGoNode(ctx, paths.GoConfigPath, suiteName)
	if err != nil {
		return nil, err
	}
	d.goNode = goNodeRes
	scalaNodeRes, scalaPorts, err := d.runScalaNode(ctx, paths.ScalaConfigPath, suiteName)
	if err != nil {
		return nil, err
	}
	d.scalaNode = scalaNodeRes

	return &Ports{
		Go:    goPorts,
		Scala: scalaPorts,
	}, nil
}

func (d *Docker) Finish(cancel context.CancelFunc) {
	cancel()
	if d.scalaNode != nil {
		if err := d.pool.Purge(d.scalaNode); err != nil {
			log.Warnf("Failed to purge scala-node: %s", err)
		}
	}
	if d.goNode != nil {
		if err := d.pool.Purge(d.goNode); err != nil {
			log.Warnf("Failed to purge go-node: %s", err)

		}
	}
	if err := d.pool.RemoveNetwork(d.network); err != nil {
		log.Warnf("Failed to remove docker network: %s", err)
	}
	if d.goLogFile != nil {
		if err := d.goLogFile.Close(); err != nil {
			log.Warnf("Failed to close go-node logs file: %s", err)
		}
	}
	if d.scalaLogFile != nil {
		if err := d.scalaLogFile.Close(); err != nil {
			log.Warnf("Failed to close scala-node logs file: %s", err)
		}
	}
}

func (d *Docker) buildGoNodeImage() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, file := filepath.Split(pwd + dockerfilePath)
	err = d.pool.Client.BuildImage(dc.BuildImageOptions{
		Name:         "go-node",
		Dockerfile:   file,
		ContextDir:   dir,
		OutputStream: io.Discard,
		BuildArgs:    nil,
		Platform:     "",
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *Docker) runGoNode(ctx context.Context, cfgPath string, suiteName string) (*dockertest.Resource, *PortConfig, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	err = d.buildGoNodeImage()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to build go-node image")
	}
	opt := &dockertest.RunOptions{
		Repository: "go-node",
		Name:       suiteName + "-" + goContainerName,
		User:       "gowaves",
		Hostname:   "go-node",
		Env: []string{
			"GRPC_ADDR=" + Localhost + ":" + GrpcApiPort,
			"API_ADDR=" + Localhost + ":" + RESTApiPort,
			"BIND_ADDR=" + Localhost + ":" + BindPort,
			"DECLARED_ADDR=" + "go-node:" + BindPort,
			"PEERS=",
			"WALLET_PASSWORD=itest",
		},
		ExposedPorts: []string{
			GrpcApiPort,
			RESTApiPort,
			BindPort,
		},
		Mounts: []string{
			cfgPath + ":/home/gowaves/config",
			filepath.Clean(filepath.Join(pwd, walletPath)) + ":/home/gowaves/wallet",
		},
		Networks: []*dockertest.Network{d.network},
	}
	res, err := d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
		hc.PublishAllPorts = true
	})
	if err != nil {
		return nil, nil, err
	}

	logfile, err := os.Create(filepath.Clean(filepath.Join(pwd, logDir, suiteName, goNodeLogFileName)))
	if err != nil {
		return nil, nil, err
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

			OutputStream: logfile,
		})
		if err != nil {
			log.Warnf("Fail to get logs from go-node: %s", err)
		}
	}()
	d.goLogFile = logfile

	portCfg := &PortConfig{
		RestApiPort: res.GetPort(RESTApiPort + tcp),
		GrpcPort:    res.GetPort(GrpcApiPort + tcp),
		BindPort:    res.GetPort(BindPort + tcp),
	}

	err = d.pool.Retry(func() error {
		nodeClient, err := client.NewClient(client.Options{
			BaseUrl: "http://" + Localhost + ":" + portCfg.RestApiPort + "/",
			Client:  &http.Client{Timeout: DefaultTimeout},
			ApiKey:  "itest-api-key",
		})
		if err != nil {
			return err
		}
		_, _, err = nodeClient.Blocks.Height(ctx)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return res, portCfg, nil
}

func (d *Docker) runScalaNode(ctx context.Context, cfgPath string, suiteName string) (*dockertest.Resource, *PortConfig, error) {
	opt := &dockertest.RunOptions{
		Repository: "wavesplatform/wavesnode",
		Name:       suiteName + "-" + scalaContainerName,
		Tag:        "latest",
		Hostname:   "scala-node",
		Mounts: []string{
			cfgPath + ":/etc/waves",
		},
		Env: []string{
			"WAVES_LOG_LEVEL=TRACE",
			"WAVES_NETWORK=custom",
			"JAVA_OPTS=" +
				"-Dwaves.network.known-peers.0=" + d.goNode.GetIPInNetwork(d.network) + ":" + BindPort + " " +
				"-Dwaves.network.declared-address=scala-node:" + BindPort + " " +
				"-Dwaves.network.port=" + BindPort + " " +
				"-Dwaves.rest-api.port=" + RESTApiPort + " " +
				"-Dwaves.grpc.port=" + GrpcApiPort + " " +
				"-Dwaves.network.enable-blacklisting=no",
		},
		ExposedPorts: []string{
			GrpcApiPort,
			RESTApiPort,
			BindPort,
		},
		Networks: []*dockertest.Network{d.network},
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	res, err := d.pool.RunWithOptions(opt, func(hc *dc.HostConfig) {
		hc.AutoRemove = true
		hc.PublishAllPorts = true
	})
	if err != nil {
		return nil, nil, err
	}

	logfile, err := os.Create(filepath.Clean(filepath.Join(pwd, logDir, suiteName, scalaNodeLogFileName)))
	if err != nil {
		return nil, nil, err
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

			OutputStream: logfile,
		})
		if err != nil {
			log.Warnf("Fail to get logs from scala-node: %s", err)
		}
	}()
	d.scalaLogFile = logfile

	portCfg := &PortConfig{
		RestApiPort: res.GetPort(RESTApiPort + tcp),
		GrpcPort:    res.GetPort(GrpcApiPort + tcp),
		BindPort:    res.GetPort(BindPort + tcp),
	}

	err = d.pool.Retry(func() error {
		nodeClient, err := client.NewClient(client.Options{
			BaseUrl: "http://" + Localhost + ":" + portCfg.RestApiPort + "/",
			Client:  &http.Client{Timeout: DefaultTimeout},
			ApiKey:  "itest-api-key",
		})
		if err != nil {
			return err
		}
		_, _, err = nodeClient.Blocks.Height(ctx)
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	return res, portCfg, nil
}
