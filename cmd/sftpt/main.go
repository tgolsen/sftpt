package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tgolsen/sftpt/internal/commands"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sftpt",
		Short: "Simple command-line SFTP utility for Mac",
		Long: `sftpt is a simple, scriptable command-line SFTP utility designed for easy
integration with shell scripts and automation workflows.

Provides clean output for scripts, handles SSH agent complexity gracefully,
and supports both interactive and automated authentication.`,
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, gitCommit, buildTime),
	}

	rootCmd.AddCommand(
		commands.NewGetCommand(),
		commands.NewPutCommand(),
		commands.NewListCommand(),
		commands.NewMkdirCommand(),
		commands.NewRmCommand(),
		commands.NewScriptCommand(),
		commands.NewHeadCommand(),
	)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output to stderr")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress informational output")
	rootCmd.PersistentFlags().StringP("key", "i", "", "SSH private key file path")
	rootCmd.PersistentFlags().Bool("password", false, "Prompt for password/passphrase")
	rootCmd.PersistentFlags().String("password-stdin", "", "Password or passphrase for authentication (for scripting)")
	rootCmd.PersistentFlags().Bool("keys-only", false, "Only use SSH key authentication")
	rootCmd.PersistentFlags().Bool("ssh-config", true, "Use ~/.ssh/config for host resolution")
	rootCmd.PersistentFlags().Bool("strict-host-checking", false, "Enable strict SSH host key checking")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
