# sftpt

A simple command-line SFTP utility for Mac, designed for easy integration with shell scripts.

## Quick Start

```bash
go install github.com/tgolsen/sftpt/cmd/sftpt@latest

# Use an SSH config entry for 'myserver' in ~/.ssh/config
sftpt list myserver:/var/log

# Or use a password
sftpt list user@myserver.com:/ --password
[password prompt]
```

## Purpose

`sftpt` provides a streamlined interface to SFTP operations with focus on:
- **Scriptability**: Simple commands that work reliably in automated scripts
- **Mac Integration**: Optimized for macOS environments and workflows
- **Minimal Dependencies**: Lightweight tool that doesn't require complex setup

## Installation

```bash
go install github.com/tgolsen/sftpt/cmd/sftpt@latest
```

Requires Go 1.26+. The binary installs to `$GOPATH/bin` (typically `~/go/bin`).

## Usage

### Basic Commands

```bash
# List remote directory
sftpt list myserver:/remote/path
sftpt list myserver:/remote/path -l --human-readable --sort time

# Download files (supports globs)
sftpt get myserver:/remote/file.txt ./local/path/
sftpt get "myserver:/var/log/*.log" ./logs/

# Upload files (shell expands local globs)
sftpt put ./local/file.txt myserver:/remote/path/
sftpt put *.log myserver:/remote/path/

# Create directory
sftpt mkdir myserver:/remote/newdir/

# Remove files (supports globs)
sftpt rm myserver:/remote/oldfile.txt
sftpt rm "myserver:/tmp/*.bak"

# Batch commands from file or inline
sftpt script --file deploy.txt
sftpt script --inline "list myserver:/logs/; get myserver:/logs/app.log ./"
```

### Shell Script Examples

```bash
#!/bin/bash
# Backup script example

REMOTE_HOST="backup@server.com"
BACKUP_DIR="/backups/$(date +%Y-%m-%d)"

# Create remote backup directory
sftpt mkdir "$REMOTE_HOST:$BACKUP_DIR"

# Upload files
for file in ./data/*; do
    sftpt put "$file" "$REMOTE_HOST:$BACKUP_DIR/"
done

# Verify upload
sftpt list "$REMOTE_HOST:$BACKUP_DIR"
```

```bash
#!/bin/bash
# Download and process files

REMOTE_HOST="data@server.com"
REMOTE_PATH="/incoming/*.csv"

# Download all CSV files
sftpt get "$REMOTE_HOST:$REMOTE_PATH" ./processing/

# Process downloaded files
for file in ./processing/*.csv; do
    # Process file
    echo "Processing $file"
done
```

## Authentication

```bash
# SSH config resolution (automatic — no flags needed)
sftpt list myserver:/path

# SSH key authentication (specific key)
sftpt get myserver:/file.txt --key ~/.ssh/id_rsa

# Password authentication (interactive prompt)
sftpt get myserver:/file.txt --password

# Password from stdin (for scripts)
echo "$PASSWORD" | sftpt get myserver:/file.txt --password-stdin "$(cat)"
```

## Development

### Prerequisites

- Go 1.26+
- Docker (for integration tests with test SFTP server)

### Setup

```bash
git clone https://github.com/tgolsen/sftpt.git
cd sftpt
```

### Development Commands

```bash
make build       # Build binary
make test        # Run unit tests
make coverage    # Run tests with coverage report
make lint        # Run golangci-lint
make format      # Format code
make dev         # Quick build for development
make quality     # Run all checks (deps, format, lint, security, test)
```

### Testing SFTP Functionality

```bash
# Start test SFTP server
docker compose -f docker-compose.test.yml up -d

# Run with test server
./build/sftpt list testuser@localhost:2222:/upload --password-stdin password

# Stop test server
docker rm -f sftpt-test-server
```

## CLI Design Principles

- **Exit codes**: Proper exit codes for script integration (0=success, non-zero=error)
- **Output format**: Clean output that's easy to parse in scripts
- **Error handling**: Clear error messages and graceful failure handling
- **Configuration**: Support for config files to avoid repeating connection details
- **Security**: SSH key authentication, no plain-text password storage

## Planned Features

- [x] Basic SFTP operations (get, put, list, mkdir, rm)
- [x] Batch operations and wildcard support
- [x] Progress indicators for large transfers
- [ ] Resume interrupted transfers
- [ ] Configuration file support
- [x] SSH key and password authentication
- [x] Verbose and quiet modes for different script needs
- [ ] Mac-specific integrations (Keychain, file notifications)

## Contributing

This project follows milestone-driven development. See `.agent/milestone-process.md` for the development workflow.

### Development Guidelines

- Follow CLI best practices for exit codes and output formatting
- Include tests for SFTP operations
- Consider shell script integration in design decisions
- Test on macOS environments

### Git Workflow

```bash
# Security-focused git workflow
git status                    # Review changes
git add src/specific-file.js  # Stage specific files (never git add .)
git diff --staged             # Review staged changes
git commit -m "feat: add command"
```

## Project Structure

```
sftpt/
├── cmd/sftpt/              # Entry point
├── internal/
│   ├── auth/               # SSH authentication
│   ├── commands/           # CLI subcommands (get, put, list, rm, mkdir, script)
│   ├── progress/           # Terminal progress bar
│   └── sftp/               # SFTP client wrapper
├── test/
│   ├── downloads/          # Download test destination
│   └── testdata/           # Test fixtures
├── docker-compose.test.yml # Test SFTP server
├── Makefile                # Build, test, lint targets
└── README.md
```
