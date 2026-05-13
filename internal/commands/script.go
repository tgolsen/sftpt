package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewScriptCommand creates the script command
func NewScriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "script",
		Short: "Execute multiple SFTP commands from file or inline",
		Long: `Execute a series of SFTP commands for batch operations and automation.

Supports both script files and inline command strings for different use cases:
- Script files for complex automation workflows
- Inline commands for simple batch operations

Examples:
  # Execute commands from file
  sftpt script --file backup-commands.txt

  # Execute inline commands
  sftpt script --inline "list user@host:/logs/; get user@host:/logs/latest.log ./downloads/"

  # With authentication options
  sftpt script --file deploy.txt --password-stdin mypassword

Script file format (one command per line):
  list user@host:/remote/path/
  get user@host:/remote/file.txt ./local/
  put ./local/upload.txt user@host:/remote/
  mkdir user@host:/remote/newdir/
  rm user@host:/remote/oldfile.txt`,
		RunE: runScriptCommand,
	}

	// Command-specific flags
	cmd.Flags().StringP("file", "f", "", "Script file containing commands (one per line)")
	cmd.Flags().StringP("inline", "i", "", "Inline commands separated by semicolons")
	cmd.Flags().Bool("stop-on-error", true, "Stop execution on first error (default: true)")
	cmd.Flags().Bool("dry-run", false, "Show commands that would be executed without running them")

	return cmd
}

func runScriptCommand(cmd *cobra.Command, args []string) error {
	scriptFile, _ := cmd.Flags().GetString("file")
	inlineScript, _ := cmd.Flags().GetString("inline")
	stopOnError, _ := cmd.Flags().GetBool("stop-on-error")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Validate that exactly one input method is provided
	if scriptFile == "" && inlineScript == "" {
		return fmt.Errorf("must provide either --file or --inline")
	}
	if scriptFile != "" && inlineScript != "" {
		return fmt.Errorf("cannot use both --file and --inline at the same time")
	}

	var commands []string
	var err error

	// Parse commands from input
	if scriptFile != "" {
		commands, err = parseScriptFile(scriptFile)
	} else {
		commands = parseInlineScript(inlineScript)
	}

	if err != nil {
		return fmt.Errorf("parsing script: %w", err)
	}

	PrintVerbose(cmd, "Executing %d commands\n", len(commands))

	// Execute commands
	var errors []string
	successCount := 0

	for i, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" || strings.HasPrefix(command, "#") {
			continue // Skip empty lines and comments
		}

		PrintVerbose(cmd, "Command %d: %s\n", i+1, command)

		if dryRun {
			PrintOutput(cmd, "[DRY-RUN] Would execute: %s\n", command)
			continue
		}

		// Execute the command
		err := executeScriptCommand(cmd, command)
		if err != nil {
			errorMsg := fmt.Sprintf("Command %d failed: %v", i+1, err)
			errors = append(errors, errorMsg)

			if !cmd.Flags().Changed("quiet") {
				fmt.Fprintf(os.Stderr, "sftpt: %s\n", errorMsg)
			}

			if stopOnError {
				break
			}
		} else {
			successCount++
			PrintVerbose(cmd, "Command %d completed successfully\n", i+1)
		}
	}

	// Report results
	if dryRun {
		PrintOutput(cmd, "Dry run completed. %d commands would be executed.\n", len(commands))
	} else {
		PrintVerbose(cmd, "Script execution completed: %d/%d commands successful\n", successCount, len(commands))

		if len(errors) > 0 && !stopOnError {
			PrintOutput(cmd, "Script completed with %d errors:\n", len(errors))
			for _, errMsg := range errors {
				PrintOutput(cmd, "  - %s\n", errMsg)
			}
			return fmt.Errorf("script execution had %d errors", len(errors))
		}
	}

	return nil
}

// parseScriptFile reads commands from a file
func parseScriptFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening script file: %w", err)
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			commands = append(commands, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading script file: %w", err)
	}

	return commands, nil
}

// parseInlineScript parses semicolon-separated commands
func parseInlineScript(script string) []string {
	commands := strings.Split(script, ";")
	var result []string

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			result = append(result, cmd)
		}
	}

	return result
}

// executeScriptCommand executes a single sftpt command
func executeScriptCommand(parentCmd *cobra.Command, commandStr string) error {
	// Parse the command string into args
	args := parseCommandArgs(commandStr)
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}

	// Get the subcommand name
	subcommandName := args[0]
	subcommandArgs := args[1:]

	// Find and execute the appropriate subcommand
	switch subcommandName {
	case "list":
		listCmd := NewListCommand()
		listCmd.SetArgs(subcommandArgs)
		copyGlobalFlags(parentCmd, listCmd)
		return listCmd.Execute()

	case "get":
		getCmd := NewGetCommand()
		getCmd.SetArgs(subcommandArgs)
		copyGlobalFlags(parentCmd, getCmd)
		return getCmd.Execute()

	case "put":
		putCmd := NewPutCommand()
		putCmd.SetArgs(subcommandArgs)
		copyGlobalFlags(parentCmd, putCmd)
		return putCmd.Execute()

	case "mkdir":
		mkdirCmd := NewMkdirCommand()
		mkdirCmd.SetArgs(subcommandArgs)
		copyGlobalFlags(parentCmd, mkdirCmd)
		return mkdirCmd.Execute()

	case "rm":
		rmCmd := NewRmCommand()
		rmCmd.SetArgs(subcommandArgs)
		copyGlobalFlags(parentCmd, rmCmd)
		return rmCmd.Execute()

	default:
		return fmt.Errorf("unknown command: %s", subcommandName)
	}
}

// parseCommandArgs parses a command string into arguments
// Simple implementation - could be enhanced to handle quoted arguments
func parseCommandArgs(commandStr string) []string {
	return strings.Fields(commandStr)
}

// copyGlobalFlags copies global flags from parent to child command
func copyGlobalFlags(parent, child *cobra.Command) {
	// Copy key global flags that affect authentication and behavior
	if parent.Flags().Changed("key") {
		if keyVal, err := parent.Flags().GetString("key"); err == nil {
			child.Flags().Set("key", keyVal)
		}
	}

	if parent.Flags().Changed("password") {
		if passVal, err := parent.Flags().GetBool("password"); err == nil && passVal {
			child.Flags().Set("password", "true")
		}
	}

	if parent.Flags().Changed("password-stdin") {
		if passVal, err := parent.Flags().GetString("password-stdin"); err == nil {
			child.Flags().Set("password-stdin", passVal)
		}
	}

	if parent.Flags().Changed("verbose") {
		if verbVal, err := parent.Flags().GetBool("verbose"); err == nil && verbVal {
			child.Flags().Set("verbose", "true")
		}
	}

	if parent.Flags().Changed("quiet") {
		if quietVal, err := parent.Flags().GetBool("quiet"); err == nil && quietVal {
			child.Flags().Set("quiet", "true")
		}
	}

	if parent.Flags().Changed("keys-only") {
		if keysVal, err := parent.Flags().GetBool("keys-only"); err == nil && keysVal {
			child.Flags().Set("keys-only", "true")
		}
	}
}