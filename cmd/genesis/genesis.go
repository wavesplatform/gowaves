package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
	"go.uber.org/zap"
)

type cli struct {
	schemeByte string
	seeds      []string
}

func (c *cli) parse() error {
	if *schemeByte == "" {
		return errors.New("please, provide network scheme")
	}
	c.schemeByte = *schemeByte
	c.seeds = strings.Split(*seedsString, ",")
	return nil
}

var (
	schemeByte  = flag.String("scheme-byte", "", "Scheme byte")
	seedsString = flag.String("seeds", "", "Seeds. Example: test1:100_000_000,test2:100_000")
)

func main() {
	var (
		scheme    string
		seed      string
		amounts   string
		pairs     string
		timestamp int64
	)
	flag.StringVar(&scheme, "scheme", "C", "Network scheme byte, defaults to 'C'")
	flag.StringVar(&seed, "seed", "", "Master seed as Base58 string")
	flag.StringVar(&amounts, "amounts", "", "Comma separated transaction amounts")
	flag.StringVar(&pairs, "pairs", "", "Comma separated pairs of address or account seed and amount to produce genesis transactions, eg '3MvRmBpZf6Cm14dY5Nrrq2pj4587EzGTnj4:100_000_000,8GVECo9addsbFumLsmnAU3Cfz7UiF5TGm64zkZnfntdA:100_000'")
	flag.Int64Var(&timestamp, "timestamp", time.Now().UnixMilli(), "Block and transactions timestamp in ms")
	flag.Parse()

	var cliArgs cli
	if err := cliArgs.parse(); err != nil {
		zap.S().Fatal(err)
	}

	t := proto.NewTimestampFromTime(time.Now())

	inf := make([]interface{}, 0, 2*len(cliArgs.seeds))
	for i, v := range cliArgs.seeds {
		splitted := strings.Split(v, ":")
		if len(splitted) != 2 {
			zap.S().Fatal("format should be test1:100000000")
		}
		kp := proto.MustKeyPair([]byte(splitted[0]))
		num, err := strconv.ParseUint(strings.Replace(splitted[1], "_", "", -1), 10, 64)
		if err != nil {
			zap.S().Fatalf("failed to parse seed (%d): %v", i, err)
		}
		inf = append(inf, kp, num)
	}

	genesis, err := genesis_generator.Generate(t, cliArgs.schemeByte[0], inf...)
	if err != nil {
		zap.S().Fatal(err)
	}

	js, err := json.Marshal(genesis)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
