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
}

func (c *cli) parse() error {
	if *schemeByte == "" {
		return errors.New("please, provide network scheme")
	}
	c.schemeByte = *schemeByte
	c.seeds = strings.Split(*seedsString, ",")
	c.timeShift = *timeShift
	return nil
}

var (
	schemeByte  = flag.String("scheme-byte", "", "Scheme byte")
	seedsString = flag.String("seeds", "", "Seeds. Example: test1:100_000_000_000_000,test2:100_000")
	timeShift   = flag.Duration("time-shift", 0, "Time shift. Format: +1h, -2h3s.")
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
	t := proto.NewTimestampFromTime(now)
	if cliArgs.timeShift != 0 {
		t = proto.NewTimestampFromTime(now.Add(cliArgs.timeShift))
	}

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

	s := *settings.DefaultCustomSettings
	s.Genesis = *genesis
	s.AddressSchemeCharacter = cliArgs.schemeByte[0]

	js, err := json.Marshal(s)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
