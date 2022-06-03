package integration_test

import (
	"log"
	"os"
	"testing"
	"time"

	d "github.com/wavesplatform/gowaves/itests/docker"
)

func TestMain(m *testing.M) {
	docker, err := d.NewDocker()
	if err != nil {
		log.Fatalf("couldn't create docker pool %s", err)
	}
	err = docker.RunContainers()
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

func TestSleep(t *testing.T) {
	time.Sleep(2 * time.Minute)
}
