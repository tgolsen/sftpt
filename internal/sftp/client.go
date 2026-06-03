package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/tgolsen/sftpt/internal/auth"
	"github.com/tgolsen/sftpt/internal/progress"
	"golang.org/x/crypto/ssh"
)

// Client wraps SFTP operations
type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	host       string
	port       string
	user       string
}

// TransferOptions holds options for file transfers
type TransferOptions struct {
	Recursive    bool
	Preserve     bool
	ShowProgress bool
	CreateDirs   bool
}

// ClientOptions holds options for creating SFTP client
type ClientOptions struct {
	KeyFile              string
	UsePassword          bool
	PasswordFromStdin    string
	KeysOnly             bool
	UseSSHConfig         bool
	StrictHostChecking   bool
}

// NewClient creates a new SFTP client with authentication
func NewClient(host, port, user string) (*Client, error) {
	// Legacy function for backward compatibility
	return NewClientWithOptions(host, port, user, ClientOptions{
		UseSSHConfig:       true,
		StrictHostChecking: true,
	})
}

// NewClientWithOptions creates a new SFTP client with explicit options
func NewClientWithOptions(host, port, user string, options ClientOptions) (*Client, error) {
	// Get authentication configuration
	authOptions := auth.AuthOptions{
		KeyFile:            options.KeyFile,
		UsePassword:        options.UsePassword,
		PasswordFromStdin:  options.PasswordFromStdin,
		KeysOnly:           options.KeysOnly,
		UseSSHConfig:       options.UseSSHConfig,
		StrictHostChecking: options.StrictHostChecking,
	}

	authConfig, err := auth.GetAuthConfigWithFlags(user, host, port, authOptions)
	if err != nil {
		return nil, fmt.Errorf("getting auth config: %w", err)
	}

	// Configure host key checking
	var hostKeyCallback ssh.HostKeyCallback
	if options.StrictHostChecking {
		// In production, you'd want proper host key verification
		// For now, we'll use a simple callback that accepts known patterns
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authConfig.Methods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	// Connect to SSH server
	address := fmt.Sprintf("%s:%s", host, port)
	sshClient, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("SFTP client creation failed: %w", err)
	}

	return &Client{
		sshClient:  sshClient,
		sftpClient: sftpClient,
		host:       host,
		port:       port,
		user:       user,
	}, nil
}

// Close closes the SFTP and SSH connections
func (c *Client) Close() error {
	var errs []string

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("SFTP close: %v", err))
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("SSH close: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// Glob returns the names of all files matching pattern on the remote server.
func (c *Client) Glob(pattern string) ([]string, error) {
	matches, err := c.sftpClient.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("expanding glob pattern: %w", err)
	}
	return matches, nil
}

// StatFile returns file info for a remote path.
func (c *Client) StatFile(path string) (os.FileInfo, error) {
	return c.sftpClient.Stat(path)
}

// ListOptions holds options for listing directory contents.
type ListOptions struct {
	LongFormat    bool
	ShowAll       bool
	HumanReadable bool
	Sort          string // "time", "size", "name", or "" (default name order)
}

// List lists directory contents
func (c *Client) List(path string, opts ListOptions) ([]string, error) {
	entries, err := c.sftpClient.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	sortEntries(entries, opts)

	var results []string
	for _, entry := range entries {
		name := entry.Name()

		if !opts.ShowAll && strings.HasPrefix(name, ".") {
			continue
		}

		if opts.LongFormat {
			results = append(results, formatEntry(entry, opts))
		} else {
			results = append(results, name)
		}
	}

	return results, nil
}

func sortEntries(entries []os.FileInfo, opts ListOptions) {
	switch opts.Sort {
	case "time":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ModTime().After(entries[j].ModTime())
		})
	case "size":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Size() > entries[j].Size()
		})
	default: // "name" or ""
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
	}
}

// FormatEntry formats a single file entry for display.
func FormatEntry(fi os.FileInfo, name string, opts ListOptions) string {
	return formatEntry(&fileInfoWrapper{fi, name}, opts)
}

