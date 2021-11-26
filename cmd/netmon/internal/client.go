package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultScheme = "http"
)

var (
	versionRegex = regexp.MustCompile(`(?mi)^.*\sv(\d+\.\d+\.\d+.*)$`)
)

type versionResponse struct {
	Version string `json:"version"`
}

type nodeClient struct {
	cl  *client.Client
	url string
}

func newNodeClient(node string, timeout int) (*nodeClient, error) {
	var u *url.URL
	var err error
	if strings.Contains(node, "//") {
		u, err = url.Parse(node)
	} else {
		u, err = url.Parse("//" + node)
	}
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = defaultScheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme '%s'", u.Scheme)
	}
	t := time.Duration(timeout) * time.Second
	opts := client.Options{
		BaseUrl: u.String(),
		Client:  &http.Client{Timeout: t},
	}
	cl, err := client.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return &nodeClient{cl: cl, url: u.String()}, nil
}

func (c *nodeClient) version(ctx context.Context) (semver.Version, error) {
	versionRequest, err := http.NewRequest("GET", c.cl.GetOptions().BaseUrl+"/node/version", nil)
	if err != nil {
		return semver.Version{}, err
	}
	resp := new(versionResponse)
	_, err = c.cl.Do(ctx, versionRequest, resp)
	if err != nil {
		return semver.Version{}, err
	}
	sm := versionRegex.FindStringSubmatch(resp.Version)
	if len(sm) < 2 {
		return semver.Version{}, errors.New("no version found")
	}
	ver, err := semver.Parse(sm[1])
	if err != nil {
		return semver.Version{}, err
	}
	return ver, nil
}

func (c *nodeClient) height(ctx context.Context) (int, error) {
	height, _, err := c.cl.Blocks.Height(ctx)
	if err != nil {
		return 0, err
	}
	return int(height.Height), nil
}

func (c *nodeClient) stateHash(ctx context.Context, height int) (*proto.StateHash, error) {
	sh, _, err := c.cl.Debug.StateHash(ctx, uint64(height))
	if err != nil {
		return nil, err
	}
	return sh, nil
}

