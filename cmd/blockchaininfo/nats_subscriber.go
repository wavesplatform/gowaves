package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	zap.S().Infof("Received on %s:\n", msg.Subject)

	switch msg.Data[0] {
	case blockchaininfo.NoPaging:
		contractMsg = msg.Data[1:]
		contractUpdatesInfo := new(g.L2ContractDataEntries)
		if err := contractUpdatesInfo.UnmarshalVT(contractMsg); err != nil {
			log.Printf("Failed to unmarshal contract updates: %v", err)
			return contractMsg
		}
		if err := printContractInfo(contractUpdatesInfo, scheme, path); err != nil {
			log.Printf("Failed to print contract info: %v", err)
			return contractMsg
		}
		contractMsg = nil

	case blockchaininfo.StartPaging:
		contractMsg = append(contractMsg, msg.Data[1:]...)

	case blockchaininfo.EndPaging:
		if contractMsg != nil {
			contractMsg = append(contractMsg, msg.Data[1:]...)
			contractUpdatesInfo := new(g.L2ContractDataEntries)
			if err := contractUpdatesInfo.UnmarshalVT(contractMsg); err != nil {
				log.Printf("Failed to unmarshal contract updates: %v", err)
				return contractMsg
			}

			go func() {
				if err := printContractInfo(contractUpdatesInfo, scheme, path); err != nil {
					log.Printf("Failed to print contract info updates: %v", err)
				}
			}()
			contractMsg = nil
		}
	}

	return contractMsg
}

//const scheme = proto.TestNetScheme
//const path = "/media/alex/ExtremePro/waves/subscription/"

func main() {
	var (
		blockchainType string
		updatesPath    string
		natsURL        string
	)
	// Initialize the zap logger
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer func(l *zap.Logger) {
		err := l.Sync()
		if err != nil {
			log.Fatalf("failed to sync zap logger %v", err)
		}
	}(l)

	flag.StringVar(&blockchainType, "blockchain-type", "testnet", "Blockchain scheme (e.g., stagenet, testnet, mainnet)")
	flag.StringVar(&updatesPath, "updates-path", "", "File path to store contract updates")
	flag.StringVar(&natsURL, "nats-url", nats.DefaultURL, "URL for the NATS server")

	scheme, err := schemeFromString(blockchainType)
	if err != nil {
		zap.S().Fatalf("Failed to parse the blockchain type: %v", err)
	}
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	// Connect to a NATS server
	nc, err := nats.Connect(natsURL)
	if err != nil {
		zap.S().Fatalf("Failed to connect to nats server: %v", err)
		return
	}
	defer nc.Close()

	_, err = nc.Subscribe(blockchaininfo.BlockUpdates, func(msg *nats.Msg) {
		receiveBlockUpdates(msg)
	})
	if err != nil {
		zap.S().Fatalf("Failed to subscribe to block updates: %v", err)
		return
	}

	var contractMsg []byte
	_, err = nc.Subscribe(blockchaininfo.ContractUpdates, func(msg *nats.Msg) {
		contractMsg = receiveContractUpdates(msg, contractMsg, scheme, updatesPath)
	})
	if err != nil {
		zap.S().Fatalf("Failed to subscribe to contract updates: %v", err)
		return
	}
	<-ctx.Done()
	zap.S().Info("NATS subscriber finished...")
}

func schemeFromString(networkType string) (proto.Scheme, error) {
	switch strings.ToLower(networkType) {
	case "mainnet":
		return proto.MainNetScheme, nil
	case "testnet":
		return proto.TestNetScheme, nil
	case "stagenet":
		return proto.StageNetScheme, nil
	default:
		return 0, errors.New("invalid blockchain type string")
	}
}
