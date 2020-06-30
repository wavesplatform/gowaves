package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/java_opts"
	"go.uber.org/zap"
)

var (
	logLevel = flag.String("log-level", "DEBUG", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	node     = flag.String("node", "", "Path to node executable.")
)

type Argument struct {
	Name         string
	Value        string
	ProdiveEmpty bool
}

type Arguments []Argument

func (a Arguments) String() string {
	s := strings.Builder{}
	for _, row := range a {
		s.WriteString(" -")
		s.WriteString(row.Name)
		s.WriteString(" ")
		s.WriteString(row.Value)
	}
	return s.String()
}

func (a Arguments) Strings() []string {
	var out []string
	for _, row := range a {
		out = append(out, "-"+row.Name)
		if row.Value != "" {
			out = append(out, row.Value)
		}
	}
	return out
}

func (a Arguments) SkipEmpty(name, value string) Arguments {
	if value == "" {
		return a
	}
	return append(a, Argument{
		Name:  name,
		Value: value,
	})
}

func (a Arguments) NonEmpty(name, value string) Arguments {
	if value == "" {
		panic("empty value provided for name: " + name)
	}
	return append(a, Argument{
		Name:  name,
		Value: value,
	})
}

func (a Arguments) Empty(name string) Arguments {
	return append(a, Argument{
		Name: name,
	})
}

func main() {
	flag.Parse()
	// difference between scala System.currentTimeMillis() and time.Now()
	<-time.After(2 * time.Second)
	common.SetupLogger(*logLevel)

	zap.S().Debug(os.Getenv("WAVES_OPTS"))
	zap.S().Debug(os.Environ())

	cfg := java_opts.ParseEnvString(os.Getenv("WAVES_OPTS"))

	arguments := Arguments(nil).
		NonEmpty("log-level", "DEBUG").
		NonEmpty("state-path", cfg.String("waves.directory", "/tmp/waves")).
		SkipEmpty("peers", cfg.String("waves.network.known-peers")).
		SkipEmpty("min-peers-mining", cfg.String("waves.miner.quorum", "1")).
		NonEmpty("declared-address", cfg.String("waves.network.declared-address")).
		SkipEmpty("name", cfg.String("waves.network.node-name", "gowaves")).
		SkipEmpty("peers", strings.Join(cfg.Array("waves.network.known-peers"), ",")).
		Empty("build-extended-api").
		NonEmpty("grpc-address", "0.0.0.0:6870").
		NonEmpty("api-address", "0.0.0.0:6869").
		NonEmpty("blockchain-type", "integration").
		NonEmpty("integration.genesis.signature", cfg.String("waves.blockchain.custom.genesis.signature")).
		NonEmpty("integration.genesis.timestamp", cfg.String("waves.blockchain.custom.genesis.timestamp")).
		NonEmpty("integration.genesis.block-timestamp", cfg.String("waves.blockchain.custom.genesis.block-timestamp")).
		NonEmpty("integration.account-seed", cfg.String("account-seed")).
		NonEmpty("integration.address-scheme-character", cfg.String("waves.blockchain.custom.address-scheme-character", "I")) //

	if cfg.String("waves.miner.enable") == "no" {
		arguments = arguments.Empty("disable-miner")
	}

	cmd := exec.Command(*node, arguments.Strings()...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		zap.S().Errorf("stdout err: %+v", err)
		return
	}

	go func() {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			zap.S().Error(err)
			return
		}
		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println("err: ", m)
		}
	}()
	<-time.After(100 * time.Millisecond)

	err = cmd.Start()
	if err != nil {
		zap.S().Error(err)
		return
	}

	go func() {
		b := make([]byte, 1024*8)
		for {
			n, err := stdout.Read(b)
			if err != nil {
				fmt.Println("stdout.Read(b) err: ", err)
				break
			}
			fmt.Print(string(b[:n]))
		}
	}()

	err = cmd.Wait()
	if err != nil {
		zap.S().Errorf("%+T", err)
		if e, ok := err.(*exec.ExitError); ok {
			zap.S().Errorf("%s", e.Stderr)
		}
		zap.S().Errorf("err: %+v", err)
		return
	}
}
