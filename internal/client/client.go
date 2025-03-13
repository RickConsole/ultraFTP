package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FTPClient represents an FTP client
type FTPClient struct {
	conn          net.Conn
	controlReader *bufio.Reader
	controlWriter *bufio.Writer
	dataConn      net.Conn
	host          string
	port          int
	user          string
	password      string
}

// Connect establishes a connection to an FTP server
func Connect(host string, port int) (*FTPClient, error) {
	// Connect to the server
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	client := &FTPClient{
		conn:          conn,
		controlReader: bufio.NewReader(conn),
		controlWriter: bufio.NewWriter(conn),
		host:          host,
		port:          port,
		user:          "anonymous", // Default to anonymous login
		password:      "guest@",
	}

	// Read the welcome message
	_, _, err = client.readResponse()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error reading welcome message: %w", err)
	}

	return client, nil
}

// Login authenticates with the FTP server
func (c *FTPClient) Login(user, password string) error {
	c.user = user
	c.password = password

	// Send USER command
	code, msg, err := c.sendCommand(fmt.Sprintf("USER %s", user))
	if err != nil {
		return err
	}

	if code == 230 {
		// User logged in without needing a password
		return nil
	}

	if code != 331 {
		return fmt.Errorf("unexpected response to USER command: %d %s", code, msg)
	}

	// Send PASS command
	code, msg, err = c.sendCommand(fmt.Sprintf("PASS %s", password))
	if err != nil {
		return err
	}

	if code != 230 {
		return fmt.Errorf("login failed: %d %s", code, msg)
	}

	return nil
}

// Close closes the connection to the FTP server
func (c *FTPClient) Close() error {
	if c.dataConn != nil {
		c.dataConn.Close()
	}
	
	// Send QUIT command
	_, _, err := c.sendCommand("QUIT")
	if err != nil {
		return err
	}
	
	return c.conn.Close()
}

// Get downloads a file from the FTP server
func Get(url string, localPath string) error {
	// Parse the URL
	ftpURL, err := parseURL(url)
	if err != nil {
		return err
	}

	// Connect to the server
	client, err := Connect(ftpURL.host, ftpURL.port)
	if err != nil {
		return err
	}
	defer client.Close()

	// Login
	err = client.Login(ftpURL.user, ftpURL.password)
	if err != nil {
		return err
	}

	// Change to the directory if needed
	if ftpURL.path != "" && filepath.Dir(ftpURL.path) != "." && filepath.Dir(ftpURL.path) != "/" {
		dirPath := filepath.Dir(ftpURL.path)
		_, _, err = client.sendCommand(fmt.Sprintf("CWD %s", dirPath))
		if err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Set binary mode
	_, _, err = client.sendCommand("TYPE I")
	if err != nil {
		return fmt.Errorf("failed to set binary mode: %w", err)
	}

	// Enter passive mode
	err = client.enterPassiveMode()
	if err != nil {
		return fmt.Errorf("failed to enter passive mode: %w", err)
	}

	// Send RETR command
	filename := filepath.Base(ftpURL.path)
	code, msg, err := client.sendCommand(fmt.Sprintf("RETR %s", filename))
	if err != nil {
		return err
	}

	if code != 150 && code != 125 {
		return fmt.Errorf("failed to retrieve file: %d %s", code, msg)
	}

	// Create the local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy the data
	_, err = io.Copy(file, client.dataConn)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}

	// Close the data connection
	client.dataConn.Close()
	client.dataConn = nil

	// Read the transfer complete message
	code, msg, err = client.readResponse()
	if err != nil {
		return err
	}

	if code != 226 && code != 250 {
		return fmt.Errorf("unexpected response after transfer: %d %s", code, msg)
	}

	return nil
}

