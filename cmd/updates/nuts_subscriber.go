package updates

import (
	"github.com/nats-io/nats.go"
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

	for _, topic := range topics {
		_, err = nc.Subscribe(topic, func(msg *nats.Msg) {
			log.Printf("Received on %s: %s\n", msg.Subject, string(msg.Data))
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	runtime.Goexit()
}
