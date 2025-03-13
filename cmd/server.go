package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/titan/ultraftp/internal/server"
)

var (
	serverPort int
	serverDir  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the FTP server",
	Long: `Start an FTP server that listens for client connections
and handles file transfer operations.

Example:
  ultraftp server --port 2121 --dir /path/to/serve`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting FTP server on port %d serving directory %s\n", serverPort, serverDir)
		if err := server.Start(serverPort, serverDir); err != nil {
			er(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 2121, "Port to listen on")
	serverCmd.Flags().StringVarP(&serverDir, "dir", "d", ".", "Directory to serve")
}
