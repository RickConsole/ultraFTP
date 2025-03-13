package client

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// InteractiveSession represents an interactive FTP session
type InteractiveSession struct {
	client *FTPClient
	reader *bufio.Reader
}

// NewInteractiveSession creates a new interactive FTP session
func NewInteractiveSession(client *FTPClient) *InteractiveSession {
	return &InteractiveSession{
		client: client,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Start begins the interactive session
func (s *InteractiveSession) Start() error {
	fmt.Println("Connected to FTP server. Type 'help' for available commands, 'quit' to exit.")

	for {
		fmt.Print("ftp> ")
		input, err := s.reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Parse the command and arguments
		parts := strings.Fields(input)
		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		// Process the command
		if quit := s.processCommand(cmd, args); quit {
			return nil
		}
	}
}

// processCommand processes a command and returns true if the session should end
func (s *InteractiveSession) processCommand(cmd string, args []string) bool {
	switch cmd {
	case "quit", "exit", "bye":
		fmt.Println("Goodbye!")
		return true

	case "help":
		s.printHelp()

	case "ls", "dir":
		s.listFiles(args)

	case "cd", "cwd":
		if len(args) < 1 {
			fmt.Println("Usage: cd <directory>")
			return false
		}
		s.changeDirectory(args[0])

	case "pwd":
		s.printWorkingDirectory()

	case "get":
		if len(args) < 1 {
			fmt.Println("Usage: get <remote-file> [local-file]")
			return false
		}

		remoteFile := args[0]
		localFile := remoteFile
		if len(args) > 1 {
			localFile = args[1]
		}

		s.downloadFile(remoteFile, localFile)

	case "put":
		if len(args) < 1 {
			fmt.Println("Usage: put <local-file> [remote-file]")
			return false
		}

		localFile := args[0]
		remoteFile := filepath.Base(localFile)
		if len(args) > 1 {
			remoteFile = args[1]
		}

		s.uploadFile(localFile, remoteFile)

	case "mkdir":
		if len(args) < 1 {
			fmt.Println("Usage: mkdir <directory>")
			return false
		}
		s.makeDirectory(args[0])

	case "rmdir":
		if len(args) < 1 {
			fmt.Println("Usage: rmdir <directory>")
			return false
		}
		s.removeDirectory(args[0])

	case "rm", "delete":
		if len(args) < 1 {
			fmt.Println("Usage: rm <file>")
			return false
		}
		s.deleteFile(args[0])

	default:
		fmt.Printf("Unknown command: %s\nType 'help' for available commands.\n", cmd)
	}

	return false
}

// printHelp prints the available commands
func (s *InteractiveSession) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  ls, dir                  List files in current directory")
	fmt.Println("  cd, cwd <directory>      Change working directory")
	fmt.Println("  pwd                      Print working directory")
	fmt.Println("  get <remote> [local]     Download a file")
	fmt.Println("  put <local> [remote]     Upload a file")
	fmt.Println("  mkdir <directory>        Create a directory")
	fmt.Println("  rmdir <directory>        Remove a directory")
	fmt.Println("  rm, delete <file>        Delete a file")
	fmt.Println("  help                     Show this help")
	fmt.Println("  quit, exit, bye          Exit the shell")
}

// listFiles lists files in the current directory
func (s *InteractiveSession) listFiles(args []string) {
	// Enter passive mode
	err := s.client.enterPassiveMode()
	if err != nil {
		fmt.Printf("Error entering passive mode: %s\n", err)
		return
	}

	// Send LIST command
	cmd := "LIST"
	if len(args) > 0 {
		cmd = fmt.Sprintf("LIST %s", args[0])
	}

	code, msg, err := s.client.sendCommand(cmd)
	if err != nil {
		fmt.Printf("Error sending LIST command: %s\n", err)
		return
	}

	if code != 150 && code != 125 {
		fmt.Printf("Failed to list directory: %d %s\n", code, msg)
		return
	}

	// Read the directory listing
	if s.client.dataConn != nil {
		reader := bufio.NewReader(s.client.dataConn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading directory listing: %s\n", err)
				}
				break
			}
			fmt.Print(line)
		}

		// Close the data connection
		s.client.dataConn.Close()
		s.client.dataConn = nil

		// Read the transfer complete message
		code, msg, err = s.client.readResponse()
		if err != nil {
			fmt.Printf("Error reading transfer complete message: %s\n", err)
			return
		}

		if code != 226 && code != 250 {
			fmt.Printf("Unexpected response after transfer: %d %s\n", code, msg)
		}
	}
}

