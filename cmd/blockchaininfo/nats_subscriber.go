package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
	"os"
	"os/signal"
	"syscall"
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

func printContractInfo(contractInfoProto *g.L2ContractDataEntries, scheme proto.Scheme) error {
	contractInfo, err := blockchaininfo.L2ContractDataEntriesFromProto(contractInfoProto, scheme)
	if err != nil {
		return err
	}
	contractInfoJson, err := json.Marshal(contractInfo)
	fmt.Println(string(contractInfoJson))
	return nil
}

func main() {
	scheme := proto.TestNetScheme
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	// Connect to a NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	_, err = nc.Subscribe(blockchaininfo.BlockUpdates, func(msg *nats.Msg) {
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
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = nc.Subscribe(blockchaininfo.ContractUpdates, func(msg *nats.Msg) {
		contractUpdatesInfo := new(g.L2ContractDataEntries)
		err := contractUpdatesInfo.UnmarshalVT(msg.Data)
		if err != nil {
			return
		}
		log.Printf("Received on %s:\n", msg.Subject)

		err = printContractInfo(contractUpdatesInfo, scheme)
		if err != nil {
			return
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	<-ctx.Done()
	log.Println("Terminations of nats subscriber")
}
