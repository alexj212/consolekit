package consolekit

import (
	"fmt"
	"github.com/alexj212/consolekit/safemap"
	"github.com/spf13/cobra"
	"io"
	"net/http"

	"reflect"
	"regexp"

	"strconv"
	"strings"
	"time"
)

const ClsSeq = "\033[H\033[2J"

func (c *CLI) AddMiscCmds() {
	// Set up flags for each command
	waitCmd.Flags().StringP("time", "t", "", "Time to wait until in HH:MM format (24-hour)")
	waitCmd.MarkFlagRequired("time")

	repeatCmd.Flags().IntP("count", "c", 1, "Number of times to repeat the message (-1 for infinite)")
	repeatCmd.Flags().IntP("sleep", "s", 0, "Seconds to wait between each repetition")
	repeatCmd.Flags().BoolP("background", "b", false, "run in background")

	checkCmd.Flags().StringP("field", "f", "", "Field name to check in the struct")
	checkCmd.Flags().StringP("value", "v", "", "Expected value of the field")
	checkCmd.MarkFlagRequired("field")
	checkCmd.MarkFlagRequired("value")

	waitForCmd.Flags().IntP("target", "t", 10, "Target value to wait for")
	waitForCmd.Flags().IntP("interval", "i", 1, "Interval in seconds between each check")
	waitForCmd.MarkFlagRequired("target")

	ifCmd.Flags().String("if_true", "print test is true", "command to run if true")
	ifCmd.Flags().String("if_false", "print test is false", "command to run if false")
	ifCmd.Flags().String("if_na", "print test is not available", "command to run if not available")

	c.RootCmd.AddCommand(checkCmd)
	c.RootCmd.AddCommand(clsCmd)
	c.RootCmd.AddCommand(defaultCmd)
	c.RootCmd.AddCommand(exitCmd)
	c.RootCmd.AddCommand(historyCmd)
	c.RootCmd.AddCommand(httpCmd)
	c.RootCmd.AddCommand(ifCmd)
	c.RootCmd.AddCommand(osexecCmd)
	c.RootCmd.AddCommand(printCmd)
	c.RootCmd.AddCommand(repeatCmd)
	c.RootCmd.AddCommand(sleepCmd)
	c.RootCmd.AddCommand(testCmd)
	c.RootCmd.AddCommand(verCmd)
	c.RootCmd.AddCommand(waitCmd)
	c.RootCmd.AddCommand(waitForCmd)
	c.RootCmd.AddCommand(viewScriptCmd)
	c.RootCmd.AddCommand(runScriptCmd)
	c.RootCmd.AddCommand(spawnScriptCmd)

}

var clsCmd = &cobra.Command{
	Use:   "cls",
	Short: "Clear the screen",
	Long:  `Clear the screen`,
	Run:   clsCmdFunc,
}

var clsCmdFunc = func(cmd *cobra.Command, args []string) {

	cmd.Printf(ClsSeq)

}

var verCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v", "ver"},
	Short:   "Show version info",
	Run:     verCmdFunc,
}

var verCmdFunc = func(cmd *cobra.Command, args []string) {
	cmd.Printf("BuildDate    : %s\n", cli.BuildDate)
	cmd.Printf("LatestCommit : %s\n", cli.LatestCommit)
	cmd.Printf("Version      : %s\n", cli.Version)
	cmd.Printf("GitRepo      : %s\n", cli.GitRepo)
	cmd.Printf("GitBranch    : %s\n", cli.GitBranch)

}

var exitCmd = &cobra.Command{
	Use:     "exit {code}",
	Short:   "Exit the program",
	Aliases: []string{"x", "quit", "q", "x"},
	Long: `exit the program
`,
	Args: cobra.MaximumNArgs(1),
	Run:  exitCmdFunc,
}

var exitCmdFunc = func(cmd *cobra.Command, args []string) {
	code := 0

	if len(args) > 0 {
		code, _ = strconv.Atoi(args[0])
	}
	cli.Exit("exit cmd", code)

} //exitCmdFunc

