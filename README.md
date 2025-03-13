# UltraFTP

UltraFTP is a lightweight FTP server and client tool. It is completely static and does not require any external dependencies. 

## Features

- Single binary with both server and client functionality
- Standard FTP protocol implementation
- Interactive and inline FTP commands

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/titan/ultraftp.git
cd ultraftp

# Build the binary
go build -o ultraftp

# Make it executable
chmod +x ultraftp

# Optionally, move to a directory in your PATH
sudo mv ultraftp /usr/local/bin/
```

## Usage

### Server Mode

Start an FTP server on a specific port and serve a directory:

```bash
ultraftp server --port 2121 --dir /path/to/serve
```

Options:
- `--port`, `-p`: Port to listen on (default: 2121)
- `--dir`, `-d`: Directory to serve (default: current directory)

### Client Mode

#### Start an interactive FTP session

```bash
ultraftp client shell localhost
ultraftp client shell user:pass@remote-server
```

#### Download a file

```bash
ultraftp client get ftp://localhost:2121/file.txt local-file.txt
```

#### Upload a file

```bash
ultraftp client put local-file.txt ftp://localhost:2121/file.txt
```

### URL Format

The FTP URL format is:

```
ftp://[username[:password]@]host[:port]/path/to/file
```

Examples:
- `ftp://localhost:2121/file.txt` - Anonymous login
- `ftp://user:pass@localhost:2121/file.txt` - Authenticated login


