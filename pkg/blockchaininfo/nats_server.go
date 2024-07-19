package blockchaininfo

import (
	"github.com/nats-io/nats-server/v2/server"
	"log"
)

func RunBlockchainUpdatesServer() {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: 4222,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("failed to create NATS server: %v", err)
	}

	go s.Start()

	if !s.ReadyForConnections(10 * server.AUTH_TIMEOUT) {
		log.Fatal("NATS Server not ready for connections")
	}

	log.Println("NATS Server is running on port 4222")

	// Block main goroutine so the server keeps running
	select {}
}
