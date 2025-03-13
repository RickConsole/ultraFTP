package server

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// FTPServer represents an FTP server instance
type FTPServer struct {
	Port       int
	RootDir    string
	listener   net.Listener
	sessions   map[string]*Session
	sessionsMu sync.Mutex
}

// Session represents a client session
type Session struct {
	conn          net.Conn
	controlReader *bufio.Reader
	controlWriter *bufio.Writer
	dataConn      net.Conn
	workDir       string
	authenticated bool
}

// Start initializes and starts the FTP server
func Start(port int, rootDir string) error {
	// Resolve the root directory to an absolute path
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}

	// Check if the directory exists
	info, err := os.Stat(absRootDir)
	if err != nil {
		return fmt.Errorf("cannot access root directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", absRootDir)
	}

	// Create and initialize the server
	server := &FTPServer{
		Port:     port,
		RootDir:  absRootDir,
		sessions: make(map[string]*Session),
	}

	// Start listening for connections
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}
	server.listener = listener

	fmt.Printf("FTP Server listening on port %d, serving directory: %s\n", port, absRootDir)

	// Accept and handle client connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each client in a separate goroutine
		go server.handleClient(conn)
	}
}

// handleClient processes a client connection
func (s *FTPServer) handleClient(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("New connection from %s\n", clientAddr)

	// Create a new session for this client
	session := &Session{
		conn:          conn,
		controlReader: bufio.NewReader(conn),
		controlWriter: bufio.NewWriter(conn),
		workDir:       "/",
		authenticated: false, // We'll use a simple authentication mechanism
	}

	// Register the session
	s.sessionsMu.Lock()
	s.sessions[clientAddr] = session
	s.sessionsMu.Unlock()

	// Clean up when the client disconnects
	defer func() {
		s.sessionsMu.Lock()
		delete(s.sessions, clientAddr)
		s.sessionsMu.Unlock()
		if session.dataConn != nil {
			session.dataConn.Close()
		}
	}()

	// Send welcome message
	session.writeResponse(220, "UltraFTP Server ready")

	// Process client commands
	for {
		line, err := session.controlReader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading from client: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse the command
		parts := strings.SplitN(line, " ", 2)
		command := strings.ToUpper(parts[0])
		var param string
		if len(parts) > 1 {
			param = parts[1]
		}

		// Handle the command
		if !s.handleCommand(session, command, param) {
			break
		}
	}

	fmt.Printf("Connection from %s closed\n", clientAddr)
}

// handleCommand processes an FTP command
func (s *FTPServer) handleCommand(session *Session, command, param string) bool {
	fmt.Printf("Command: %s %s\n", command, param)

	switch command {
	case "USER":
		// For simplicity, we'll accept any username
		session.writeResponse(331, "User name okay, need password")
	case "PASS":
		// For simplicity, we'll accept any password
		session.authenticated = true
		session.writeResponse(230, "User logged in, proceed")
	case "SYST":
		session.writeResponse(215, "UNIX Type: L8")
	case "FEAT":
		session.writeMultiResponse(211, []string{
			"Features:",
			" UTF8",
			"End",
		})
	case "PWD":
		session.writeResponse(257, fmt.Sprintf("\"%s\" is the current directory", session.workDir))
	case "TYPE":
		// We'll support both ASCII and binary mode, but won't differentiate
		session.writeResponse(200, "Type set to "+param)
	case "PASV":
		s.handlePassive(session)
	case "PORT":
		s.handlePort(session, param)
	case "LIST":
		if !session.authenticated {
			session.writeResponse(530, "Not logged in")
			return true
		}
		s.handleList(session, param)
	case "RETR":
		if !session.authenticated {
			session.writeResponse(530, "Not logged in")
			return true
		}
		s.handleRetrieve(session, param)
	case "STOR":
		if !session.authenticated {
			session.writeResponse(530, "Not logged in")
			return true
		}
		s.handleStore(session, param)
	case "CWD":
		if !session.authenticated {
			session.writeResponse(530, "Not logged in")
			return true
		}
		s.handleChangeDir(session, param)
	case "CDUP":
		if !session.authenticated {
			session.writeResponse(530, "Not logged in")
			return true
		}
		s.handleChangeDir(session, "..")
	case "QUIT":
		session.writeResponse(221, "Goodbye")
		return false
	default:
		session.writeResponse(502, "Command not implemented")
	}

	return true
}

