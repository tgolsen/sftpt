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
Supports glob patterns (quote them to prevent local shell expansion).

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt get user@server.com:/remote/file.txt ./local/
  sftpt get "server.com:/var/log/*.log" ./logs/
  sftpt get user@server.com:2222:/home/user/docs/ ./documents/`,
		Args: cobra.ExactArgs(2),
		RunE: runGetCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("recursive", "r", false, "Download directories recursively")
	cmd.Flags().BoolP("preserve", "p", false, "Preserve file timestamps and permissions")
	cmd.Flags().Bool("progress", true, "Show progress bar during transfer")

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

	transferOptions := sftp.TransferOptions{
		Recursive:    recursive,
		Preserve:     preserve,
		ShowProgress: showProgress,
	}

	// Handle remote glob patterns
	if containsGlob(connInfo.Path) {
		matches, err := client.Glob(connInfo.Path)
		if err != nil {
			return fmt.Errorf("expanding glob %s: %w", connInfo.Path, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("no files matching: %s", connInfo.Path)
		}
		PrintVerbose(cmd, "Glob %s matched %d files\n", connInfo.Path, len(matches))
		for _, match := range matches {
			PrintVerbose(cmd, "Downloading %s to %s\n", match, localPath)
			if err := client.Download(match, localPath, transferOptions); err != nil {
				return fmt.Errorf("downloading %s: %w", match, err)
			}
		}
		PrintOutput(cmd, "Downloaded %d files\n", len(matches))
		return nil
	}

	PrintVerbose(cmd, "Downloading %s to %s\n", connInfo.Path, localPath)

	err = client.Download(connInfo.Path, localPath, transferOptions)
	if err != nil {
		return fmt.Errorf("downloading files: %w", err)
	}

	if !showProgress {
		filename := filepath.Base(connInfo.Path)
		PrintOutput(cmd, "Downloaded: %s\n", filename)
	}

	return nil
}
