package main

import (
	"errors"
	logger "github.com/Sirupsen/logrus"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/pnegahdar/electro/pkg/sites"
	reaper "github.com/ramr/go-reaper"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var listenAddrHTTPS string
var listenAddrHTTP string
var dataDir string
var wildCard string
var adminUsername string
var adminPassword string

func init() {
	// Add flags
	startServerCommand.Flags().StringVarP(&listenAddrHTTP, "listen-http", "l", ":4200", "The address to listen for http connections. e.g. 0.0.0.0:80, :5000, etc.")
	startServerCommand.Flags().StringVarP(&listenAddrHTTPS, "listen-https", "s", "", "The address to listen for https servers. https server not started when ommited.")
	startServerCommand.Flags().StringVarP(&dataDir, "data-dir", "d", "~/.electro", "The directory to store the state and certificates.")
	startServerCommand.Flags().StringVarP(&wildCard, "wildcard", "w", "", "The wildcard domain to serve e.g electro.site.com")
	startServerCommand.Flags().StringVarP(&adminUsername, "admin-username", "u", "", "The username for the admin dashboard and api. Must provide password as well.")
	startServerCommand.Flags().StringVarP(&adminPassword, "admin-password", "p", "", "The password for the admin dashboard and api. Must provide username as well.")
	rootCmd.AddCommand(startServerCommand)

}

func runLoop(name string, addr string, run func() error) {
	for {
		logger.Infof("Starting %s Server on: %s", name, addr)
		err := run()
		logger.Errorf("Http server exited with error: %v", err)
		logger.Infof("Restarting %s server in 5 seconds.", name)
		<-time.After(time.Second * 5)
	}

}

var startServerCommand = &cobra.Command{
	Use:   "start",
	Short: "Start the server.",
	Run: func(cmd *cobra.Command, args []string) {
		// Checking pid here to avoid printing reaper does.
		mypid := os.Getpid()
		if mypid == 1 {
			go reaper.Reap()
		}
		if dataDir == "" {
			dataDir = "~/.electro"
		}
		if wildCard == "" {
			exitErr(errors.New("--wildcard domain required."))
		}
		if (adminUsername != "" && adminPassword == "") || (adminUsername == "" && adminPassword != "") {
			exitErr(errors.New("Both admin-username and admin-password must be provided."))
		}

		fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "fe/build"}
		site, err := sites.NewManager(dataDir, wildCard, fs, adminUsername, adminPassword)
		exitErr(err)

		if listenAddrHTTP == "" {
			exitErr(errors.New("Must past http listen port -l"))
		}
		go func() {
			runLoop("http server", listenAddrHTTP, func() error {
				return site.RunHTTP(listenAddrHTTP)
			})
		}()
		if listenAddrHTTPS != "" {
			go func() {
				runLoop("https server", listenAddrHTTPS, func() error {
					return site.RunHTTPS(listenAddrHTTPS)
				})
			}()
		}
		select {}

	},
}