// writeResponse sends a response to the client
func (s *Session) writeResponse(code int, message string) {
	response := fmt.Sprintf("%d %s\r\n", code, message)
	s.controlWriter.WriteString(response)
	s.controlWriter.Flush()
}

// writeMultiResponse sends a multi-line response to the client
func (s *Session) writeMultiResponse(code int, messages []string) {
	// First line
	s.controlWriter.WriteString(fmt.Sprintf("%d-%s\r\n", code, messages[0]))
	
	// Middle lines
	for i := 1; i < len(messages)-1; i++ {
		s.controlWriter.WriteString(fmt.Sprintf(" %s\r\n", messages[i]))
	}
	
	// Last line
	s.controlWriter.WriteString(fmt.Sprintf("%d %s\r\n", code, messages[len(messages)-1]))
	s.controlWriter.Flush()
}

// handlePassive handles the PASV command
func (s *FTPServer) handlePassive(session *Session) {
	// Close any existing data connection
	if session.dataConn != nil {
		session.dataConn.Close()
		session.dataConn = nil
	}

	// Create a listener for the data connection
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		session.writeResponse(425, "Cannot open data connection")
		return
	}

	// Get the port that was assigned
	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		listener.Close()
		session.writeResponse(425, "Cannot open data connection")
		return
	}

	port, _ := strconv.Atoi(portStr)
	p1 := port / 256
	p2 := port % 256

	// Get the local IP address
	host, _, _ := net.SplitHostPort(session.conn.LocalAddr().String())
	hostParts := strings.Split(host, ".")

	// Send the passive mode response
	response := fmt.Sprintf("Entering Passive Mode (%s,%s,%s,%s,%d,%d)",
		hostParts[0], hostParts[1], hostParts[2], hostParts[3], p1, p2)
	session.writeResponse(227, response)

	// Accept the data connection in a goroutine
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting data connection: %v\n", err)
			return
		}
		session.dataConn = conn
	}()
}

// handlePort handles the PORT command
func (s *FTPServer) handlePort(session *Session, param string) {
	// Close any existing data connection
	if session.dataConn != nil {
		session.dataConn.Close()
		session.dataConn = nil
	}

	// Parse the PORT command parameters
	parts := strings.Split(param, ",")
	if len(parts) != 6 {
		session.writeResponse(501, "Invalid PORT command")
		return
	}

	// Extract the IP and port
	ip := strings.Join(parts[0:4], ".")
	p1, _ := strconv.Atoi(parts[4])
	p2, _ := strconv.Atoi(parts[5])
	port := p1*256 + p2

	// Connect to the client's data port
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		session.writeResponse(425, "Cannot open data connection")
		return
	}

	session.dataConn = conn
	session.writeResponse(200, "PORT command successful")
}

