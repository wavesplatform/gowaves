package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
	"go.uber.org/zap"
)

type Cli struct {
	SchemeByte string   `kong:"schemebyte,help='Scheme byte.',required"`
	Seed       []string `kong:"seed,help='Seeds.',"`
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

	genesis, err := genesis_generator.Generate(t, cli.SchemeByte[0], inf...)
	if err != nil {
		zap.S().Fatal(err)
	}

	js, err := json.Marshal(genesis)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
