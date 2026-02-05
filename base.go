package consolekit

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const ClsSeq = "\033[H\033[2J"

// AddBaseCmds registers essential core commands: cls, exit, print, date
// These are the minimal commands needed for basic CLI operation.
func AddBaseCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// cls command - clear screen
		var clsCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf(ClsSeq)
		}

		var clsCmd = &cobra.Command{
			Use:     "cls",
			Aliases: []string{"clear"},
			Short:   "Clear the screen",
			Long:    `Clear the screen`,
			Run:     clsCmdFunc,
		}
		rootCmd.AddCommand(clsCmd)

		// exit command - exit the program
		var exitCmdFunc = func(cmd *cobra.Command, args []string) {
			code := 0

			if len(args) > 0 {
				code, _ = strconv.Atoi(args[0])
			}
			os.Exit(code)

		} //exitCmdFunc
		var exitCmd = &cobra.Command{
			Use:     "exit {code}",
			Short:   "Exit the program",
			Aliases: []string{"x", "quit", "q"},
			Long:    `exit the program`,
			Args:    cobra.MaximumNArgs(1),
			Run:     exitCmdFunc,
		}
		rootCmd.AddCommand(exitCmd)

		// print command - display message
		var printCmdFunc = func(cmd *cobra.Command, args []string) {
			line := strings.Join(args, " ")

			// Use ExpandVariables instead of ExpandCommand to avoid alias expansion in arguments
			line = exec.ExpandVariables(cmd, nil, line)

			cmd.Printf("%s\n", line)
		}

		var printCmd = &cobra.Command{
			Use:                "print {message}",
			Short:              "print message",
			Aliases:            []string{"p", "echo"},
			Run:                printCmdFunc,
			DisableFlagParsing: true,
			DisableSuggestions: true,
		}
		rootCmd.AddCommand(printCmd)

		// date command - display current date/time
		var dateCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s\n", time.Now().Format(time.RFC3339))
		}

		var dateCmd = &cobra.Command{
			Use:   "date",
			Short: "print date",
			Run:   dateCmdFunc,
		}
		rootCmd.AddCommand(dateCmd)
	}
}

// AddNetworkCommands registers network-related commands: http
func AddNetworkCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var httpCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("http call to %s\n", args[0])
			data, respCode, err := FetchURLContent(cmd, args[0])
			if err != nil {
				cmd.Printf("error fetching url: %v resp code: %d err: %v\n", args[0], respCode, err)
				return
			}
			cmd.Printf("\n%s\n", data)
		}

		var httpCmd = &cobra.Command{
			Use:   "http {url}",
			Short: "http fetch url content",
			Run:   httpCmdFunc,
			Args:  cobra.ExactArgs(1),
		}
		httpCmd.Flags().BoolP("show-headers", "", false, "show headers")
		httpCmd.Flags().BoolP("show-status_code", "", false, "show status_code")
		httpCmd.Flags().BoolP("show-details", "", false, "show details")

		rootCmd.AddCommand(httpCmd)
	}
}

