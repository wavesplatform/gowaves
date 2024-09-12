package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
	g "github.com/wavesplatform/gowaves/pkg/grpc/l2/blockchain_info"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func printBlockInfo(blockInfoProto *g.BlockInfo) error {
	blockInfo, err := blockchaininfo.BUpdatesInfoFromProto(blockInfoProto)
	if err != nil {
		return err
	}
	blockInfoJSON, err := json.Marshal(blockInfo)
	if err != nil {
		return err
	}
	log.Println(string(blockInfoJSON))
	return nil
}

func printContractInfo(contractInfoProto *g.L2ContractDataEntries, scheme proto.Scheme, path string) error {
	contractInfo, err := blockchaininfo.L2ContractDataEntriesFromProto(contractInfoProto, scheme)
	if err != nil {
		return err
	}
	prettyJSON, err := json.MarshalIndent(contractInfo, "", "    ")
	if err != nil {
		log.Println("Error converting to pretty JSON:", err)
		return err
	}
	heightStr := strconv.Itoa(int(contractInfoProto.Height))
	// Write the pretty JSON to a file
	err = os.WriteFile(path+heightStr+".json", prettyJSON, 0600)
	if err != nil {
		log.Println("Error writing to file:", err)
		return err
	}

	return nil
}

func receiveBlockUpdates(msg *nats.Msg) {
	blockUpdatesInfo := new(g.BlockInfo)
	unmrshlErr := blockUpdatesInfo.UnmarshalVT(msg.Data)
	if unmrshlErr != nil {
		log.Printf("failed to unmarshal block updates, %v", unmrshlErr)
		return
	}

	err := printBlockInfo(blockUpdatesInfo)
	if err != nil {
		return
	}
	log.Printf("Received on %s:\n", msg.Subject)
}

func receiveContractUpdates(msg *nats.Msg, contractMsg []byte, scheme proto.Scheme, path string) []byte {
	log.Printf("Received on %s:\n", msg.Subject)
	if msg.Data[0] == blockchaininfo.NoPaging {
		contractMsg = msg.Data[1:]
		contractUpdatesInfo := new(g.L2ContractDataEntries)
		unmrshlErr := contractUpdatesInfo.UnmarshalVT(contractMsg)
		if unmrshlErr != nil {
			log.Printf("failed to unmarshal contract updates, %v", unmrshlErr)
			return contractMsg
		}
		err := printContractInfo(contractUpdatesInfo, scheme, path)
		if err != nil {
			return contractMsg
		}
		contractMsg = nil
		return contractMsg
	}

	if msg.Data[0] == blockchaininfo.StartPaging {
		contractMsg = append(contractMsg, msg.Data[1:]...)
	}

	if msg.Data[0] == blockchaininfo.EndPaging && contractMsg != nil {
		contractMsg = append(contractMsg, msg.Data[1:]...)
		contractUpdatesInfo := new(g.L2ContractDataEntries)
		unmrshlErr := contractUpdatesInfo.UnmarshalVT(contractMsg)
		if unmrshlErr != nil {
			log.Printf("failed to unmarshal contract updates, %v", unmrshlErr)
			return contractMsg
		}

		go func() {
			prntErr := printContractInfo(contractUpdatesInfo, scheme, path)
			if prntErr != nil {
				log.Printf("failed to print contract info updates")
			}
		}()
		contractMsg = nil
	}
	return contractMsg
}

const scheme = proto.TestNetScheme
const path = "/media/alex/My_Book/dolgavin/waves/subscriber/contract_data/"

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	// Connect to a NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Print(err)
		return
	}
	defer nc.Close()

	_, err = nc.Subscribe(blockchaininfo.BlockUpdates, func(msg *nats.Msg) {
		receiveBlockUpdates(msg)
	})
	if err != nil {
		log.Printf("failed to subscribe to block updates, %v", err)
		return
	}

	var contractMsg []byte
	_, err = nc.Subscribe(blockchaininfo.ContractUpdates, func(msg *nats.Msg) {
		contractMsg = receiveContractUpdates(msg, contractMsg, scheme, path)
	})
	if err != nil {
		log.Printf("failed to subscribe to contract updates, %v", err)
		return
	}
	<-ctx.Done()
	log.Println("Terminations of nats subscriber")
}
