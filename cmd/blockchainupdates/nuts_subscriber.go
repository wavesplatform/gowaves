package blockchainupdates

import (
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/blockchainupdates"
	"log"
	"runtime"
)

func main() {
	// Connect to a NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	for _, topic := range blockchainupdates.Topics {
		_, err = nc.Subscribe(topic, func(msg *nats.Msg) {
			log.Printf("Received on %s: %s\n", msg.Subject, string(msg.Data))
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	runtime.Goexit()
}
