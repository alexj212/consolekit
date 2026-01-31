package consolekit

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// AddControlFlowCommands adds control flow commands (case, while, for)
func AddControlFlowCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// case command - simple case/switch statement
		var caseCmd = &cobra.Command{
			Use:   "case [value] [pattern1] [command1] [pattern2] [command2] ...",
			Short: "Execute commands based on pattern matching",
			Long: `Execute different commands based on pattern matching (similar to switch/case).
The value is compared against each pattern, and the matching command is executed.
Use '*' as a catch-all pattern.

Examples:
  case "$env" prod "print Production" dev "print Development" "*" "print Unknown"
  case "$(date +%u)" 6 "print Weekend" 7 "print Weekend" "*" "print Weekday"`,
			Args: cobra.MinimumNArgs(3),
			Run: func(cmd *cobra.Command, args []string) {
				value := args[0]
				patterns := args[1:]

				if len(patterns)%2 != 0 {
					cmd.PrintErrln(fmt.Sprintf("case requires pairs of pattern and command"))
					return
				}

				executed := false
				for i := 0; i < len(patterns); i += 2 {
					pattern := patterns[i]
					command := patterns[i+1]

					// Check if pattern matches
					if pattern == "*" || pattern == value {
						output, err := exec.Execute(command, nil)
						if err != nil {
							cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
							return
						}
						if output != "" {
							cmd.Print(output)
							if !strings.HasSuffix(output, "\n") {
								cmd.Println()
							}
						}
						executed = true
						break
					}
				}

				if !executed {
					cmd.PrintErrln(fmt.Sprintf("No matching case found"))
				}
			},
		}

		// while command - execute command while condition is true
		var whileCmd = &cobra.Command{
			Use:   "while [condition_command] [body_command]",
			Short: "Execute command repeatedly while condition is true",
			Long: `Execute a command repeatedly while a condition command succeeds.
The condition command should exit with 0 for true, non-zero for false.
Use with caution - infinite loops are possible!

Examples:
  let i=0; while "test @i -lt 5" "print @i; inc i"
  while "http api.com/status | grep -q pending" "sleep 5s; print Waiting..."`,
			Args: cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				condition := args[0]
				body := args[1]

				maxIterations := 1000 // Safety limit
				iteration := 0

				for iteration < maxIterations {
					// Check condition
					_, err := exec.Execute(condition, nil)
					if err != nil {
						// Condition failed, exit loop
						break
					}

					// Execute body
					output, err := exec.Execute(body, nil)
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Error in loop body: %v", err))
						return
					}

					if output != "" {
						cmd.Print(output)
						if !strings.HasSuffix(output, "\n") {
							cmd.Println()
						}
					}

					iteration++
				}

				if iteration >= maxIterations {
					cmd.PrintErrln(fmt.Sprintf("while loop exceeded maximum iterations (%d)", maxIterations))
				}
			},
		}

		// for command - iterate over values
		var forCmd = &cobra.Command{
			Use:   "for [var] [in] [values...] [do] [command]",
			Short: "Iterate over values and execute command",
			Long: `Execute a command for each value in a list.
The variable is set to each value in turn.

Examples:
  for i in 1 2 3 4 5 do "print Item @i"
  for file in *.txt do "print Processing @file"
  for env in dev qa prod do "print Deploying to @env"`,
			Args: cobra.MinimumNArgs(5),
			Run: func(cmd *cobra.Command, args []string) {
				if len(args) < 5 {
					cmd.PrintErrln(fmt.Sprintf("for requires: var in values... do command"))
					return
				}

				varName := "@" + args[0]
				if args[1] != "in" {
					cmd.PrintErrln(fmt.Sprintf("for requires 'in' keyword"))
					return
				}

				// Find 'do' keyword
				doIndex := -1
				for i := 2; i < len(args); i++ {
					if args[i] == "do" {
						doIndex = i
						break
					}
				}

				if doIndex == -1 {
					cmd.PrintErrln(fmt.Sprintf("for requires 'do' keyword"))
					return
				}

				values := args[2:doIndex]
				command := strings.Join(args[doIndex+1:], " ")

				// Execute command for each value
				for _, value := range values {
					// Set the variable temporarily
					oldValue, hasOld := exec.Variables.Get(varName)
					exec.Variables.Set(varName, value)

					output, err := exec.Execute(command, nil)

					// Restore old value
					if hasOld {
						exec.Variables.Set(varName, oldValue)
					} else {
						exec.Variables.Delete(varName)
					}

					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Error in loop body: %v", err))
						return
					}

					if output != "" {
						cmd.Print(output)
						if !strings.HasSuffix(output, "\n") {
							cmd.Println()
						}
					}
				}
			},
		}

		// test command - test numeric and string conditions
		var testCmd = &cobra.Command{
			Use:   "test [arg1] [operator] [arg2]",
			Short: "Test conditions (for use with while/if)",
			Long: `Test conditions and exit with 0 (true) or 1 (false).
Supports numeric comparisons: -eq, -ne, -lt, -le, -gt, -ge
Supports string comparisons: =, !=

Examples:
  test 5 -lt 10        # 5 less than 10
  test @count -eq 0    # count equals 0
  test "$name" = "John"  # string equality`,
			Args: cobra.ExactArgs(3),
			Run: func(cmd *cobra.Command, args []string) {
				arg1 := args[0]
				operator := args[1]
				arg2 := args[2]

				result := false

				// Try numeric comparison first
				num1, err1 := strconv.Atoi(arg1)
				num2, err2 := strconv.Atoi(arg2)

				if err1 == nil && err2 == nil {
					// Numeric comparison
					switch operator {
					case "-eq":
						result = num1 == num2
					case "-ne":
						result = num1 != num2
					case "-lt":
						result = num1 < num2
					case "-le":
						result = num1 <= num2
					case "-gt":
						result = num1 > num2
					case "-ge":
						result = num1 >= num2
					default:
						cmd.PrintErrln(fmt.Sprintf("Unknown numeric operator: %s", operator))
						os.Exit(1)
						return
					}
				} else {
					// String comparison
					switch operator {
					case "=", "==":
						result = arg1 == arg2
					case "!=":
						result = arg1 != arg2
					default:
						cmd.PrintErrln(fmt.Sprintf("Unknown string operator: %s", operator))
						os.Exit(1)
						return
					}
				}

				if !result {
					os.Exit(1)
				}
			},
		}

		rootCmd.AddCommand(caseCmd)
		rootCmd.AddCommand(whileCmd)
		rootCmd.AddCommand(forCmd)
		rootCmd.AddCommand(testCmd)
	}
}
