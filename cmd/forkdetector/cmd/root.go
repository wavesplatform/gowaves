package cmd

import (
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

			peers := viper.GetStringSlice("waves.network.peers")
			s := &server.Server{BootPeerAddrs: peers}
			s.RunClients()

			var gracefulStop = make(chan os.Signal)
			signal.Notify(gracefulStop, syscall.SIGTERM)
			signal.Notify(gracefulStop, syscall.SIGINT)

			select {
			case sig := <-gracefulStop:
				zap.S().Infow("Caught signal, stopping", "signal", sig)
				os.Exit(0)
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
