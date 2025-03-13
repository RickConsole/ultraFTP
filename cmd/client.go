package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/titan/ultraftp/internal/client"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "FTP client operations",
	Long: `Perform FTP client operations such as get and put files.

Example:
  ultraftp client get ftp://localhost:2121/file.txt local-file.txt
  ultraftp client put local-file.txt ftp://localhost:2121/file.txt`,
}

var getCmd = &cobra.Command{
	Use:   "get [remote-url] [local-path]",
	Short: "Download a file from an FTP server",
	Long: `Download a file from an FTP server to a local path.

Example:
  ultraftp client get ftp://localhost:2121/file.txt local-file.txt`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		remoteURL := args[0]
		localPath := args[1]
		fmt.Printf("Downloading %s to %s\n", remoteURL, localPath)
		if err := client.Get(remoteURL, localPath); err != nil {
			er(err)
		}
		fmt.Println("Download complete")
	},
}

var putCmd = &cobra.Command{
	Use:   "put [local-path] [remote-url]",
	Short: "Upload a file to an FTP server",
	Long: `Upload a local file to an FTP server.

Example:
  ultraftp client put local-file.txt ftp://localhost:2121/file.txt`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		localPath := args[0]
		remoteURL := args[1]
		fmt.Printf("Uploading %s to %s\n", localPath, remoteURL)
		if err := client.Put(localPath, remoteURL); err != nil {
			er(err)
		}
		fmt.Println("Upload complete")
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.AddCommand(getCmd)
	clientCmd.AddCommand(putCmd)
}
