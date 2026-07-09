package commands

import (
	"bufio"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/sftp"
)

func NewHeadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "head [user@]host[:port]:remote_path",
		Short: "Print the first N lines of a remote file",
		Long: `Print the first N lines of a remote file to stdout.

The connection string format is: [user@]host[:port]:path

Examples:
  sftpt head user@server.com:/var/log/app.log
  sftpt head -n 20 user@server.com:/etc/nginx/nginx.conf
  sftpt head server.com:2222:/home/user/data.csv`,
		Args: cobra.ExactArgs(1),
		RunE: runHeadCommand,
	}

	cmd.Flags().IntP("lines", "n", 10, "Number of lines to print")

	return cmd
}

func runHeadCommand(cmd *cobra.Command, args []string) error {
	connInfo, err := GetConnectionInfo(args)
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}

	n, _ := cmd.Flags().GetInt("lines")
	if n <= 0 {
		return fmt.Errorf("--lines must be a positive integer")
	}

	PrintVerbose(cmd, "Connecting to %s@%s:%s\n", connInfo.User, connInfo.Host, connInfo.Port)

	options := GetSFTPClientOptions(cmd)
	client, err := sftp.NewClientWithOptions(connInfo.Host, connInfo.Port, connInfo.User, options)
	if err != nil {
		return fmt.Errorf("creating SFTP client: %w", err)
	}
	defer client.Close()

	rc, err := client.OpenReader(connInfo.Path)
	if err != nil {
		return fmt.Errorf("opening remote file: %w", err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	for i := 0; i < n && scanner.Scan(); i++ {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading remote file: %w", err)
	}

	return nil
}