// handleList handles the LIST command
func (s *FTPServer) handleList(session *Session, param string) {
	if session.dataConn == nil {
		session.writeResponse(425, "Use PORT or PASV first")
		return
	}

	// Ensure the data connection is closed when we're done
	defer func() {
		session.dataConn.Close()
		session.dataConn = nil
	}()

	// Determine the directory to list
	path := param
	if path == "" || path == "-a" || path == "-l" {
		path = session.workDir
	}

	// Convert the path to an absolute path in the server's filesystem
	fullPath := filepath.Join(s.RootDir, filepath.Clean(path))

	// Check if the path exists and is a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		session.writeResponse(550, "File not found")
		return
	}

	// Notify the client that we're about to send the listing
	session.writeResponse(150, "Here comes the directory listing")

	// If it's a directory, list its contents
	if info.IsDir() {
		files, err := os.ReadDir(fullPath)
		if err != nil {
			session.writeResponse(550, "Error reading directory")
			return
		}

		// Send the directory listing
		writer := bufio.NewWriter(session.dataConn)
		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				continue
			}

			// Format: "-rw-r--r-- 1 owner group size month day time filename"
			mode := info.Mode().String()
			size := info.Size()
			modTime := info.ModTime().Format("Jan 02 15:04")
			name := info.Name()

			fmt.Fprintf(writer, "%s 1 owner group %d %s %s\r\n", mode, size, modTime, name)
		}
		writer.Flush()
	} else {
		// It's a file, just send its info
		writer := bufio.NewWriter(session.dataConn)
		mode := info.Mode().String()
		size := info.Size()
		modTime := info.ModTime().Format("Jan 02 15:04")
		name := info.Name()

		fmt.Fprintf(writer, "%s 1 owner group %d %s %s\r\n", mode, size, modTime, name)
		writer.Flush()
	}

	// Notify the client that the transfer is complete
	session.writeResponse(226, "Directory send OK")
}

// handleRetrieve handles the RETR command (download)
func (s *FTPServer) handleRetrieve(session *Session, param string) {
	if session.dataConn == nil {
		session.writeResponse(425, "Use PORT or PASV first")
		return
	}

	// Ensure the data connection is closed when we're done
	defer func() {
		session.dataConn.Close()
		session.dataConn = nil
	}()

	// Convert the path to an absolute path in the server's filesystem
	fullPath := filepath.Join(s.RootDir, filepath.Clean(param))

	// Check if the file exists
	file, err := os.Open(fullPath)
	if err != nil {
		session.writeResponse(550, "File not found")
		return
	}
	defer file.Close()

	// Get the file size
	info, err := file.Stat()
	if err != nil {
		session.writeResponse(550, "Error accessing file")
		return
	}

	// Notify the client that we're about to send the file
	session.writeResponse(150, fmt.Sprintf("Opening data connection for %s (%d bytes)", param, info.Size()))

	// Send the file
	_, err = bufio.NewReader(file).WriteTo(session.dataConn)
	if err != nil {
		fmt.Printf("Error sending file: %v\n", err)
		return
	}

	// Notify the client that the transfer is complete
	session.writeResponse(226, "Transfer complete")
}

// handleStore handles the STOR command (upload)
func (s *FTPServer) handleStore(session *Session, param string) {
	if session.dataConn == nil {
		session.writeResponse(425, "Use PORT or PASV first")
		return
	}

	// Ensure the data connection is closed when we're done
	defer func() {
		session.dataConn.Close()
		session.dataConn = nil
	}()

	// Convert the path to an absolute path in the server's filesystem
	fullPath := filepath.Join(s.RootDir, filepath.Clean(param))

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		session.writeResponse(550, "Cannot create file")
		return
	}
	defer file.Close()

	// Notify the client that we're ready to receive the file
	session.writeResponse(150, "Ok to send data")

	// Receive the file
	_, err = bufio.NewReader(session.dataConn).WriteTo(file)
	if err != nil {
		fmt.Printf("Error receiving file: %v\n", err)
		return
	}

	// Notify the client that the transfer is complete
	session.writeResponse(226, "Transfer complete")
}

// handleChangeDir handles the CWD command
func (s *FTPServer) handleChangeDir(session *Session, param string) {
	// Handle absolute paths
	var newPath string
	if strings.HasPrefix(param, "/") {
		newPath = param
	} else {
		// Handle relative paths
		newPath = filepath.Join(session.workDir, param)
	}

	// Clean up the path
	newPath = filepath.Clean(newPath)
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}

	// Convert to server filesystem path
	fullPath := filepath.Join(s.RootDir, newPath)

	// Check if the directory exists
	info, err := os.Stat(fullPath)
	if err != nil || !info.IsDir() {
		session.writeResponse(550, "Directory not found")
		return
	}

	// Update the working directory
	session.workDir = newPath
	session.writeResponse(250, "Directory successfully changed")
}
