package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// ConnectionInfo holds parsed connection details
type ConnectionInfo struct {
	User string
	Host string
	Port string
	Path string
}

// ParseConnectionString parses connection strings in the format [user@]host[:port]:[path]
// Also supports SSH config aliases like: config-host:/path
func ParseConnectionString(connStr string) (*ConnectionInfo, error) {
	if connStr == "" {
		return nil, fmt.Errorf("empty connection string")
	}

	// Split on the last colon to separate path
	lastColon := strings.LastIndex(connStr, ":")
	if lastColon == -1 {
		return nil, fmt.Errorf("invalid connection string format: missing path")
	}

	hostPart := connStr[:lastColon]
	path := connStr[lastColon+1:]

	// Parse user@host:port part
	var user, host, port string
	var userHost string

	if atIndex := strings.LastIndex(hostPart, "@"); atIndex != -1 {
		user = hostPart[:atIndex]
		userHost = hostPart[atIndex+1:]
	} else {
		userHost = hostPart
	}

	// Parse host:port
	if colonIndex := strings.LastIndex(userHost, ":"); colonIndex != -1 {
		host = userHost[:colonIndex]
		port = userHost[colonIndex+1:]
	} else {
		host = userHost
		port = "22" // default SSH port
	}

	// Check if host might be an SSH config alias
	if user == "" && port == "22" && !strings.Contains(host, ".") {
		// Looks like it might be an SSH config alias, try to resolve it
		if sshConfig := tryResolveSSHConfig(host); sshConfig != nil {
			if sshConfig.User != "" {
				user = sshConfig.User
			}
			if sshConfig.HostName != "" {
				host = sshConfig.HostName
			}
			if sshConfig.Port != "" {
				port = sshConfig.Port
			}
		}
	}

	// Validate required fields
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	return &ConnectionInfo{
		User: user,
		Host: host,
		Port: port,
		Path: path,
	}, nil
}

// BaseCommand provides common functionality for all SFTP commands
// Currently unused but kept for future enhancements
type BaseCommand struct {
	// cmd      *cobra.Command  // Removed unused field
	// connInfo *ConnectionInfo // Removed unused field
}

// NewBaseCommand creates a new base command with common setup
func NewBaseCommand(use, short, long string, runFunc func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		RunE:  runFunc,
	}

	return cmd
}

// ValidateArgs ensures we have exactly one connection string argument
func ValidateConnectionArg(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one connection string required")
	}

	connInfo, err := ParseConnectionString(args[0])
	if err != nil {
		return fmt.Errorf("invalid connection string: %w", err)
	}

	// Store connection info for use in command execution
	cmd.SetContext(cmd.Context())
	// We'll need to pass this through execution context or command flags
	_ = connInfo

	return nil
}

// GetConnectionInfo extracts connection info from command arguments
func GetConnectionInfo(args []string) (*ConnectionInfo, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("connection string required")
	}
	return ParseConnectionString(args[0])
}

// ExitWithError prints error and exits with code 1 (for script integration)
func ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "sftpt: %v\n", err)
	os.Exit(1)
}

// PrintVerbose prints verbose output only if verbose flag is set
func PrintVerbose(cmd *cobra.Command, format string, args ...interface{}) {
	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// PrintQuiet prints output only if quiet flag is not set
func PrintOutput(cmd *cobra.Command, format string, args ...interface{}) {
	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf(format, args...)
	}
}

// GetSFTPClientOptions extracts SFTP client options from command flags
// If SSH config was used to resolve connection info, it may override keyFile
func GetSFTPClientOptions(cmd *cobra.Command) sftp.ClientOptions {
	keyFile, _ := cmd.Flags().GetString("key")
	usePassword, _ := cmd.Flags().GetBool("password")
	passwordStdin, _ := cmd.Flags().GetString("password-stdin")
	keysOnly, _ := cmd.Flags().GetBool("keys-only")
	useSSHConfig, _ := cmd.Flags().GetBool("ssh-config")
	strictHostChecking, _ := cmd.Flags().GetBool("strict-host-checking")

	return sftp.ClientOptions{
		KeyFile:            keyFile,
		UsePassword:        usePassword,
		PasswordFromStdin:  passwordStdin,
		KeysOnly:           keysOnly,
		UseSSHConfig:       useSSHConfig,
		StrictHostChecking: strictHostChecking,
	}
}

// GetSFTPClientOptionsWithSSHConfig gets options and applies SSH config overrides
func GetSFTPClientOptionsWithSSHConfig(cmd *cobra.Command, connInfo *ConnectionInfo, hostAlias string) sftp.ClientOptions {
	options := GetSFTPClientOptions(cmd)

	// If no explicit key file was provided and SSH config is enabled, try to get key from SSH config
	if options.KeyFile == "" && options.UseSSHConfig {
		if sshConfig := tryResolveSSHConfig(hostAlias); sshConfig != nil && sshConfig.KeyFile != "" {
			// Expand ~ to home directory
			if strings.HasPrefix(sshConfig.KeyFile, "~/") {
				if homeDir, err := os.UserHomeDir(); err == nil {
					options.KeyFile = filepath.Join(homeDir, sshConfig.KeyFile[2:])
				}
			} else {
				options.KeyFile = sshConfig.KeyFile
			}
		}
	}

	return options
}

// SSHConfig holds SSH configuration for a host
type SSHConfig struct {
	User     string
	HostName string
	Port     string
	KeyFile  string
}

// tryResolveSSHConfig attempts to resolve SSH config for a host alias
func tryResolveSSHConfig(hostAlias string) *SSHConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")
	file, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var config *SSHConfig
	var inTargetHost bool
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split line into key and value
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		// Check for Host directive
		if key == "host" {
			if value == hostAlias {
				config = &SSHConfig{}
				inTargetHost = true
			} else {
				inTargetHost = false
			}
			continue
		}

		// If we're in the target host block, collect configuration
		if inTargetHost && config != nil {
			switch key {
			case "hostname":
				config.HostName = value
			case "user":
				config.User = value
			case "port":
				config.Port = value
			case "identityfile":
				config.KeyFile = value
			}
		}
	}

	return config
}
