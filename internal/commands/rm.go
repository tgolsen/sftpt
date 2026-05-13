package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewRmCommand creates the rm command
func NewRmCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [user@]host[:port]:path",
		Short: "Remove remote files or directories",
		Long: `Remove files or directories on the remote server via SFTP.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt rm user@server.com:/remote/file.txt
  sftpt rm server.com:/var/log/old.log
  sftpt rm -r user@server.com:2222:/home/user/olddir/`,
		Args: cobra.ExactArgs(1),
		RunE: runRmCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("recursive", "r", false, "Remove directories and their contents recursively")
	cmd.Flags().BoolP("force", "f", false, "Ignore nonexistent files and continue")

	return cmd
}

func runRmCommand(cmd *cobra.Command, args []string) error {
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
	recursive, _ := cmd.Flags().GetBool("recursive")
	force, _ := cmd.Flags().GetBool("force")

	PrintVerbose(cmd, "Removing: %s\n", connInfo.Path)

	// Remove file/directory
	err = client.Remove(connInfo.Path, recursive, force)
	if err != nil {
		if force {
			PrintVerbose(cmd, "Warning: %v\n", err)
		} else {
			return fmt.Errorf("removing file/directory: %w", err)
		}
	}

	// Clean output for scripts
	PrintOutput(cmd, "Removed: %s\n", connInfo.Path)

	return nil
}
