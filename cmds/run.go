package cmds

import (
	"bufio"
	"embed"
	"fmt"
	"github.com/alexj212/consolekit"
	"github.com/kballard/go-shellquote"

	"github.com/spf13/cobra"
	"io"
	"os"
	"time"

	"strings"
)

func AddRun(cli *consolekit.CLI, scripts embed.FS) {

	var viewScriptCmdFunc = func(cmd *cobra.Command, args []string) {

		if args[0] == "@" {
			f, err := cli.Scripts.ReadDir(".")
			if err != nil {
				cmd.Printf("Error reading scripts: %v\n", err)
				return
			}

			cmd.Printf("Scripts Available:\n")
			for _, script := range f {
				if strings.HasSuffix(script.Name(), ".go") {
					continue
				}
				cmd.Printf("@%s\n", script.Name())
			}
			return
		}

		cmds, err := LoadScript(scripts, cmd, args[0])
		if err != nil {
			cmd.Printf(cli.ErrorString("error loading file: %s, %s\n", args[0], err))
			return
		}

		cmd.Printf("Script file: %s - %d commands\n", args[0], len(cmds))
		for _, cmdLine := range cmds {
			if cmdLine == "" {
				continue
			}
			cmdLine = strings.TrimSpace(cmdLine)
			cmd.Printf("%s\n", cmdLine)
		}
	}
	var viewScriptCmd = &cobra.Command{
		Use:     "vs {file | @}",
		Aliases: []string{"view-script"},
		Short:   "view script file, pass @ to list all scripts",
		Args:    cobra.ExactArgs(1),

		Run: viewScriptCmdFunc,
	}

	var runScriptCmdFunc = func(cmd *cobra.Command, args []string) {
		for i, arg := range args[1:] {
			cli.Defaults.Set(fmt.Sprintf("@arg%d", i), arg)
		}

		if args[0] == "@" {
			f, err := scripts.ReadDir(".")
			if err != nil {
				cmd.Printf("Error reading scripts: %v\n", err)
				return
			}

			cmd.Printf("Scripts Available:\n")
			for _, script := range f {
				if strings.HasSuffix(script.Name(), ".go") {
					continue
				}
				cmd.Printf("@%s\n", script.Name())
			}
			return
		}

		cmds, err := LoadScript(scripts, cmd, args[0])
		if err != nil {
			cmd.Printf(cli.ErrorString("error loading file %s, %s\n", args[0], err))
			return
		}

		spawn, err := cmd.Flags().GetBool("spawn")
		if err != nil {
			cmd.Printf(cli.ErrorString("unable to get flag spawn, %v\n", err))
			return
		}

		doExec := func() {
			startTime := time.Now()
			cmd.Printf("Executing file: %s - %d commands\n", args[0], len(cmds))

			for _, cmdLine := range cmds {
				if cmdLine == "" {
					continue
				}

				cmdLine = cli.ReplaceDefaults(nil, cmdLine)

				cmd.Printf("doExec: %s\n", cmdLine)
				res, err := cli.Repl.ExecuteCommand(cli.RootCmd, cmdLine)
				cmd.Printf("res %s\n", res)
				if err != nil {
					cmd.Printf(cli.ErrorString("error executing command: %s, %s\n", cmdLine, err))
					break
				}
			}
			elapsed := time.Since(startTime)
			timeSince := HumanizeDuration(elapsed, false)
			cmd.Printf("script `%s` - Execution time: %s\n", args[0], timeSince)
		}

		if spawn {
			go doExec()
			return
		}
		doExec()
	}

	var runScriptCmd = &cobra.Command{
		Use:   "run [--spawn] {file | @file | @ } [args]...",
		Short: "exec script file, use `@name` files for internal scripts. pass args that can be referenced in script as @arg0, @arg1, ...",
		Args:  cobra.MinimumNArgs(1),
		PostRun: func(cmd *cobra.Command, args []string) {
			consolekit.ResetHelpFlagRecursively(cmd)
			consolekit.ResetAllFlags(cmd)
		},

		Run: runScriptCmdFunc,
	}

	var spawnScriptCmdFunc = func(cmd *cobra.Command, args []string) {

		go func() {
			cmdLine := strings.Join(args, " ")
			cmdLineArgs, err := shellquote.Split(cmdLine)
			if err != nil {
				cmd.Printf(cli.ErrorString("error splitting cmd `%s`, %s\n", cmdLine, err))
				return
			}
			cmd.Printf("spawn cmd: %s | %s\n", cli.RootCmd.Use, cmdLine)
			cli.RootCmd.SetArgs(cmdLineArgs)
			if err := cli.RootCmd.Execute(); err != nil {
				cmd.Printf(cli.ErrorString("error %s executing command: %s, %s\n", cli.RootCmd.Name(), cmdLine, err))
				return
			}
		}()

	}

	var spawnScriptCmd = &cobra.Command{
		Use:   "spawn {cmd}",
		Short: "exec command",
		Args:  cobra.ExactArgs(1),

		Run: spawnScriptCmdFunc,
	}

	cli.AddCommand(viewScriptCmd)
	cli.AddCommand(runScriptCmd)
	cli.AddCommand(spawnScriptCmd)

	runScriptCmd.Flags().BoolVarP(new(bool), "spawn", "", false, "run script in background")
}

