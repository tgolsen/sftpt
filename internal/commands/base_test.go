package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      *ConnectionInfo
		expectedError string
	}{
		{
			name:  "full connection string",
			input: "user@host.com:2222:/remote/path",
			expected: &ConnectionInfo{
				User: "user",
				Host: "host.com",
				Port: "2222",
				Path: "/remote/path",
			},
		},
		{
			name:  "no user, default port",
			input: "host.com:/remote/path",
			expected: &ConnectionInfo{
				User: "",
				Host: "host.com",
				Port: "22",
				Path: "/remote/path",
			},
		},
		{
			name:  "user with default port",
			input: "user@host.com:/remote/path",
			expected: &ConnectionInfo{
				User: "user",
				Host: "host.com",
				Port: "22",
				Path: "/remote/path",
			},
		},
		{
			name:  "custom port, no user",
			input: "host.com:2222:/remote/path",
			expected: &ConnectionInfo{
				User: "",
				Host: "host.com",
				Port: "2222",
				Path: "/remote/path",
			},
		},
		{
			name:  "empty path",
			input: "user@host.com:2222:",
			expected: &ConnectionInfo{
				User: "user",
				Host: "host.com",
				Port: "2222",
				Path: "",
			},
		},
		{
			name:          "empty string",
			input:         "",
			expectedError: "empty connection string",
		},
		{
			name:          "no colon separator",
			input:         "user@host.com",
			expectedError: "invalid connection string format: missing path",
		},
		{
			name:          "no host",
			input:         ":2222:/path",
			expectedError: "host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseConnectionString(tt.input)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseConnectionStringSSHConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0o700))
	config := `Host sftp.example.com
    User deploy
    Port 2200

Host myalias
    HostName real.example.com
    User alias-user
`
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte(config), 0o600))

	t.Run("resolves user for dotted hostname", func(t *testing.T) {
		result, err := ParseConnectionString("sftp.example.com:/remote/path")
		require.NoError(t, err)
		assert.Equal(t, "deploy", result.User)
		assert.Equal(t, "sftp.example.com", result.Host)
		assert.Equal(t, "2200", result.Port)
	})

	t.Run("explicit user overrides ssh config", func(t *testing.T) {
		result, err := ParseConnectionString("bob@sftp.example.com:/remote/path")
		require.NoError(t, err)
		assert.Equal(t, "bob", result.User)
	})

	t.Run("explicit port overrides ssh config", func(t *testing.T) {
		result, err := ParseConnectionString("sftp.example.com:2222:/remote/path")
		require.NoError(t, err)
		assert.Equal(t, "deploy", result.User)
		assert.Equal(t, "2222", result.Port)
	})

	t.Run("alias still resolves hostname and user", func(t *testing.T) {
		result, err := ParseConnectionString("myalias:/remote/path")
		require.NoError(t, err)
		assert.Equal(t, "alias-user", result.User)
		assert.Equal(t, "real.example.com", result.Host)
	})
}

func TestGetConnectionInfo(t *testing.T) {
	t.Run("valid args", func(t *testing.T) {
		args := []string{"user@host.com:2222:/path"}
		result, err := GetConnectionInfo(args)

		require.NoError(t, err)
		assert.Equal(t, "user", result.User)
		assert.Equal(t, "host.com", result.Host)
		assert.Equal(t, "2222", result.Port)
		assert.Equal(t, "/path", result.Path)
	})

	t.Run("no args", func(t *testing.T) {
		args := []string{}
		result, err := GetConnectionInfo(args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection string required")
		assert.Nil(t, result)
	})
}