// AddTimeCommands registers time-related commands: sleep, wait, waitfor
func AddTimeCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// sleep command
		var sleepCmd = &cobra.Command{
			Use:     "sleep [--quiet] {secs}",
			Short:   "sleep {n} seconds with optional progress updates",
			Long:    "Sleep for specified seconds. Shows progress for sleeps >= 5 seconds unless --quiet is specified.",
			Example: "sleep 5\nsleep 60\nsleep --quiet 30",
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				delay, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.Printf("Invalid delay %s\n", args[0])
					return
				}

				quiet, _ := cmd.Flags().GetBool("quiet")

				// For short sleeps (< 5 seconds) or quiet mode, just sleep without updates
				if delay < 5 || quiet {
					if !quiet {
						cmd.Printf("Sleeping for %d seconds\n", delay)
					}
					time.Sleep(time.Duration(delay) * time.Second)
					return
				}

				// For longer sleeps, show progress updates
				startTime := time.Now()
				endTime := startTime.Add(time.Duration(delay) * time.Second)
				cmd.Printf("⏱  Sleeping for %d seconds (until %s)\n", delay, endTime.Format("15:04:05"))

				// Determine update interval based on total duration
				updateInterval := time.Second
				if delay >= 300 {
					updateInterval = 30 * time.Second // Every 30 seconds for sleeps >= 5 minutes
				} else if delay >= 60 {
					updateInterval = 10 * time.Second // Every 10 seconds for sleeps >= 1 minute
				} else if delay >= 30 {
					updateInterval = 5 * time.Second // Every 5 seconds for sleeps >= 30 seconds
				} else {
					updateInterval = 2 * time.Second // Every 2 seconds for sleeps >= 5 seconds
				}

				ticker := time.NewTicker(updateInterval)
				defer ticker.Stop()

				done := time.After(time.Duration(delay) * time.Second)

				for {
					select {
					case <-done:
						elapsed := time.Since(startTime)
						cmd.Printf("✓ Sleep completed - waited %s\n", HumanizeDuration(elapsed, false))
						return
					case <-ticker.C:
						elapsed := time.Since(startTime)
						remaining := time.Until(endTime)
						percentage := int((float64(elapsed) / float64(delay*int(time.Second))) * 100)
						cmd.Printf("  ⏳ Progress: %s elapsed, %s remaining (%d%%)\n",
							HumanizeDuration(elapsed, false),
							HumanizeDuration(remaining, false),
							percentage)
					}
				}
			},
		}
		sleepCmd.Flags().BoolP("quiet", "q", false, "suppress progress updates")

		// wait command - pauses execution until a specified time
		var waitCmd = &cobra.Command{
			Use:   "wait --time HH:MM",
			Short: "Pauses execution until the specified time (24-hour format)",
			Long: `Pauses the execution of the command until the specified time in HH:MM format (24-hour clock).
If the specified time is earlier than the current time, the command will wait until that time on the next day.`,
			Example: `  wait --time 14:30  # Waits until 2:30 PM today or the next day if past
  wait --time 08:00  # Waits until 8:00 AM`,
			RunE: func(cmd *cobra.Command, args []string) error {
				targetTime, err := cmd.Flags().GetString("time")
				if err != nil {
					return err
				}

				t, err := time.Parse("15:04", targetTime)
				if err != nil {
					return fmt.Errorf("invalid time format, use HH:MM (24-hour format): %v", err)
				}

				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
				if next.Before(now) {
					next = next.Add(24 * time.Hour)
				}

				cmd.Printf("Waiting until %v\n", next)
				time.Sleep(time.Until(next))

				cmd.Printf("Time reached!\n")
				return nil
			},
		}
		waitCmd.Flags().StringP("time", "t", "", "Time to wait until in HH:MM format (24-hour)")
		_ = waitCmd.MarkFlagRequired("time")

		// waitfor command - waits until a condition is met
		var waitForCmd = &cobra.Command{
			Use:   "waitfor --target TARGET",
			Short: "Waits until a specified condition is met",
			Long: `This command waits until a specific condition is met.
In this example, it waits until a counter reaches or exceeds a target value.`,
			Example: ` waitfor --target 10 --interval 2  # Waits until counter reaches 10, checking every 2 seconds`,
			RunE: func(cmd *cobra.Command, args []string) error {
				target, err := cmd.Flags().GetInt("target")
				if err != nil {
					return err
				}

				interval, err := cmd.Flags().GetInt("interval")
				if err != nil {
					return err
				}

				counter := 0
				cmd.Printf("Waiting until counter reaches %d...\n", target)

				for {
					if counter >= target {
						cmd.Printf("Condition met! Counter has reached %d.\n", counter)
						break
					}

					cmd.Printf("Counter at %d, waiting %d seconds before next check...\n", counter, interval)
					time.Sleep(time.Duration(interval) * time.Second)
					counter++
				}

				return nil
			},
		}
		waitForCmd.Flags().IntP("target", "t", 10, "Target value to wait for")
		waitForCmd.Flags().IntP("interval", "i", 1, "Interval in seconds between each check")
		_ = waitForCmd.MarkFlagRequired("target")

		rootCmd.AddCommand(sleepCmd)
		rootCmd.AddCommand(waitCmd)
		rootCmd.AddCommand(waitForCmd)
	}
}