// changeDirectory changes the current working directory
func (s *InteractiveSession) changeDirectory(dir string) {
	code, msg, err := s.client.sendCommand(fmt.Sprintf("CWD %s", dir))
	if err != nil {
		fmt.Printf("Error changing directory: %s\n", err)
		return
	}

	if code != 250 {
		fmt.Printf("Failed to change directory: %d %s\n", code, msg)
	} else {
		fmt.Printf("Changed to directory: %s\n", dir)
	}
}

// printWorkingDirectory prints the current working directory
func (s *InteractiveSession) printWorkingDirectory() {
	code, msg, err := s.client.sendCommand("PWD")
	if err != nil {
		fmt.Printf("Error getting working directory: %s\n", err)
		return
	}

	if code != 257 {
		fmt.Printf("Failed to get working directory: %d %s\n", code, msg)
	} else {
		// Extract the directory from the response
		// The format is typically: 257 "/some/directory" is current directory
		start := strings.Index(msg, "\"")
		end := strings.LastIndex(msg, "\"")
		if start != -1 && end != -1 && start < end {
			dir := msg[start+1 : end]
			fmt.Printf("Current directory: %s\n", dir)
		} else {
			fmt.Println(msg)
		}
	}
}

// downloadFile downloads a file from the server
func (s *InteractiveSession) downloadFile(remoteFile, localFile string) {
	// Set binary mode
	_, _, err := s.client.sendCommand("TYPE I")
	if err != nil {
		fmt.Printf("Failed to set binary mode: %s\n", err)
		return
	}

	// Enter passive mode
	err = s.client.enterPassiveMode()
	if err != nil {
		fmt.Printf("Error entering passive mode: %s\n", err)
		return
	}

	// Send RETR command
	code, msg, err := s.client.sendCommand(fmt.Sprintf("RETR %s", remoteFile))
	if err != nil {
		fmt.Printf("Error sending RETR command: %s\n", err)
		return
	}

	if code != 150 && code != 125 {
		fmt.Printf("Failed to retrieve file: %d %s\n", code, msg)
		return
	}

	// Create the local file
	file, err := os.Create(localFile)
	if err != nil {
		fmt.Printf("Failed to create local file: %s\n", err)
		return
	}
	defer file.Close()

	fmt.Printf("Downloading %s to %s...\n", remoteFile, localFile)

	// Copy the data
	bytesTransferred, err := io.Copy(file, s.client.dataConn)
	if err != nil {
		fmt.Printf("Error downloading file: %s\n", err)
		return
	}

	// Close the data connection
	s.client.dataConn.Close()
	s.client.dataConn = nil

	// Read the transfer complete message
	code, msg, err = s.client.readResponse()
	if err != nil {
		fmt.Printf("Error reading transfer complete message: %s\n", err)
		return
	}

	if code != 226 && code != 250 {
		fmt.Printf("Unexpected response after transfer: %d %s\n", code, msg)
	} else {
		fmt.Printf("Download complete. %d bytes transferred.\n", bytesTransferred)
	}
}

