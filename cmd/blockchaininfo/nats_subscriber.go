package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func printContractInfo(contractInfoProto *g.L2ContractDataEntries, scheme proto.Scheme) error {
	contractInfo, err := blockchaininfo.L2ContractDataEntriesFromProto(contractInfoProto, scheme)
	if err != nil {
		return err
	}
	// temporary

	prettyJSON, err := json.MarshalIndent(contractInfo, "", "    ")
	if err != nil {
		fmt.Println("Error converting to pretty JSON:", err)
		return err
	}
	heightStr := strconv.Itoa(int(contractInfoProto.Height))
	// Write the pretty JSON to a file named "index.json"
	err = os.WriteFile("/media/alex/My_Book/dolgavin/waves/subscriber/contract_data/"+heightStr+".json", prettyJSON, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	contractInfoJSON, err := json.Marshal(contractInfo)
	if err != nil {
		return err
	}
	log.Println(string(contractInfoJSON))
	return nil
}

func main() {
	scheme := proto.TestNetScheme
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	// Connect to a NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Print(err)
		return
	}
	defer nc.Close()

	//_, err = nc.Subscribe(blockchaininfo.BlockUpdates, func(msg *nats.Msg) {
	//	blockUpdatesInfo := new(g.BlockInfo)
	//	unmrshlErr := blockUpdatesInfo.UnmarshalVT(msg.Data)
	//	if unmrshlErr != nil {
	//		log.Printf("failed to unmarshal block updates, %v", unmrshlErr)
	//		return
	//	}
	//
	//	err = printBlockInfo(blockUpdatesInfo)
	//	if err != nil {
	//		return
	//	}
	//	log.Printf("Received on %s:\n", msg.Subject)
	//})
	//if err != nil {
	//	log.Printf("failed to subscribe to block updates, %v", err)
	//	return
	//}

	var contractMsg []byte
	_, err = nc.Subscribe(blockchaininfo.ContractUpdates, func(msg *nats.Msg) {
		if msg.Data[0] == blockchaininfo.NO_PAGING {
			contractMsg = msg.Data[1:]
			contractUpdatesInfo := new(g.L2ContractDataEntries)
			unmrshlErr := contractUpdatesInfo.UnmarshalVT(contractMsg)
			if unmrshlErr != nil {
				log.Printf("failed to unmarshal contract updates, %v", unmrshlErr)
				return
			}
			err = printContractInfo(contractUpdatesInfo, scheme)
			if err != nil {
				return
			}
			contractMsg = nil
		}

		if msg.Data[0] == blockchaininfo.START_PAGING {
			contractMsg = append(contractMsg, msg.Data[1:]...)
		}

		if msg.Data[0] == blockchaininfo.END_PAGING && contractMsg != nil {
			contractMsg = append(contractMsg, msg.Data[1:]...)
			contractUpdatesInfo := new(g.L2ContractDataEntries)
			unmrshlErr := contractUpdatesInfo.UnmarshalVT(contractMsg)
			if unmrshlErr != nil {
				log.Printf("failed to unmarshal contract updates, %v", unmrshlErr)
				return
			}

			err = printContractInfo(contractUpdatesInfo, scheme)
			if err != nil {
				return
			}
			contractMsg = nil
		}

	})
	if err != nil {
		log.Printf("failed to subscribe to contract updates, %v", err)
		return
	}
	<-ctx.Done()
	log.Println("Terminations of nats subscriber")
}
