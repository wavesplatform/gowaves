package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config  string
	rootCmd = &cobra.Command{
		Use:   "forkdetector",
		Short: "detects forks in waves",
		Long:  `detects forks in waves`,
		Run: func(cmd *cobra.Command, args []string) {

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
