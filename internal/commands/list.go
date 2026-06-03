package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

type matchEntry struct {
	path string
	fi   os.FileInfo
}

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [user@]host[:port]:path",
		Short: "List remote directory contents",
		Long: `List the contents of a remote directory via SFTP.
Supports glob patterns (quote them to prevent local shell expansion).

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt list user@server.com:/home/user/
  sftpt list "server.com:/var/log/*.log"
  sftpt list user@server.com:2222:/custom/path/`,
		Args: cobra.ExactArgs(1),
		RunE: runListCommand,
	}

	// Command-specific flags
	cmd.Flags().BoolP("long", "l", false, "Long format (permissions, size, date)")
	cmd.Flags().BoolP("all", "a", false, "Include hidden files")
	cmd.Flags().Bool("human-readable", false, "Human-readable sizes (e.g. 1.2K, 4.5M)")
	cmd.Flags().String("sort", "", "Sort by: time, size, name (default name order)")

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
	humanReadable, _ := cmd.Flags().GetBool("human-readable")
	sortBy, _ := cmd.Flags().GetString("sort")

	switch sortBy {
	case "", "name", "time", "size":
	default:
		return fmt.Errorf("invalid --sort value: %q (use time, size, or name)", sortBy)
	}

	listOpts := sftp.ListOptions{
		LongFormat:    longFormat,
		ShowAll:       showAll,
		HumanReadable: humanReadable,
		Sort:          sortBy,
	}

	// Handle glob patterns
	if containsGlob(connInfo.Path) {
		matches, err := client.Glob(connInfo.Path)
		if err != nil {
			return fmt.Errorf("expanding glob %s: %w", connInfo.Path, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("no files matching: %s", connInfo.Path)
		}
		PrintVerbose(cmd, "Glob %s matched %d files\n", connInfo.Path, len(matches))

		var entries []matchEntry
		for _, match := range matches {
			fi, _ := client.StatFile(match)
			entries = append(entries, matchEntry{path: match, fi: fi})
		}

		sort.Slice(entries, func(i, j int) bool {
			return sortMatchEntries(entries[i], entries[j], sortBy)
		})

		for _, e := range entries {
			if longFormat && e.fi != nil {
				PrintOutput(cmd, "%s\n", sftp.FormatEntry(e.fi, e.path, listOpts))
			} else {
				PrintOutput(cmd, "%s\n", e.path)
			}
		}
		return nil
	}

	// List directory contents
	entries, err := client.List(connInfo.Path, listOpts)
	if err != nil {
		return fmt.Errorf("listing directory: %w", err)
	}

	for _, entry := range entries {
		PrintOutput(cmd, "%s\n", entry)
	}

	return nil
}

func sortMatchEntries(a, b matchEntry, sortBy string) bool {
	switch sortBy {
	case "time":
		if a.fi == nil && b.fi == nil {
			return a.path < b.path
		}
		if a.fi == nil {
			return false
		}
		if b.fi == nil {
			return true
		}
		return a.fi.ModTime().After(b.fi.ModTime())
	case "size":
		if a.fi == nil && b.fi == nil {
			return a.path < b.path
		}
		if a.fi == nil {
			return false
		}
		if b.fi == nil {
			return true
		}
		return a.fi.Size() > b.fi.Size()
	default:
		return a.path < b.path
	}
}
