package main

import (
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"github.com/pnegahdar/electro"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "electro",
	Short: "A mini static file repo server..",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(electro.Version)
	},
}

func exitErr(err error) {
	if err != nil {
		logger.Fatalf("Failed with error: %s", err)
	}
}

func init() {
	logger.SetLevel(logger.DebugLevel)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		exitErr(err)
	}
}