//func ExecRaw(cmd *cobra.Command, cmdLine string) error {
//	cmd.Printf("Executing ExecRaw: %s\n", cmdLine)
//	if strings.HasPrefix(cmdLine, "##") {
//		msg := strings.TrimPrefix(cmdLine, "##")
//		cmd.Printf("%s\n", msg)
//		return nil
//	}
//	if cmdLine == "" || strings.HasPrefix(cmdLine, "#") {
//		return nil
//	}
//
//	cmd.Printf("Executing cmd: %s\n", cmdLine)
//	cmds := strings.Split(cmdLine, "\n")
//	for _, execCmd := range cmds {
//
//		cmdLineArgs, err := shellquote.Split(execCmd)
//		if err != nil {
//			cmd.Printf(cli.ErrorString("error splitting, `%s`, %s\n", execCmd, err))
//			return err
//		}
//
//		cmd.Printf("\n----------------------------------------\n")
//		cmd.Printf("cmd: %s\n\n", execCmd)
//
//		cli.RootCmd.SetArgs(cmdLineArgs)
//		if err := cli.RootCmd.Execute(); err != nil {
//			cmd.Printf(cli.ErrorString("execute `%s`, command: %s, error %s\n", cli.RootCmd.Name(), execCmd, err))
//			return err
//		}
//	}
//	return nil
//}

func HumanizeDuration(duration time.Duration, showMs bool) string {
	ms := duration.Milliseconds() % 1000
	totalSeconds := int(duration.Seconds()) // Convert to int
	minutes := totalSeconds / 60
	hours := minutes / 60
	minutes = minutes % 60
	seconds := totalSeconds % 60
	if hours < 0 {
		hours = 0
	}
	if minutes < 0 {
		minutes = 0
	}
	if seconds < 0 {
		seconds = 0
	}

	if showMs {
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, ms)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func ReadLines(rdr io.Reader) ([]string, error) {

	// Prepare to read lines and accumulate multi-line commands
	scanner := bufio.NewScanner(rdr)
	var commandBuilder strings.Builder
	results := make([]string, 0)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line ends with a backslash, indicating a multi-line command
		if strings.HasSuffix(line, "\\") {
			// Replace the trailing backslash with a newline character
			commandBuilder.WriteString(strings.TrimSuffix(line, "\\"))
		} else {
			commandBuilder.WriteString(line + "\n")
			command := commandBuilder.String()
			commandBuilder.Reset() // Clear the builder for the next command
			// Execute the command
			results = append(results, command)
		}
	}
	return results, scanner.Err()
}

func LoadScript(scripts embed.FS, cmd *cobra.Command, filename string) ([]string, error) {

	if len(filename) == 0 {
		return nil, fmt.Errorf("no filename provided")
	}

	if strings.HasPrefix(filename, "@") {

		// Remove the @ prefix
		// Read the file content
		content, err := scripts.ReadFile(filename[1:])
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Convert the content into a string
		text := string(content)

		// Split the text into lines
		return ReadLines(strings.NewReader(text))
	}

	// Read the entire file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("LoadScript failed to read file: %w", err)
	}
	cmd.Printf("LoadScript content: %d\n", len(content))
	return ReadLines(strings.NewReader(string(content)))
}