var printCmd = &cobra.Command{
	Use:   "print {message}",
	Short: "print message",
	Run:   printCmdFunc,
}

var printCmdFunc = func(cmd *cobra.Command, args []string) {
	line := strings.Join(args, " ")

	line = ReplaceDefaults(cmd, line)
	cmd.Printf("%s\n", line)
}

func ReplaceDefaults(cmd *cobra.Command, input string) string {
	defaults.ForEach(func(k string, v string) bool {

		input = strings.ReplaceAll(input, k, v)
		return false
	})
	// Regular expression to split by spaces, but keep quoted sections intact
	re := regexp.MustCompile(`"[^"]*"|\S+`)
	words := re.FindAllString(input, -1)

	for i, word := range words {
		// Check for tokens inside quotes but retain quotes
		if strings.HasPrefix(word, `"@`) {
			// Remove quotes, replace token, then re-wrap with quotes
			cleanWord := strings.Trim(word, `"`)
			words[i] = `"` + replaceToken(cmd, cleanWord) + `"`
		} else if strings.HasPrefix(word, "@") {
			// Replace token directly if not in quotes
			words[i] = replaceToken(cmd, word)
		}
	}

	return strings.Join(words, " ")
} //ReplaceDefaults

// replaceToken handles token replacement. Modify this function as needed.
func replaceToken(cmd *cobra.Command, token string) string {

	v, ok := defaults.Get(token)
	if ok {
		cmd.Printf("replaceToken: %s -> %s\n", token, v)
		return v
	}
	return token
}

/*
	defaults.ForEach(func(k string, v string) bool {
		line = strings.ReplaceAll(line, k, v)
		return false
	})

//
// @uid:xyz,ck

return line
*/
var httpCmd = &cobra.Command{
	Use:   "http {url}",
	Short: "http call url",
	Run:   httpCmdFunc,
	Args:  cobra.ExactArgs(1),
}

var httpCmdFunc = func(cmd *cobra.Command, args []string) {

	cmd.Printf("http call to %s\n", args[0])
	data, err := FetchURLContent(args[0])
	if err != nil {
		cmd.Printf("error fetching url: %v\n", err)
		return
	}
	cmd.Printf("data: %s\n", data)

} //httpCmdFunc

var LocalCmd = &cobra.Command{
	Use:   "local",
	Short: "switch to local mode",
	Run: func(cmd *cobra.Command, args []string) {

	},
	Args: cobra.ExactArgs(0),
}

