package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewPutCommand creates the put command
func NewPutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put local_path... [user@]host[:port]:remote_path",
		Short: "Upload file(s) to remote server",
		Long: `Upload files or directories to a remote server via SFTP.

The connection string format is: [user@]host[:port]:path
Multiple local paths are supported; the shell will expand wildcards automatically.

Examples:
  sftpt put ./local/file.txt user@server.com:/remote/
  sftpt put *.log server.com:/var/log/app/
  sftpt put ./documents/ user@server.com:2222:/home/user/docs/`,
		Args: cobra.MinimumNArgs(2),
		RunE: runPutCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("recursive", "r", false, "Upload directories recursively")
	cmd.Flags().BoolP("preserve", "p", false, "Preserve file timestamps and permissions")
	cmd.Flags().Bool("progress", true, "Show progress bar during transfer")
	cmd.Flags().BoolP("create-dirs", "m", false, "Create remote directories if they don't exist")

	return cmd
}

func runPutCommand(cmd *cobra.Command, args []string) error {
	// Last arg is always the connection string: [user@]host[:port]:remote_path
	// All preceding args are local source paths (shell-expanded or single)
	remoteArg := args[len(args)-1]
	localPaths := args[:len(args)-1]

	connInfo, err := GetConnectionInfo([]string{remoteArg})
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}

	PrintVerbose(cmd, "Connecting to %s@%s:%s\n", connInfo.User, connInfo.Host, connInfo.Port)

	options := GetSFTPClientOptions(cmd)
	client, err := sftp.NewClientWithOptions(connInfo.Host, connInfo.Port, connInfo.User, options)
	if err != nil {
		return fmt.Errorf("creating SFTP client: %w", err)
	}
	defer client.Close()

	recursive, _ := cmd.Flags().GetBool("recursive")
	preserve, _ := cmd.Flags().GetBool("preserve")
	showProgress, _ := cmd.Flags().GetBool("progress")
	createDirs, _ := cmd.Flags().GetBool("create-dirs")

	transferOptions := sftp.TransferOptions{
		Recursive:    recursive,
		Preserve:     preserve,
		ShowProgress: showProgress,
		CreateDirs:   createDirs,
	}

	// Resolve local paths: expand any quoted globs, collect the set
	var sources []string
	for _, lp := range localPaths {
		if containsGlob(lp) {
			matches, err := filepath.Glob(lp)
			if err != nil {
				return fmt.Errorf("expanding glob %s: %w", lp, err)
			}
			if len(matches) == 0 {
				return fmt.Errorf("no files matching: %s", lp)
			}
			sources = append(sources, matches...)
		} else {
			sources = append(sources, lp)
		}
	}

	if len(sources) > 1 && !recursive {
		// Check that we're not uploading directories without --recursive
		for _, src := range sources {
			if fi, err := os.Stat(src); err == nil && fi.IsDir() {
				return fmt.Errorf("%s is a directory (use --recursive)", src)
			}
		}
	}

	if len(sources) > 1 {
		PrintVerbose(cmd, "Uploading %d files to %s\n", len(sources), connInfo.Path)
	}
	for _, src := range sources {
		PrintVerbose(cmd, "Uploading %s to %s\n", src, connInfo.Path)
		if err := client.Upload(src, connInfo.Path, transferOptions); err != nil {
			return fmt.Errorf("uploading %s: %w", src, err)
		}
	}

	if len(sources) > 1 {
		PrintOutput(cmd, "Uploaded %d files\n", len(sources))
	} else if !showProgress {
		filename := filepath.Base(sources[0])
		PrintOutput(cmd, "Uploaded: %s\n", filename)
	}

	return nil
}
