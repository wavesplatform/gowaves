package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/segmentio/objconv/json"
	"github.com/spf13/afero"
	"github.com/wavesplatform/gowaves/pkg/client"
	"go.uber.org/zap"
	"io/ioutil"
)

var Cli struct {
	Json struct {
		File string `kong:"short='f',help='From file.',required"`
		//Output string `kong:"short='o',help='File output.'"`
	} `kong:"cmd,help='Convert from json to binary'"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	ctx := kong.Parse(&Cli)
	//zap.S().Info(ctx.Command())
	switch ctx.Command() {
	case "json":
		serveJson()
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

	tt := client.TransactionTypeVersion{}
	err = json.Unmarshal(b, &tt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	realType, err := client.GuessTransactionType(&tt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	err = json.Unmarshal(b, realType)
	if err != nil {
		zap.S().Error(err)
		return
	}

	bts, err := realType.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}

	fmt.Print(bts)
}

func inputBytes(path string) ([]byte, error) {
	fs := afero.NewOsFs()

	//if path != "" {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
	//}

}