// AddControlFlowBasicCmds registers basic control flow commands: repeat, set, if
// Note: These complement the advanced control flow commands in controlflowcmds.go
func AddControlFlowBasicCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// repeat command - repeats a command multiple times
		var repeatCmd = &cobra.Command{
			Use:   "repeat [--background] [--count {n}]  [--sleep {secs}] {cmd}",
			Short: "Repeats a message a specified number of times with optional delay between each repetition",
			Long: `Repeats the provided message a specified number of times.
You can control the repetition count and the delay between each repetition.

To run indefinitely, set --count to -1.`,
			Example: `
repeat --count 5 --sleep 2 "print This is a custom message;print another message"
repeat --count -1 --sleep 1 "print Infinite loop example"
repeat --background --count 5 --sleep 1 "print alex in background"
repeat --background --count 5 --sleep 1 'client im "uid 11122757" 11122757 hello'
`,
			Args: cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				count, err := cmd.Flags().GetInt("count")
				if err != nil {
					return err
				}

				sleep, err := cmd.Flags().GetInt("sleep")
				if err != nil {
					return err
				}

				bg, err := cmd.Flags().GetBool("background")
				if err != nil {
					return err
				}

				cmdLine := strings.Join(args, " ")

				doExec := func() {
					i := 0
					for count == -1 || i < count {

						res, err := exec.Execute(cmdLine, nil)
						if err != nil {
							cmd.Printf("Error executing command: %s err: %v\n", cmdLine, err)
							continue
						}

						cmd.Printf("Result: %s\n", res)

						if count != -1 {
							i++
						}
						if sleep > 0 {
							time.Sleep(time.Duration(sleep) * time.Second)
						}
					}
				}

				if bg {
					go doExec()
					return nil
				}
				doExec()
				return nil
			},
		}
		repeatCmd.Flags().IntP("count", "c", 1, "Number of times to repeat the message (-1 for infinite)")
		repeatCmd.Flags().IntP("sleep", "s", 0, "Seconds to wait between each repetition")
		repeatCmd.Flags().BoolP("background", "b", false, "run in background")

		// set command - sets a default value for a script param
		var defaultCmd = &cobra.Command{
			Use:     "set [token [value] ]",
			Short:   "set or view default values. If no token is provides, will list all defaults tokens out. If no value is provided, it will print the current default value.",
			Aliases: []string{"default", "def", "block", "set"},

			Run: func(cmd *cobra.Command, args []string) {

				if len(args) == 0 {
					cmd.Printf("defaults: %d\n", exec.Variables.Len())
					exec.Variables.ForEach(func(s string, s2 string) bool {
						cmd.Printf("    %-20s %s\n", s, s2)
						return false
					})
					return
				}

				if len(args) == 1 {
					val, ok := exec.Variables.Get(args[0])
					if !ok {
						cmd.Printf("default not found: %s\n", args[0])
						return
					}
					cmd.Printf("default: %s = %s\n", args[0], val)
					return
				}

				if strings.HasPrefix(args[0], "@") {
					cmd.Printf("default cannot start with @\n")
					return
				}

				key := args[0]
				value := args[1]

				// Use ExpandVariables instead of ExpandCommand to avoid alias expansion
				value = exec.ExpandVariables(cmd, nil, value)

				key = fmt.Sprintf("@%s", key)
				overwrite, _ := cmd.Flags().GetBool("overwrite")
				if overwrite {
					cmd.Printf("overwriting default: %s\n", key)
				} else {
					_, ok := exec.Variables.Get(key)
					if ok {
						cmd.Printf("default already set key: %s\n", key)
						return
					}
					cmd.Printf("setting default: %s\n", key)
				}

				exec.Variables.Set(key, value)
			},
		}
		defaultCmd.Flags().BoolP("overwrite", "o", false, "overwrite default value")

		// if command - conditional execution
		var IfCmdFunc = func(cmd *cobra.Command, args []string) {
			// Evaluate the condition: compare args[0] with args[1]
			iff := args[0] == args[1]

			ifTrue := cmd.Flag("if_true").Value.String()
			ifFalse := cmd.Flag("if_false").Value.String()

			if iff && ifTrue != "" {
				cmd.Printf("Condition true (%s == %s), running: `%s`\n", args[0], args[1], ifTrue)
				res, err := exec.Execute(ifTrue, nil)
				if err != nil {
					cmd.Printf("Error executing command: %s err: %v\n", ifTrue, err)
					return
				}
				if res != "" {
					cmd.Printf("%s\n", res)
				}
				return
			}

			if !iff && ifFalse != "" {
				cmd.Printf("Condition false (%s != %s), running: `%s`\n", args[0], args[1], ifFalse)
				res, err := exec.Execute(ifFalse, nil)
				if err != nil {
					cmd.Printf("Error executing command: %s err: %v\n", ifFalse, err)
					return
				}
				if res != "" {
					cmd.Printf("%s\n", res)
				}
				return
			}

			// No action taken
			cmd.Printf("Condition: %s == %s is %v (no matching command specified)\n", args[0], args[1], iff)
		}

		var ifCmd = &cobra.Command{
			Use:   "if {var} {val}  [--if_true={cmd}] [--if_false={cmd}] [--if_na={cmd}]",
			Short: "if var equals val",
			Args:  cobra.ExactArgs(2),
			Run:   IfCmdFunc,
		}
		ifCmd.Flags().String("if_true", "print test is true", "command to run if true")
		ifCmd.Flags().String("if_false", "print test is false", "command to run if false")
		ifCmd.Flags().String("if_na", "print test is not available", "command to run if not available")

		rootCmd.AddCommand(repeatCmd)
		rootCmd.AddCommand(defaultCmd)
		rootCmd.AddCommand(ifCmd)
	}
}

// FetchURLContent fetches content from a URL and returns the body, status code, and error
var FetchURLContent = func(cmd *cobra.Command, url string) (string, int, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {

	} else {
		url = "http://" + url
	}

	showHeader, _ := cmd.Flags().GetBool("show-headers")
	showDetails, _ := cmd.Flags().GetBool("show-details")
	showStatusCode, _ := cmd.Flags().GetBool("show-status_code")

	// Make the HTTP GET request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch URL: %v err: %v", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if showHeader {
		cmd.Printf("headers: %d\n", len(resp.Header))
		for k, v := range resp.Header {
			cmd.Printf("  %-30s   %s\n", k, v)
		}
		cmd.Printf("%s\n", strings.Repeat("-", 80))
	}
	if showDetails || showStatusCode {
		cmd.Printf("status code: %d\n\n", resp.StatusCode)
	}
	// Check if the HTTP status code is OK
	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("url: %v unexpected status code: %d", url, resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}
	if showDetails {
		cmd.Printf("response len: %d\n", len(body))
	}

	// Return the body as a string
	return string(body), resp.StatusCode, nil
}