// Put uploads a file to the FTP server
func Put(localPath string, url string) error {
	// Parse the URL
	ftpURL, err := parseURL(url)
	if err != nil {
		return err
	}

	// Connect to the server
	client, err := Connect(ftpURL.host, ftpURL.port)
	if err != nil {
		return err
	}
	defer client.Close()

	// Login
	err = client.Login(ftpURL.user, ftpURL.password)
	if err != nil {
		return err
	}

	// Change to the directory if needed
	if ftpURL.path != "" && filepath.Dir(ftpURL.path) != "." && filepath.Dir(ftpURL.path) != "/" {
		dirPath := filepath.Dir(ftpURL.path)
		_, _, err = client.sendCommand(fmt.Sprintf("CWD %s", dirPath))
		if err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Set binary mode
	_, _, err = client.sendCommand("TYPE I")
	if err != nil {
		return fmt.Errorf("failed to set binary mode: %w", err)
	}

	// Enter passive mode
	err = client.enterPassiveMode()
	if err != nil {
		return fmt.Errorf("failed to enter passive mode: %w", err)
	}

	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Send STOR command
	filename := filepath.Base(ftpURL.path)
	code, msg, err := client.sendCommand(fmt.Sprintf("STOR %s", filename))
	if err != nil {
		return err
	}

	if code != 150 && code != 125 {
		return fmt.Errorf("failed to store file: %d %s", code, msg)
	}

	// Copy the data
	_, err = io.Copy(client.dataConn, file)
	if err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	// Close the data connection
	client.dataConn.Close()
	client.dataConn = nil

	// Read the transfer complete message
	code, msg, err = client.readResponse()
	if err != nil {
		return err
	}

	if code != 226 && code != 250 {
		return fmt.Errorf("unexpected response after transfer: %d %s", code, msg)
	}

	return nil
}

// sendCommand sends a command to the FTP server and reads the response
func (c *FTPClient) sendCommand(command string) (int, string, error) {
	// Send the command
	fmt.Printf("> %s\n", command)
	_, err := c.controlWriter.WriteString(command + "\r\n")
	if err != nil {
		return 0, "", err
	}
	err = c.controlWriter.Flush()
	if err != nil {
		return 0, "", err
	}

	// Read the response
	return c.readResponse()
}

// readResponse reads a response from the FTP server
func (c *FTPClient) readResponse() (int, string, error) {
	// Read the response line
	line, err := c.controlReader.ReadString('\n')
	if err != nil {
		return 0, "", err
	}

	fmt.Printf("< %s", line)

	// Parse the response code
	if len(line) < 4 || line[3] != ' ' && line[3] != '-' {
		return 0, "", fmt.Errorf("invalid response format: %s", line)
	}

	code, err := strconv.Atoi(line[:3])
	if err != nil {
		return 0, "", fmt.Errorf("invalid response code: %s", line[:3])
	}

	// Handle multi-line responses
	message := strings.TrimSpace(line[4:])
	if line[3] == '-' {
		// This is a multi-line response, read until we get a line with the same code and a space
		for {
			line, err := c.controlReader.ReadString('\n')
			if err != nil {
				return 0, "", err
			}
			fmt.Printf("< %s", line)

			if len(line) >= 4 && line[:3] == strconv.Itoa(code) && line[3] == ' ' {
				message += "\n" + strings.TrimSpace(line[4:])
				break
			}
			message += "\n" + strings.TrimSpace(line)
		}
	}

	return code, message, nil
}

// enterPassiveMode switches to passive mode and establishes a data connection
func (c *FTPClient) enterPassiveMode() error {
	// Close any existing data connection
	if c.dataConn != nil {
		c.dataConn.Close()
		c.dataConn = nil
	}

	// Send PASV command
	code, msg, err := c.sendCommand("PASV")
	if err != nil {
		return err
	}

	if code != 227 {
		return fmt.Errorf("passive mode failed: %d %s", code, msg)
	}

	// Parse the response to get the data connection address
	// The response format is: 227 Entering Passive Mode (h1,h2,h3,h4,p1,p2)
	start := strings.Index(msg, "(")
	end := strings.Index(msg, ")")
	if start == -1 || end == -1 {
		return fmt.Errorf("invalid PASV response format: %s", msg)
	}

	// Extract the IP and port
	parts := strings.Split(msg[start+1:end], ",")
	if len(parts) != 6 {
		return fmt.Errorf("invalid PASV response format: %s", msg)
	}

	// Convert the parts to integers
	nums := make([]int, 6)
	for i, p := range parts {
		num, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("invalid PASV response format: %s", msg)
		}
		nums[i] = num
	}

	// Construct the IP address and port
	ip := fmt.Sprintf("%d.%d.%d.%d", nums[0], nums[1], nums[2], nums[3])
	port := nums[4]*256 + nums[5]

	// Connect to the data port
	addr := fmt.Sprintf("%s:%d", ip, port)
	dataConn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to data port: %w", err)
	}

	c.dataConn = dataConn
	return nil
}

// FTPURL represents a parsed FTP URL
type FTPURL struct {
	host     string
	port     int
	user     string
	password string
	path     string
}

// parseURL parses an FTP URL
func parseURL(rawURL string) (FTPURL, error) {
	result := FTPURL{
		port:     21,
		user:     "anonymous",
		password: "guest@",
	}

	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return result, fmt.Errorf("invalid URL: %w", err)
	}

	// Check the scheme
	if u.Scheme != "ftp" {
		return result, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Extract the host and port
	result.host = u.Hostname()
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return result, fmt.Errorf("invalid port: %s", u.Port())
		}
		result.port = port
	}

	// Extract the user and password
	if u.User != nil {
		result.user = u.User.Username()
		if password, ok := u.User.Password(); ok {
			result.password = password
		}
	}

	// Extract the path
	result.path = u.Path
	if result.path != "" && result.path[0] == '/' {
		result.path = result.path[1:]
	}

	return result, nil
}
