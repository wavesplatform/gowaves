package itests

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

const dockerfilePath = "/../Dockerfile.gowaves-it"

func TestMain(m *testing.M) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("%s", err)
	}
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Failed to create docker pool: %s", err)
	}
	dir, file := filepath.Split(pwd + dockerfilePath)
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
		log.Fatalf("Failed to create go-node image: %s", err)
	}

	// remove dangling images
	imgs, err := pool.Client.ListImages(dc.ListImagesOptions{
		Filters: map[string][]string{
			"label": {"tmp=true"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to get list of images from docker: %s", err)
	}
	for _, i := range imgs {
		err = pool.Client.RemoveImage(i.ID)
		if err != nil {
			log.Fatalf("Failed to remove dangling images: %s", err)
		}
	}
	os.Exit(m.Run())
}
