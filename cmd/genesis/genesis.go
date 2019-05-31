package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

type Cli struct {
	SchemaByte string `kong:"schemabyte,short='b',help='Schema byte.',required"`
	//Schema       string   `kong:"schema,short='s',help='Schema byte.',required"`
	Seed []string `kong:"seed,short='s',help='Seeds.',"`
	//Run  struct {
	//	Addresses string `kong:"address,short='a',help='Addresses connect to.'"`
	//	DeclAddr  string `kong:"decladdr,short='d',help='Address listen on.'"`
	//	HttpAddr  string `kong:"httpaddr,short='w',help='Http addr bind on.'"`
	//} `kong:"cmd,help='Run node'"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {

	var cli Cli
	kong.Parse(&cli)

	t := proto.NewTimestampFromTime(time.Now())

	inf := []interface{}{}
	for _, v := range cli.Seed {
		splitted := strings.Split(v, ":")
		if len(splitted) != 2 {
			zap.S().Fatal("format should be test1:100000000")
		}

		kp := proto.NewKeyPair([]byte(splitted[0]))
		inf = append(inf, kp)
		num, _ := strconv.ParseUint(splitted[1], 10, 64)
		inf = append(inf, int(num))
	}

	genesis, err := genesis_generator.Generate(t, cli.SchemaByte[0], inf...)
	if err != nil {
		zap.S().Fatal(err)
	}

	js, err := json.Marshal(genesis)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
