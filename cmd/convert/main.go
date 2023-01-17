package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var (
	command    = flag.String("command", "", "Command which will be executed. Values: 'json' - convert from json to binary, 'bytes' - convert from binary to json.")
	file       = flag.String("file", "", "From file.")
	schemeByte = flag.String("scheme-byte", "", "Network scheme.")
)

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	if *file == "" {
		zap.S().Fatal("please, provide file argument")
	}
	if *schemeByte == "" {
		zap.S().Fatal("please, provide scheme-byte argument")
	}
	if len(*schemeByte) != 1 {
		zap.S().Fatal("invalid scheme-byte argument %q", *schemeByte)
	}
	scheme := []byte(*schemeByte)[0]
	switch *command {
	case "json":
		if err := serveJson(*file, scheme); err != nil {
			zap.S().Fatalf("failed to serveJSON: %v", err)
		}
	case "bytes":
		if err := serveBinary(*file, scheme); err != nil {
			zap.S().Fatalf("failed to serveBinary: %v", err)
		}
	case "":
		zap.S().Fatal("please, provide command argument")
	default:
		zap.S().Fatalf("invalid command %q", *command)

	}
}

func serveJson(pathToJSON string, scheme proto.Scheme) error {
	b, err := inputBytes(pathToJSON)
	if err != nil {
		return err
	}

	tt := proto.TransactionTypeVersion{}
	err = json.Unmarshal(b, &tt)
	if err != nil {
		return err
	}

	realType, err := proto.GuessTransactionType(&tt)
	if err != nil {
		return err
	}

	err = proto.UnmarshalTransactionFromJSON(b, scheme, realType)
	if err != nil {
		return err
	}

	bts, err := realType.MarshalBinary(scheme)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(bts)
	return err
}

func serveBinary(pathToBinary string, scheme proto.Scheme) error {
	b, err := inputBytes(pathToBinary)
	if err != nil {
		return err
	}

	trans, err := proto.BytesToTransaction(b, scheme)
	if err != nil {
		return err
	}

	js, err := json.Marshal(trans)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(js)
	return err
}

func inputBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
	if err != nil {
		return nil, err
	}
	return data, nil
}
