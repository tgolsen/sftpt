package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryDefaultSSHKeys(t *testing.T) {
	// This test checks that the function doesn't panic and returns appropriately
	methods := tryDefaultSSHKeys()

	// Function should always return a slice (may be empty or contain methods)
	assert.NotNil(t, methods)
	// Should be zero or more methods depending on available SSH keys
	assert.True(t, len(methods) >= 0, "Should return zero or more auth methods")
}

func TestTrySSHAgent(t *testing.T) {
	t.Run("no SSH_AUTH_SOCK", func(t *testing.T) {
		// Temporarily clear SSH_AUTH_SOCK
		originalSock := os.Getenv("SSH_AUTH_SOCK")
		os.Unsetenv("SSH_AUTH_SOCK")
		defer func() {
			if originalSock != "" {
				os.Setenv("SSH_AUTH_SOCK", originalSock)
			}
		}()

		methods := trySSHAgent()
		assert.Nil(t, methods)
	})

	t.Run("invalid SSH_AUTH_SOCK", func(t *testing.T) {
		// Set invalid SSH_AUTH_SOCK
		originalSock := os.Getenv("SSH_AUTH_SOCK")
		os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
		defer func() {
			if originalSock != "" {
				os.Setenv("SSH_AUTH_SOCK", originalSock)
			} else {
				os.Unsetenv("SSH_AUTH_SOCK")
			}
		}()

		methods := trySSHAgent()
		assert.Nil(t, methods)
	})
}

func TestTryLoadSSHKey(t *testing.T) {
	t.Run("nonexistent key", func(t *testing.T) {
		method := tryLoadSSHKey("/nonexistent/key")
		assert.Nil(t, method)
	})

	t.Run("invalid key file", func(t *testing.T) {
		// Create a temporary invalid key file
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "invalid_key")

		err := os.WriteFile(keyFile, []byte("not a valid SSH key"), 0600)
		require.NoError(t, err)

		method := tryLoadSSHKey(keyFile)
		assert.Nil(t, method)
	})
}

func TestGetAuthConfig(t *testing.T) {
	t.Run("basic config", func(t *testing.T) {
		config, err := GetAuthConfig("testuser", "testhost", "22")

		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Methods)

		// Should have at least password authentication as fallback
		assert.True(t, len(config.Methods) >= 1)
	})
}

func TestGetAuthConfigWithOptions(t *testing.T) {
	t.Run("password only", func(t *testing.T) {
		config, err := GetAuthConfigWithOptions("", true)

		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Methods)
	})

	t.Run("no methods available", func(t *testing.T) {
		config, err := GetAuthConfigWithOptions("", false)

		// This might succeed if SSH agent or default keys are available,
		// or fail if no methods are available
		if err != nil {
			assert.Contains(t, err.Error(), "no authentication methods available")
		} else {
			assert.NotNil(t, config)
		}
	})

	t.Run("nonexistent key file", func(t *testing.T) {
		config, err := GetAuthConfigWithOptions("/nonexistent/key", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading SSH key")
		assert.Nil(t, config)
	})
}
