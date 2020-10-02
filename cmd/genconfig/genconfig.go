package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
	"go.uber.org/zap"
)

/**
usage:
go run cmd/genconfig/genconfig.go --scheme-byte=C --time-shift=-1h --seed=test1:100_000_000_000_000 > config.json
*/

type Cli struct {
	SchemeByte string   `kong:"schemebyte,help='Scheme byte.',required"`
	Seed       []string `kong:"seed,help='Seeds.',"`
	TimeShift  string   `kong:"help='Format: +1h, -2h3s.',optional"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {

	var cli Cli
	kong.Parse(&cli)

	t := proto.NewTimestampFromTime(time.Now())

	if cli.TimeShift != "" {
		d, err := time.ParseDuration(cli.TimeShift)
		if err != nil {
			zap.S().Fatal(err)
			return
		}
		t = proto.NewTimestampFromTime(time.Now().Add(d))
	}

	inf := []interface{}{}
	for _, v := range cli.Seed {
		splitted := strings.Split(v, ":")
		if len(splitted) != 2 {
			zap.S().Fatal("format should be test1:100000000")
		}
		kp := proto.MustKeyPair([]byte(splitted[0]))
		inf = append(inf, kp)
		num, _ := strconv.ParseUint(strings.Replace(splitted[1], "_", "", -1), 10, 64)
		inf = append(inf, int(num))
	}

	genesis, err := genesis_generator.Generate(t, cli.SchemeByte[0], inf...)
	if err != nil {
		zap.S().Fatal(err)
	}

	s := *settings.DefaultCustomSettings
	s.Genesis = *genesis
	s.AddressSchemeCharacter = cli.SchemeByte[0]

	js, err := json.Marshal(s)
	if err != nil {
		zap.S().Fatal(err)
	}

	fmt.Println(string(js))
}
