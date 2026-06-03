# sftpt

A simple command-line SFTP utility for Mac, designed for easy integration with shell scripts.

## Quick Start

```bash
go install github.com/tgolsen/sftpt/cmd/sftpt@latest
sftpt list myserver:/var/log
```

## Purpose

`sftpt` provides a streamlined interface to SFTP operations with focus on:
- **Scriptability**: Simple commands that work reliably in automated scripts
- **Mac Integration**: Optimized for macOS environments and workflows
- **Minimal Dependencies**: Lightweight tool that doesn't require complex setup

## Installation

```bash
# Installation method TBD (when built)
# Options: brew install, direct binary download, or build from source
```

## Usage

### Basic Commands

```bash
# Connect and list remote directory
sftpt list user@host:/remote/path

# Download file
sftpt get user@host:/remote/file.txt ./local/path/

# Upload file
sftpt put ./local/file.txt user@host:/remote/path/

# Sync directory
sftpt sync ./local/dir/ user@host:/remote/dir/
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
# SSH key authentication (recommended for scripts)
sftpt get user@host:/file.txt --key ~/.ssh/id_rsa

# Password authentication (interactive)
sftpt get user@host:/file.txt --password

# Configuration file support
sftpt get user@host:/file.txt --config ~/.sftpt/config
```

## Development

### Prerequisites

- macOS development environment
- [Language-specific requirements TBD]

### Setup

```bash
# Clone repository
git clone [repository-url]
cd sftpt

# Copy environment template
cp .env.example .env

# Install development dependencies
# [Command TBD based on chosen language]
```

### Development Commands

```bash
# Build CLI tool
# [Build command TBD]

# Run tests
# [Test command TBD]

# Install locally for testing
# [Install command TBD]

# Run linting
# [Lint command TBD]
```

### Testing SFTP Functionality

```bash
# Test with local SFTP server (for development)
# Set up test SFTP server in .env:
# TEST_SFTP_HOST=localhost
# TEST_SFTP_USER=testuser
# TEST_SFTP_PATH=/tmp/sftpt-test

# Run integration tests
# [Integration test command TBD]
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
├── .agent/                 # Development guidelines and process
├── src/                    # Source code (TBD)
├── tests/                  # Test files (TBD)
├── docs/                   # Additional documentation
├── .env.example           # Development environment template
└── README.md              # This file
```

*Note: This is a new project. Implementation language and specific build commands will be determined during initial development.*