var sleepCmd = &cobra.Command{
	Use:     "sleep {secs}",
	Short:   "sleep {n} seconds",
	Example: "sleep 5",
	Run: func(cmd *cobra.Command, args []string) {

		delay, err := strconv.Atoi(args[0])
		if err != nil {
			cmd.Printf("Invalid delay %s\n", args[0])
			return
		}
		cmd.Printf("Sleeping for %d seconds\n", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	},
} //sleepCmd

// waitCmd pauses execution until a specified time
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

// repeatCmd repeats a message a specified number of times
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

				cmdLine = ReplaceDefaults(cmd, cmdLine)
				cmds := strings.Split(cmdLine, ";")
				for _, c := range cmds {
					res, err := cli.ExecuteCommands(c)
					if err != nil {
						cmd.Printf("Error executing command: %s err: %v\n", c, err)
						continue
					}
					cmd.Printf("Result: %s\n", res)
				}

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
var data any

// checkCmd checks if a struct field matches a provided value
var checkCmd = &cobra.Command{
	Use:   "check --field FIELD_NAME --value VALUE",
	Short: "Checks if the specified field in a struct matches the provided value",
	Long: `Checks if a field in the Person struct has the specified value.
You can supply the field name and the expected value to verify against the struct.`,
	Example: ` check --field Name --value Alice
 check --field Age --value 25
 check --field Email --value "alice@example.com"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fieldName, err := cmd.Flags().GetString("field")
		if err != nil || fieldName == "" {
			return fmt.Errorf("please provide a valid field name with --field")
		}

		fieldValue, err := cmd.Flags().GetString("value")
		if err != nil || fieldValue == "" {
			return fmt.Errorf("please provide a valid value with --value")
		}

		pValue := reflect.ValueOf(data)
		pType := pValue.Type()

		found := false
		for i := 0; i < pType.NumField(); i++ {
			field := pType.Field(i)
			if strings.EqualFold(field.Name, fieldName) {
				found = true
				fieldVal := pValue.FieldByName(field.Name)
				if fmt.Sprint(fieldVal.Interface()) == fieldValue {
					cmd.Printf("Match found: %s = %s\n", field.Name, fieldValue)
				} else {
					cmd.Printf("No match: %s is %v, expected %s\n", field.Name, fieldVal.Interface(), fieldValue)
				}
				break
			}
		}

		if !found {
			cmd.Printf("Field %s not found in Person struct\n", fieldName)
		}

		return nil
	},
}

// waitForCmd waits until a condition is met
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

var defaults = safemap.New[string, string]()

// defaultCmd sets a default value for a script param.
var defaultCmd = &cobra.Command{
	Use:     "default {token} {value}",
	Short:   "sets a default value for a script param.",
	Example: ` default default_user mario`,
	Aliases: []string{"def", "block"},
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		key = fmt.Sprintf("@%s", key)
		_, ok := defaults.Get(key)
		if ok {
			cmd.Printf("default already set: %s = %s\n", key, value)
			return
		}
		defaults.Set(key, value)
		cmd.Printf("default set: %s = %s\n", key, value)
	},
}

var ifCmd = &cobra.Command{
	Use:   "if {var} {val}  [--if_true={cmd}] [--if_false={cmd}] [--if_na={cmd}]",
	Short: "if var equals val",
	Args:  cobra.ExactArgs(2),
	Run:   IfCmdFunc,
}

var IfCmdFunc = func(cmd *cobra.Command, args []string) {
	cmd.Printf("IfCmdFunc expr: `%s` val: `%s`\n", args[0], args[1])
	cmd.Printf("if_true : `%s`\n", cmd.Flag("if_true").Value.String())
	cmd.Printf("if_false: `%s`\n", cmd.Flag("if_false").Value.String())
	cmd.Printf("if_na   : `%s`\n", cmd.Flag("if_na").Value.String())
	iff := true

	ifTrue := cmd.Flag("if_true").Value.String()
	if ifTrue != "" && iff {
		cmd.Printf("running if_true: `%s`\n", ifTrue)

		ifTrue = ReplaceDefaults(cmd, ifTrue)
		cmds := strings.Split(ifTrue, ";")
		for _, c := range cmds {
			cmd.Printf("running if_true: `%s`\n", c)
			res, err := cli.ExecuteCommands(c)
			if err != nil {
				cmd.Printf("Error executing command: %s err: %v\n", c, err)
				continue
			}
			cmd.Printf("Result: %s\n", res)
		}
		return
	}
	ifFalse := cmd.Flag("if_false").Value.String()
	if ifFalse != "" && !iff {
		cmd.Printf("running if_false: `%s`\n", ifFalse)

		ifFalse = ReplaceDefaults(cmd, ifFalse)
		cmds := strings.Split(ifFalse, ";")
		for _, c := range cmds {
			cmd.Printf("running if_false: `%s`\n", c)
			res, err := cli.ExecuteCommands(c)
			if err != nil {
				cmd.Printf("Error executing command: %s err: %v\n", c, err)
				continue
			}
			cmd.Printf("Result: %s\n", res)

		}
		return
	}

	cmd.Printf("if_na: `%s`\n", cmd.Flag("if_na").Value.String())

}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test related commands",
}

func FetchURLContent(url string) (string, error) {
	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Check if the HTTP status code is OK
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Return the body as a string
	return string(body), nil
}
