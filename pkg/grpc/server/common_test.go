package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

const (
	sleepTime = 2 * time.Second
)

var (
	server       *Server
	grpcTestAddr string
)

func connect(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	assert.NoError(t, err, "grpc.Dial() failed")
	return conn
}

func TestMain(m *testing.M) {
	server = &Server{}
	grpcTestAddr = fmt.Sprintf("127.0.0.1:%d", freeport.GetPort())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Run(ctx, grpcTestAddr); err != nil {
			log.Fatalf("server.Run(): %v\n", err)
		}
	}()

	time.Sleep(sleepTime)
	code := m.Run()
	cancel()
	os.Exit(code)
}