// uploadFile uploads a file to the server
func (s *InteractiveSession) uploadFile(localFile, remoteFile string) {
	// Open the local file
	file, err := os.Open(localFile)
	if err != nil {
		fmt.Printf("Failed to open local file: %s\n", err)
		return
	}
	defer file.Close()

	// Set binary mode
	_, _, err = s.client.sendCommand("TYPE I")
	if err != nil {
		fmt.Printf("Failed to set binary mode: %s\n", err)
		return
	}

	// Enter passive mode
	err = s.client.enterPassiveMode()
	if err != nil {
		fmt.Printf("Error entering passive mode: %s\n", err)
		return
	}

	// Send STOR command
	code, msg, err := s.client.sendCommand(fmt.Sprintf("STOR %s", remoteFile))
	if err != nil {
		fmt.Printf("Error sending STOR command: %s\n", err)
		return
	}

	if code != 150 && code != 125 {
		fmt.Printf("Failed to store file: %d %s\n", code, msg)
		return
	}

	fmt.Printf("Uploading %s to %s...\n", localFile, remoteFile)

	// Copy the data
	bytesTransferred, err := io.Copy(s.client.dataConn, file)
	if err != nil {
		fmt.Printf("Error uploading file: %s\n", err)
		return
	}

	// Close the data connection
	s.client.dataConn.Close()
	s.client.dataConn = nil

	// Read the transfer complete message
	code, msg, err = s.client.readResponse()
	if err != nil {
		fmt.Printf("Error reading transfer complete message: %s\n", err)
		return
	}

	if code != 226 && code != 250 {
		fmt.Printf("Unexpected response after transfer: %d %s\n", code, msg)
	} else {
		fmt.Printf("Upload complete. %d bytes transferred.\n", bytesTransferred)
	}
}

// makeDirectory creates a directory on the server
func (s *InteractiveSession) makeDirectory(dir string) {
	code, msg, err := s.client.sendCommand(fmt.Sprintf("MKD %s", dir))
	if err != nil {
		fmt.Printf("Error creating directory: %s\n", err)
		return
	}

	if code != 257 {
		fmt.Printf("Failed to create directory: %d %s\n", code, msg)
	} else {
		fmt.Printf("Directory created: %s\n", dir)
	}
}

// removeDirectory removes a directory from the server
func (s *InteractiveSession) removeDirectory(dir string) {
	code, msg, err := s.client.sendCommand(fmt.Sprintf("RMD %s", dir))
	if err != nil {
		fmt.Printf("Error removing directory: %s\n", err)
		return
	}

	if code != 250 {
		fmt.Printf("Failed to remove directory: %d %s\n", code, msg)
	} else {
		fmt.Printf("Directory removed: %s\n", dir)
	}
}

// deleteFile deletes a file from the server
func (s *InteractiveSession) deleteFile(file string) {
	code, msg, err := s.client.sendCommand(fmt.Sprintf("DELE %s", file))
	if err != nil {
		fmt.Printf("Error deleting file: %s\n", err)
		return
	}

	if code != 250 {
		fmt.Printf("Failed to delete file: %d %s\n", code, msg)
	} else {
		fmt.Printf("File deleted: %s\n", file)
	}
}

// StartShell connects to an FTP server and starts an interactive session
func StartShell(connStr string) error {
	// Parse the connection string
	host, port, user, pass := parseConnectionString(connStr)

	// Connect to the server
	client, err := Connect(host, port)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Login
	err = client.Login(user, pass)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Start the interactive session
	session := NewInteractiveSession(client)
	return session.Start()
}

// parseConnectionString parses a connection string which could be:
// - ftp://user:pass@host:port
// - user:pass@host:port
// - host:port
// - host
func parseConnectionString(connStr string) (host string, port int, user string, pass string) {
	// Default values
	port = 21
	user = "anonymous"
	pass = "guest@"

	// Check if it's a full URL or just a host
	if !strings.HasPrefix(connStr, "ftp://") {
		// If it contains @ symbol, it has credentials
		if strings.Contains(connStr, "@") {
			connStr = "ftp://" + connStr
		} else {
			// Just a hostname or hostname:port
			connStr = "ftp://" + user + ":" + pass + "@" + connStr
		}
	}

	// Now parse as a standard URL
	u, err := url.Parse(connStr)
	if err != nil {
		host = connStr
		return
	}

	host = u.Hostname()

	if u.Port() != "" {
		portNum, err := strconv.Atoi(u.Port())
		if err == nil {
			port = portNum
		}
	}

	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			pass = p
		}
	}

	return
}
