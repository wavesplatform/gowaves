package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var Cli struct {
	Json struct {
		File string `kong:"short='f',help='From file.',required"`
	} `kong:"cmd,help='Convert from json to binary'"`
	Bytes struct {
		SchemeByte string `kong:"short='s',help='Network scheme.',required"`
		File       string `kong:"short='f',help='From file.',required"`
	} `kong:"cmd,help='Convert from binary to json'"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	ctx := kong.Parse(&Cli)
	switch ctx.Command() {
	case "json":
		serveJson()
	case "bytes":
		serveBinary(Cli.Bytes.SchemeByte[0])
	default:
		zap.S().Error(ctx.Command())
		return
	}
}

func serveJson() {
	b, err := inputBytes(Cli.Json.File)
	if err != nil {
		fmt.Println(err)
		return
	}

	tt := proto.TransactionTypeVersion{}
	err = json.Unmarshal(b, &tt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	realType, err := proto.GuessTransactionType(&tt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	err = json.Unmarshal(b, realType)
	if err != nil {
		fmt.Println(err)
		zap.S().Error(err)
		return
	}

	bts, err := realType.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	_, _ = os.Stdout.Write(bts)
}

func serveBinary(scheme proto.Scheme) {
	b, err := inputBytes(Cli.Bytes.File)
	if err != nil {
		fmt.Println(err)
		return
	}

	trans, err := proto.BytesToTransaction(b, scheme)
	if err != nil {
		zap.S().Error(err)
		return
	}

	js, err := json.Marshal(trans)
	if err != nil {
		fmt.Println(err)
		zap.S().Error(err)
		return
	}
	_, _ = os.Stdout.Write(js)
}

func inputBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
