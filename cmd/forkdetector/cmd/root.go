package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wavesplatform/gowaves/pkg/server"
	"go.uber.org/zap"
)

var (
	config  string
	rootCmd = &cobra.Command{
		Use:   "forkdetector",
		Short: "detects forks in waves",
		Long:  `detects forks in waves`,
		Run: func(cmd *cobra.Command, args []string) {
			logger, _ := zap.NewDevelopment()
			zap.ReplaceGlobals(logger)

			ctx, cancel := context.WithCancel(context.Background())
			s, err := server.NewServer(
				server.WithPeers(viper.GetStringSlice("waves.network.peers")),
				server.WithLevelDBPath(viper.GetString("waves.storage.path")),
				server.WithGenesis(viper.GetString("waves.blockchain.genesis")),
				server.WithRestAddr(viper.GetString("waves.network.rest-address")),
				server.WithDeclaredAddr(viper.GetString("waves.network.declared-address")),
			)
			if err != nil {
				zap.S().Error("failed to create a new server instance ", err)
				cancel()
				return
			}

			s.RunClients(ctx)
			defer s.Stop()

			var gracefulStop = make(chan os.Signal)
			signal.Notify(gracefulStop, syscall.SIGTERM)
			signal.Notify(gracefulStop, syscall.SIGINT)

			select {
			case sig := <-gracefulStop:
				cancel()
				zap.S().Infow("Caught signal, stopping", "signal", sig)
			}
		},
	}
)

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&config, "config", "", "config file")
}

func initConfig() {
	if config != "" {
		viper.SetConfigFile(config)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("$HOME/.waves/")
		viper.AddConfigPath(".")

	}
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
