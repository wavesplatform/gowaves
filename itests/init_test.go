package itests

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

const dockerfilePath = "./../Dockerfile.gowaves-it"
const keepDanglingEnvKey = "ITESTS_KEEP_DANGLING"

func TestMain(m *testing.M) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get pwd: %v", err)
	}
	keepDangling := mustBoolEnv(keepDanglingEnvKey)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Failed to create docker pool: %v", err)
	}
	if err := pool.Client.PullImage(dc.PullImageOptions{Repository: "wavesplatform/wavesnode", Tag: "latest"}, dc.AuthConfiguration{}); err != nil {
		log.Fatalf("Failed to pull node image: %v", err)
	}
	dir, file := filepath.Split(filepath.Join(pwd, dockerfilePath))
	err = pool.Client.BuildImage(dc.BuildImageOptions{
		Name:           "go-node",
		Dockerfile:     file,
		ContextDir:     dir,
		OutputStream:   io.Discard,
		BuildArgs:      nil,
		Platform:       "",
		RmTmpContainer: true,
	})
	if err != nil {
		log.Fatalf("Failed to create go-node image: %v", err)
	}

	if !keepDangling { // remove dangling images
		imgs, err := pool.Client.ListImages(dc.ListImagesOptions{
			Filters: map[string][]string{
				"label": {"wavesplatform-gowaves-itests-tmp=true"},
			},
		})
		if err != nil {
			log.Fatalf("Failed to get list of images from docker: %v", err)
		}
		for _, i := range imgs {
			err = pool.Client.RemoveImage(i.ID)
			if err != nil {
				log.Fatalf("Failed to remove dangling images: %v", err)
			}
		}
	}
	os.Exit(m.Run())
}

func mustBoolEnv(key string) bool {
	envFlag := os.Getenv(key)
	if envFlag == "" {
		return false
	}
	r, err := strconv.ParseBool(envFlag)
	if err != nil {
		log.Fatalf("Invalid flag value %q for the env key %q: %v", envFlag, key, err)
	}
	return r
}
