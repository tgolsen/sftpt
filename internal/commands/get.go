package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewGetCommand creates the get command
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [user@]host[:port]:remote_path local_path",
		Short: "Download file(s) from remote server",
		Long: `Download files or directories from a remote server via SFTP.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt get user@server.com:/remote/file.txt ./local/
  sftpt get server.com:/var/log/app.log ./logs/
  sftpt get user@server.com:2222:/home/user/docs/ ./documents/`,
		Args: cobra.ExactArgs(2),
		RunE: runGetCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("recursive", "r", false, "Download directories recursively")
	cmd.Flags().BoolP("preserve", "p", false, "Preserve file timestamps and permissions")
	cmd.Flags().Bool("progress", false, "Show progress for large files")

	return cmd
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	// Parse connection string
	connInfo, err := GetConnectionInfo(args[:1])
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}

	localPath := args[1]

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
	preserve, _ := cmd.Flags().GetBool("preserve")
	showProgress, _ := cmd.Flags().GetBool("progress")

	PrintVerbose(cmd, "Downloading %s to %s\n", connInfo.Path, localPath)

	// Download file(s)
	transferOptions := sftp.TransferOptions{
		Recursive:    recursive,
		Preserve:     preserve,
		ShowProgress: showProgress,
	}

	err = client.Download(connInfo.Path, localPath, transferOptions)
	if err != nil {
		return fmt.Errorf("downloading files: %w", err)
	}

	// Clean output for scripts
	if !showProgress {
		filename := filepath.Base(connInfo.Path)
		PrintOutput(cmd, "Downloaded: %s\n", filename)
	}

	return nil
}