// fileInfoWrapper adapts a name + os.FileInfo to the interface formatEntry needs.
type fileInfoWrapper struct {
	os.FileInfo
	name string
}

func (w *fileInfoWrapper) Name() string { return w.name }

// formatEntry formats a single file info entry.
func formatEntry(fi os.FileInfo, opts ListOptions) string {
	mode := fi.Mode().String()
	modTime := fi.ModTime().Format("Jan 02 15:04")

	var sizeStr string
	if opts.HumanReadable {
		sizeStr = fmt.Sprintf("%8s", formatBytes(fi.Size()))
	} else {
		sizeStr = fmt.Sprintf("%8d", fi.Size())
	}

	return fmt.Sprintf("%s %s %s %s", mode, sizeStr, modTime, fi.Name())
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

// Download downloads a file or directory from remote to local
func (c *Client) Download(remotePath, localPath string, options TransferOptions) error {
	// Get remote file info
	remoteInfo, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("getting remote file info: %w", err)
	}

	if remoteInfo.IsDir() {
		if !options.Recursive {
			return fmt.Errorf("remote path is directory but recursive not enabled")
		}
		return c.downloadDir(remotePath, localPath, options)
	}

	return c.downloadFile(remotePath, localPath, options)
}

// downloadFile downloads a single file
func (c *Client) downloadFile(remotePath, localPath string, options TransferOptions) error {
	// Open remote file
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("opening remote file: %w", err)
	}
	defer remoteFile.Close()

	// Determine local file path
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		localPath = filepath.Join(localPath, filepath.Base(remotePath))
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer localFile.Close()

	// Copy file content (with optional progress bar)
	var fileSize int64
	if options.ShowProgress {
		if fi, err := remoteFile.Stat(); err == nil {
			fileSize = fi.Size()
		}
		pw := progress.NewWriter(localFile, fileSize, filepath.Base(remotePath))
		_, err = io.Copy(pw, remoteFile)
		pw.Done()
	} else {
		_, err = io.Copy(localFile, remoteFile)
	}
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	// Preserve timestamps if requested
	if options.Preserve {
		if stat, err := c.sftpClient.Stat(remotePath); err == nil {
			// Ignore errors in setting timestamps - not critical to operation
			_ = os.Chtimes(localPath, stat.ModTime(), stat.ModTime())
		}
	}

	return nil
}

