package auth

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Methods []ssh.AuthMethod
}

// AuthOptions holds authentication preferences
type AuthOptions struct {
	KeyFile              string
	UsePassword          bool
	PasswordFromStdin    string
	KeysOnly             bool
	UseSSHConfig         bool
	StrictHostChecking   bool
}

// GetAuthConfig creates authentication configuration with smart method selection
func GetAuthConfig(user, host, port string) (*AuthConfig, error) {
	// This is the legacy function - use GetAuthConfigWithFlags for new calls
	return GetAuthConfigWithFlags(user, host, port, AuthOptions{
		UseSSHConfig:       true,
		StrictHostChecking: true,
	})
}

// GetAuthConfigWithFlags creates authentication config with explicit options
func GetAuthConfigWithFlags(user, host, port string, options AuthOptions) (*AuthConfig, error) {
	var methods []ssh.AuthMethod

	// When password is explicitly requested (either method), use ONLY password auth
	// This prevents "too many authentication failures" from trying multiple SSH keys first
	if options.PasswordFromStdin != "" {
		methods = append(methods, ssh.Password(options.PasswordFromStdin))
		return &AuthConfig{Methods: methods}, nil
	}

	if options.UsePassword {
		methods = append(methods, ssh.PasswordCallback(func() (string, error) {
			return promptPassword(user)
		}))
		return &AuthConfig{Methods: methods}, nil
	}

	// For key-based authentication, build method list based on preferences

	// Priority 1: Specific SSH key file
	if options.KeyFile != "" {
		keyMethod, err := loadSSHKey(options.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading SSH key from %s: %w", options.KeyFile, err)
		}
		methods = append(methods, keyMethod)
	}

	// Priority 2: SSH config file keys (if enabled and no explicit key specified)
	if options.UseSSHConfig && options.KeyFile == "" {
		if sshConfigMethods := trySSHConfigKeys(user, host, port); len(sshConfigMethods) > 0 {
			methods = append(methods, sshConfigMethods...)
		}
	}

	// Priority 3: SSH agent (if not keys-only mode)
	if !options.KeysOnly {
		if agentMethods := trySSHAgent(); len(agentMethods) > 0 {
			methods = append(methods, agentMethods...)
		}
	}

	// Priority 4: Default SSH keys (if not keys-only mode and no explicit key)
	if !options.KeysOnly && options.KeyFile == "" {
		if keyMethods := tryDefaultSSHKeys(); len(keyMethods) > 0 {
			methods = append(methods, keyMethods...)
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods available. Try --password, --key, or ensure SSH keys are available")
	}

	return &AuthConfig{
		Methods: methods,
	}, nil
}

// GetAuthConfigWithOptions creates authentication config with explicit options
func GetAuthConfigWithOptions(keyFile string, usePassword bool) (*AuthConfig, error) {
	var methods []ssh.AuthMethod

	// Use specific SSH key if provided
	if keyFile != "" {
		keyMethod, err := loadSSHKey(keyFile)
		if err != nil {
			return nil, fmt.Errorf("loading SSH key from %s: %w", keyFile, err)
		}
		methods = append(methods, keyMethod)
	} else {
		// Try SSH agent first (if available)
		if agentMethods := trySSHAgent(); len(agentMethods) > 0 {
			methods = append(methods, agentMethods...)
		}

		// Try default SSH keys
		if keyMethods := tryDefaultSSHKeys(); len(keyMethods) > 0 {
			methods = append(methods, keyMethods...)
		}
	}

	// Add password authentication if requested
	if usePassword {
		methods = append(methods, ssh.PasswordCallback(func() (string, error) {
			return promptPassword("user")
		}))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	return &AuthConfig{
		Methods: methods,
	}, nil
}

// trySSHAgent attempts to connect to SSH agent
func trySSHAgent() []ssh.AuthMethod {
	// Check if SSH_AUTH_SOCK is set
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return nil
	}

	// Try to connect to SSH agent
	agentConn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil
	}

	return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers)}
}

// tryDefaultSSHKeys attempts to load default SSH keys
func tryDefaultSSHKeys() []ssh.AuthMethod {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Common SSH key paths
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		filepath.Join(homeDir, ".ssh", "id_dsa"),
	}

	var methods []ssh.AuthMethod
	for _, keyPath := range keyPaths {
		if method := tryLoadSSHKey(keyPath); method != nil {
			methods = append(methods, method)
		}
	}

	return methods
}

// tryLoadSSHKey tries to load an SSH key, returns nil if it fails
func tryLoadSSHKey(keyPath string) ssh.AuthMethod {
	method, err := loadSSHKey(keyPath)
	if err != nil {
		return nil
	}
	return method
}

// loadSSHKey loads an SSH private key from file
func loadSSHKey(keyPath string) (ssh.AuthMethod, error) {
	// Read private key file
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	// Try to parse key without passphrase first
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		// Key might be encrypted, try with passphrase
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			passphrase, err := promptPassphrase(keyPath)
			if err != nil {
				return nil, fmt.Errorf("getting passphrase: %w", err)
			}

			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("parsing encrypted key: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parsing key: %w", err)
		}
	}

	return ssh.PublicKeys(signer), nil
}

// promptPassphrase prompts user for SSH key passphrase
func promptPassphrase(keyPath string) (string, error) {
	fmt.Printf("Enter passphrase for SSH key %s: ", keyPath)

	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input

	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}

	return string(passphrase), nil
}

// promptPassword prompts user for password
func promptPassword(user string) (string, error) {
	fmt.Printf("Password for %s: ", user)

	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input

	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}

	return string(password), nil
}

// trySSHConfigKeys attempts to load SSH keys specified in ~/.ssh/config
func trySSHConfigKeys(user, host, port string) []ssh.AuthMethod {
	// This is a simplified implementation - a full implementation would
	// parse ~/.ssh/config properly, but for now we'll return empty slice
	// TODO: Implement proper SSH config parsing
	return []ssh.AuthMethod{}
}
