package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use: "kube-db",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(2)
	},
}

func main() {
	var c Config
	cobra.OnInitialize(func() {
		initConfig(&c)
	})
	if err := commandRoot(&c).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func commandRoot(c *Config) *cobra.Command {
	rootCmd.PersistentFlags().StringVar(&c.MetricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	rootCmd.PersistentFlags().StringVar(&c.Provider, "provider", "aws", "Provider [aws, gcloud]")
	rootCmd.MarkFlagRequired("Provider")

	rootCmd.AddCommand(commandServe(c))
	return rootCmd
}

func initConfig(c *Config) {
	err := viper.Unmarshal(c)
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}
