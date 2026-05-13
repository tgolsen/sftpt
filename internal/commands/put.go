package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewPutCommand creates the put command
func NewPutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put local_path [user@]host[:port]:remote_path",
		Short: "Upload file(s) to remote server",
		Long: `Upload files or directories to a remote server via SFTP.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt put ./local/file.txt user@server.com:/remote/
  sftpt put ./logs/ server.com:/var/log/app/
  sftpt put ./documents/ user@server.com:2222:/home/user/docs/`,
		Args: cobra.ExactArgs(2),
		RunE: runPutCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("recursive", "r", false, "Upload directories recursively")
	cmd.Flags().BoolP("preserve", "p", false, "Preserve file timestamps and permissions")
	cmd.Flags().Bool("progress", false, "Show progress for large files")
	cmd.Flags().BoolP("create-dirs", "m", false, "Create remote directories if they don't exist")

	return cmd
}

func runPutCommand(cmd *cobra.Command, args []string) error {
	localPath := args[0]

	// Parse connection string from second argument
	connInfo, err := GetConnectionInfo(args[1:])
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
	preserve, _ := cmd.Flags().GetBool("preserve")
	showProgress, _ := cmd.Flags().GetBool("progress")
	createDirs, _ := cmd.Flags().GetBool("create-dirs")

	PrintVerbose(cmd, "Uploading %s to %s\n", localPath, connInfo.Path)

	// Upload file(s)
	transferOptions := sftp.TransferOptions{
		Recursive:    recursive,
		Preserve:     preserve,
		ShowProgress: showProgress,
		CreateDirs:   createDirs,
	}

	err = client.Upload(localPath, connInfo.Path, transferOptions)
	if err != nil {
		return fmt.Errorf("uploading files: %w", err)
	}

	// Clean output for scripts
	if !showProgress {
		filename := filepath.Base(localPath)
		PrintOutput(cmd, "Uploaded: %s\n", filename)
	}

	return nil
}
