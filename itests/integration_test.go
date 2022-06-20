package integration_test

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
)

func TestMain(m *testing.M) {
	paths, err := config.CreateFileConfigs()
	if err != nil {
		log.Fatalf("couldn't create config %s", err)
	}
	docker, err := d.NewDocker()
	if err != nil {
		log.Fatalf("couldn't create docker pool %s", err)
	}
	err = docker.RunContainers(paths)
	if err != nil {
		log.Fatalf("couldn't run docker containers %s", err)
	}
	code := m.Run()
	err = docker.Purge()
	if err != nil {
		log.Fatalf("couldn't purge docker containers %s", err)
	}
	os.Exit(code)
}

func TestCheckHeight(t *testing.T) {
	goHeight, err := d.GoNodeClient.GetHeight()
	if err != nil {
		log.Printf("failed to get heigth from go node: %s", err)
		return
	}
	scalaHeight, err := d.ScalaNodeClient.GetHeight()
	if err != nil {
		log.Printf("failed to get heigth from scala node: %s", err)
		return
	}
	assert.Equal(t, goHeight, scalaHeight)
}
