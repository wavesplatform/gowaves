package main

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"log"
)

func printBlockInfo(blockInfoProto *g.BlockInfo) error {
	blockInfo, err := blockchaininfo.BUpdatesInfoFromProto(blockInfoProto)
	if err != nil {
		return err
	}
	blockInfoJson, err := json.Marshal(blockInfo)
	fmt.Println(string(blockInfoJson))
	return nil
}

func printContractInfo(contractInfoProto *g.L2ContractDataEntries) error {
	contractInfo, err := blockchaininfo.L2ContractDataEntriesFromProto(contractInfoProto)
	if err != nil {
		return err
	}
	contractInfoJson, err := json.Marshal(contractInfo)
	fmt.Println(string(contractInfoJson))
	return nil
}

func main() {
	// Connect to a NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	topic := blockchaininfo.BlockUpdates
	//for _, topic := range blockchaininfo.Topics {
	_, err = nc.Subscribe(topic, func(msg *nats.Msg) {

		//log.Printf("Received on %s: %s\n", msg.Subject, string(msg.Data))
		if topic == blockchaininfo.BlockUpdates {
			blockUpdatesInfo := new(g.BlockInfo)
			err := blockUpdatesInfo.UnmarshalVT(msg.Data)
			if err != nil {
				return
			}

			err = printBlockInfo(blockUpdatesInfo)
			if err != nil {
				return
			}
			log.Printf("Received on %s:\n", msg.Subject)
		}

		if topic == blockchaininfo.ContractUpdates {
			contractUpdatesInfo := new(g.L2ContractDataEntries)
			err := contractUpdatesInfo.UnmarshalVT(msg.Data)
			if err != nil {
				return
			}
			log.Printf("Received on %s:\n", msg.Subject)

			err = printContractInfo(contractUpdatesInfo)
			if err != nil {
				return
			}

		}

	})
	if err != nil {
		log.Fatal(err)
	}
	//}
	// Block main goroutine so the server keeps running
	select {}
}
