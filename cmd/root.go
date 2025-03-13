package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ultraftp",
	Short: "UltraFTP - A minimal FTP server and client",
	Long: `UltraFTP is a lightweight FTP server and client implementation
that provides basic file transfer capabilities using the standard FTP protocol.

Use the 'server' subcommand to start an FTP server or the 'client' subcommand
to interact with an FTP server.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Initialize any global configuration here
}

func er(msg interface{}) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
