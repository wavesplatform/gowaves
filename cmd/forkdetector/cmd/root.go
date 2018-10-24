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
			a := viper.GetString("waves.network.bind-address")
			fmt.Println("waves.net.bind " + a)

			ctx, cancel := context.WithCancel(context.Background())
			peers := viper.GetStringSlice("waves.network.peers")
			s := server.NewServer(peers)
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
