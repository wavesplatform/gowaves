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
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
	"go.uber.org/zap"
)

/**
usage:
go run cmd/genconfig/genconfig.go -scheme-byte=C -time-shift=-1h -seeds=test1:100_000_000_000_000,test2:100_000 > config.json
*/

type cli struct {
	schemeByte string
	seeds      []string
	timeShift  time.Duration
	bt         uint64
}

func (c *cli) parse() error {
	if *schemeByte == "" {
		return errors.New("please, provide network scheme")
	}
	c.schemeByte = *schemeByte
	c.seeds = strings.Split(*seedsString, ",")
	c.timeShift = *timeShift
	c.bt = *baseTarget
	return nil
}

var (
	schemeByte  = flag.String("scheme-byte", "", "Scheme byte")
	seedsString = flag.String("seeds", "", "Seeds. Example: test1:100_000_000_000_000,test2:100_000")
	timeShift   = flag.Duration("time-shift", 0, "Time shift. Format: +1h, -2h3s.")
	baseTarget  = flag.Uint64("base-target", 0, "Base Target")
)

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	var cliArgs cli
	if err := cliArgs.parse(); err != nil {
		zap.S().Fatal(err)
	}

	now := time.Now()
	ts := proto.NewTimestampFromTime(now)
	if cliArgs.timeShift != 0 {
		ts = proto.NewTimestampFromTime(now.Add(cliArgs.timeShift))
	}
	scheme := cliArgs.schemeByte[0]
	inf := make([]genesis_generator.GenesisTransactionInfo, 0, len(cliArgs.seeds))
	for i, v := range cliArgs.seeds {
		split := strings.Split(v, ":")
		if len(split) != 2 {
			zap.S().Fatal("format should be test1:100000000")
		}
		kp := proto.MustKeyPair([]byte(split[0]))
		addr, err := proto.NewAddressFromPublicKey(scheme, kp.Public)
		if err != nil {
			zap.S().Fatalf("Failed to parse seed (%d): %v", i, err)
		}
		num, err := strconv.ParseUint(strings.Replace(split[1], "_", "", -1), 10, 64)
		if err != nil {
			zap.S().Fatalf("failed to parse seed (%d): %v", i, err)
		}
		inf = append(inf, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: num, Timestamp: ts})
	}

	genesis, err := genesis_generator.GenerateGenesisBlock(scheme, inf, cliArgs.bt, ts)
	if err != nil {
		zap.S().Fatal(err)
	}

	s := *settings.DefaultCustomSettings
	s.Genesis = *genesis
	s.AddressSchemeCharacter = cliArgs.schemeByte[0]

	js, err := json.Marshal(s)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