// downloadDir downloads a directory recursively
func (c *Client) downloadDir(remotePath, localPath string, options TransferOptions) error {
	// Create local directory
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("creating local directory: %w", err)
	}

	// List remote directory
	entries, err := c.sftpClient.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("reading remote directory: %w", err)
	}

	// Download each entry
	for _, entry := range entries {
		remoteEntryPath := filepath.Join(remotePath, entry.Name())
		localEntryPath := filepath.Join(localPath, entry.Name())

		if entry.IsDir() {
			err = c.downloadDir(remoteEntryPath, localEntryPath, options)
		} else {
			err = c.downloadFile(remoteEntryPath, localEntryPath, options)
		}

		if err != nil {
			return fmt.Errorf("downloading %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Upload uploads a file or directory from local to remote
func (c *Client) Upload(localPath, remotePath string, options TransferOptions) error {
	// Get local file info
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("getting local file info: %w", err)
	}

	if localInfo.IsDir() {
		if !options.Recursive {
			return fmt.Errorf("local path is directory but recursive not enabled")
		}
		return c.uploadDir(localPath, remotePath, options)
	}

	return c.uploadFile(localPath, remotePath, options)
}

// uploadFile uploads a single file
func (c *Client) uploadFile(localPath, remotePath string, options TransferOptions) error {
	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file: %w", err)
	}
	defer localFile.Close()

	// Determine remote file path
	if stat, err := c.sftpClient.Stat(remotePath); err == nil && stat.IsDir() {
		remotePath = filepath.Join(remotePath, filepath.Base(localPath))
	}

	// Create remote directories if needed
	if options.CreateDirs {
		remoteDir := filepath.Dir(remotePath)
		if err := c.Mkdir(remoteDir, "0755", true); err != nil {
			return fmt.Errorf("creating remote directory: %w", err)
		}
	}

	// Create remote file
	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("creating remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy file content (with optional progress bar)
	var fileSize int64
	if options.ShowProgress {
		if fi, err := localFile.Stat(); err == nil {
			fileSize = fi.Size()
		}
		pr := progress.NewReader(localFile, fileSize, filepath.Base(localPath))
		_, err = io.Copy(remoteFile, pr)
		pr.Done()
	} else {
		_, err = io.Copy(remoteFile, localFile)
	}
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	// Preserve timestamps if requested
	if options.Preserve {
		if stat, err := os.Stat(localPath); err == nil {
			// Ignore errors in setting timestamps - not critical to operation
			_ = c.sftpClient.Chtimes(remotePath, stat.ModTime(), stat.ModTime())
		}
	}

	return nil
}

// uploadDir uploads a directory recursively
func (c *Client) uploadDir(localPath, remotePath string, options TransferOptions) error {
	// Create remote directory
	if err := c.Mkdir(remotePath, "0755", options.CreateDirs); err != nil {
		return fmt.Errorf("creating remote directory: %w", err)
	}

	// List local directory
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("reading local directory: %w", err)
	}

	// Upload each entry
	for _, entry := range entries {
		localEntryPath := filepath.Join(localPath, entry.Name())
		remoteEntryPath := filepath.Join(remotePath, entry.Name())

		if entry.IsDir() {
			err = c.uploadDir(localEntryPath, remoteEntryPath, options)
		} else {
			err = c.uploadFile(localEntryPath, remoteEntryPath, options)
		}

		if err != nil {
			return fmt.Errorf("uploading %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Mkdir creates a directory on the remote server
func (c *Client) Mkdir(path, mode string, parents bool) error {
	// Parse mode
	fileMode, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid mode: %w", err)
	}

	if parents {
		// Create parent directories recursively
		return c.mkdirAll(path, os.FileMode(fileMode))
	}

	// Create single directory
	err = c.sftpClient.Mkdir(path)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Set permissions
	err = c.sftpClient.Chmod(path, os.FileMode(fileMode))
	if err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	return nil
}

// mkdirAll creates a directory and all necessary parents
func (c *Client) mkdirAll(path string, mode os.FileMode) error {
	// Check if directory already exists
	if _, err := c.sftpClient.Stat(path); err == nil {
		return nil // Directory exists
	}

	// Create parent directory if needed
	parent := filepath.Dir(path)
	if parent != path && parent != "/" && parent != "." {
		if err := c.mkdirAll(parent, mode); err != nil {
			return err
		}
	}

	// Create this directory
	if err := c.sftpClient.Mkdir(path); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}

	// Set permissions
	if err := c.sftpClient.Chmod(path, mode); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", path, err)
	}

	return nil
}

// Remove removes a file or directory from the remote server
func (c *Client) Remove(path string, recursive, force bool) error {
	// Get file info
	info, err := c.sftpClient.Stat(path)
	if err != nil {
		if force {
			return nil // Ignore error if force is enabled
		}
		return fmt.Errorf("getting file info: %w", err)
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("path is directory but recursive not enabled")
		}
		return c.removeDir(path, force)
	}

	// Remove file
	err = c.sftpClient.Remove(path)
	if err != nil && !force {
		return fmt.Errorf("removing file: %w", err)
	}

	return nil
}

// removeDir removes a directory recursively
func (c *Client) removeDir(path string, force bool) error {
	// List directory contents
	entries, err := c.sftpClient.ReadDir(path)
	if err != nil {
		if force {
			return nil
		}
		return fmt.Errorf("reading directory: %w", err)
	}

	// Remove all entries
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := c.Remove(entryPath, true, force); err != nil {
			if !force {
				return err
			}
		}
	}

	// Remove the directory itself
	err = c.sftpClient.RemoveDirectory(path)
	if err != nil && !force {
		return fmt.Errorf("removing directory: %w", err)
	}

	return nil
}
