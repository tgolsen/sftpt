package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [user@]host[:port]:path",
		Short: "List remote directory contents",
		Long: `List the contents of a remote directory via SFTP.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt list user@server.com:/home/user/
  sftpt list server.com:/var/log/
  sftpt list user@server.com:2222:/custom/path/`,
		Args: cobra.ExactArgs(1),
		RunE: runListCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("long", "l", false, "Long format (include permissions, size, date)")
	cmd.Flags().BoolP("all", "a", false, "Include hidden files")

	return cmd
}

func runListCommand(cmd *cobra.Command, args []string) error {
	// Store original arg for SSH config resolution
	originalArg := args[0]

	// Parse connection string
	connInfo, err := GetConnectionInfo(args)
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}

	PrintVerbose(cmd, "Connecting to %s@%s:%s\n", connInfo.User, connInfo.Host, connInfo.Port)

	// Get the host alias from original arg (before the colon)
	hostAlias := originalArg
	if colonIndex := strings.LastIndex(originalArg, ":"); colonIndex != -1 {
		hostAlias = originalArg[:colonIndex]
		// Remove user@ part if present
		if atIndex := strings.LastIndex(hostAlias, "@"); atIndex != -1 {
			hostAlias = hostAlias[atIndex+1:]
		}
	}

	// Create SFTP client with SSH config-aware options
	options := GetSFTPClientOptionsWithSSHConfig(cmd, connInfo, hostAlias)
	client, err := sftp.NewClientWithOptions(connInfo.Host, connInfo.Port, connInfo.User, options)
	if err != nil {
		return fmt.Errorf("creating SFTP client: %w", err)
	}
	defer client.Close()

	PrintVerbose(cmd, "Listing directory: %s\n", connInfo.Path)

	// Get command options
	longFormat, _ := cmd.Flags().GetBool("long")
	showAll, _ := cmd.Flags().GetBool("all")

	// List directory contents
	entries, err := client.List(connInfo.Path, longFormat, showAll)
	if err != nil {
		return fmt.Errorf("listing directory: %w", err)
	}

	// Print results (clean output for script consumption)
	for _, entry := range entries {
		PrintOutput(cmd, "%s\n", entry)
	}

	return nil
}
