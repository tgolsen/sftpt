package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewMkdirCommand creates the mkdir command
func NewMkdirCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mkdir [user@]host[:port]:path",
		Short: "Create remote directory",
		Long: `Create a directory on the remote server via SFTP.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt mkdir user@server.com:/remote/newdir/
  sftpt mkdir server.com:/var/backups/2024/
  sftpt mkdir user@server.com:2222:/home/user/projects/`,
		Args: cobra.ExactArgs(1),
		RunE: runMkdirCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("parents", "p", false, "Create parent directories as needed")
	cmd.Flags().StringP("mode", "m", "0755", "Directory permissions (octal)")

	return cmd
}

func runMkdirCommand(cmd *cobra.Command, args []string) error {
	// Parse connection string
	connInfo, err := GetConnectionInfo(args)
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}

	PrintVerbose(cmd, "Connecting to %s@%s:%s\n", connInfo.User, connInfo.Host, connInfo.Port)

	// Create SFTP client with options from flags
	options := GetSFTPClientOptions(cmd)
	client, err := sftp.NewClientWithOptions(connInfo.Host, connInfo.Port, connInfo.User, options)
	if err != nil {
		return fmt.Errorf("creating SFTP client: %w", err)
	}
	defer client.Close()

	// Get command options
	parents, _ := cmd.Flags().GetBool("parents")
	mode, _ := cmd.Flags().GetString("mode")

	PrintVerbose(cmd, "Creating directory: %s\n", connInfo.Path)

	// Create directory
	err = client.Mkdir(connInfo.Path, mode, parents)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Clean output for scripts
	PrintOutput(cmd, "Created directory: %s\n", connInfo.Path)

	return nil
}
