package consolekit

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// AddVariableCommands adds enhanced variable management commands
func AddVariableCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		// let command - set variables with enhanced features
		letCmd := &cobra.Command{
			Use:   "let [name=value]",
			Short: "Set a variable with arithmetic and command substitution support",
			Long: `Set a variable with enhanced features:
  let name=value           - Simple assignment
  let counter=0            - Numeric values
  let result=$(print hi)   - Command substitution
  let path="$HOME/data"    - Environment variable expansion (not yet implemented)
  let counter=$((counter+1)) - Arithmetic (not yet implemented)`,
			Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				for _, arg := range args {
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) != 2 {
						cmd.PrintErrf("Invalid assignment: %s (expected name=value)\n", arg)
						continue
					}

					name := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Validate variable name
					if !isValidVarName(name) {
						cmd.PrintErrf("Invalid variable name: %s (must start with letter/underscore)\n", name)
						continue
					}

					// Process value - handle command substitution $(...)
					if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
						// Command substitution
						cmdToExec := value[2 : len(value)-1]
						result, err := exec.Execute(cmdToExec, nil)
						if err != nil {
							cmd.PrintErrf("Error executing command '%s': %v\n", cmdToExec, err)
							continue
						}
						value = strings.TrimSpace(result)
					}

					// Store with @ prefix for consistency with token system
					varName := "@" + name
					exec.Variables.Set(varName, value)
					cmd.Printf("%s = %s\n", name, value)
				}
			},
		}

		// unset command - remove variables
		unsetCmd := &cobra.Command{
			Use:   "unset [name...]",
			Short: "Remove one or more variables",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				for _, name := range args {
					varName := "@" + name
					if _, ok := exec.Variables.Get(varName); ok {
						exec.Variables.Delete(varName)
						cmd.Printf("Removed variable: %s\n", name)
					} else {
						cmd.Printf("Variable not found: %s\n", name)
					}
				}
			},
		}

		// vars command - list all variables
		varsCmd := &cobra.Command{
			Use:   "vars",
			Short: "List all variables",
			Run: func(cmd *cobra.Command, args []string) {
				export, _ := cmd.Flags().GetBool("export")
				jsonFormat, _ := cmd.Flags().GetBool("json")

				if jsonFormat {
					exportJSON(cmd, exec)
					return
				}

				if export {
					exportShell(cmd, exec)
					return
				}

				// Default: pretty print
				vars := make(map[string]string)
				exec.Variables.ForEach(func(k, v string) bool {
					if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
						// Remove @ prefix for display
						varName := strings.TrimPrefix(k, "@")
						vars[varName] = v
					}
					return false
				})

				if len(vars) == 0 {
					cmd.Println("No variables set")
					return
				}

				cmd.Println("Variables:")
				cmd.Println(strings.Repeat("-", 60))

				// Sort and display
				exec.Variables.SortedForEach(func(k, v string) bool {
					if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
						varName := strings.TrimPrefix(k, "@")
						// Truncate long values
						displayValue := v
						if len(displayValue) > 50 {
							displayValue = displayValue[:47] + "..."
						}
						cmd.Printf("%-20s = %s\n", varName, displayValue)
					}
					return false
				})
			},
		}
		varsCmd.Flags().Bool("export", false, "Export variables as shell script")
		varsCmd.Flags().Bool("json", false, "Export variables as JSON")

		// increment command - increment a numeric variable
		incCmd := &cobra.Command{
			Use:   "inc [name] [amount]",
			Short: "Increment a numeric variable",
			Long:  "Increment a numeric variable by the specified amount (default: 1)",
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0]
				amount := 1

				if len(args) > 1 {
					var err error
					amount, err = strconv.Atoi(args[1])
					if err != nil {
						cmd.PrintErrf("Invalid amount: %s (must be integer)\n", args[1])
						return
					}
				}

				varName := "@" + name
				currentValue := "0"
				if val, ok := exec.Variables.Get(varName); ok {
					currentValue = val
				}

				current, err := strconv.Atoi(currentValue)
				if err != nil {
					cmd.PrintErrf("Variable %s is not numeric: %s\n", name, currentValue)
					return
				}

				newValue := current + amount
				exec.Variables.Set(varName, strconv.Itoa(newValue))
				cmd.Printf("%s = %d\n", name, newValue)
			},
		}

		// decrement command - decrement a numeric variable
		decCmd := &cobra.Command{
			Use:   "dec [name] [amount]",
			Short: "Decrement a numeric variable",
			Long:  "Decrement a numeric variable by the specified amount (default: 1)",
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0]
				amount := 1

				if len(args) > 1 {
					var err error
					amount, err = strconv.Atoi(args[1])
					if err != nil {
						cmd.PrintErrf("Invalid amount: %s (must be integer)\n", args[1])
						return
					}
				}

				varName := "@" + name
				currentValue := "0"
				if val, ok := exec.Variables.Get(varName); ok {
					currentValue = val
				}

				current, err := strconv.Atoi(currentValue)
				if err != nil {
					cmd.PrintErrf("Variable %s is not numeric: %s\n", name, currentValue)
					return
				}

				newValue := current - amount
				exec.Variables.Set(varName, strconv.Itoa(newValue))
				cmd.Printf("%s = %d\n", name, newValue)
			},
		}

		rootCmd.AddCommand(letCmd)
		rootCmd.AddCommand(unsetCmd)
		rootCmd.AddCommand(varsCmd)
		rootCmd.AddCommand(incCmd)
		rootCmd.AddCommand(decCmd)
	}
}

// isValidVarName checks if a variable name is valid (starts with letter or underscore)
func isValidVarName(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := rune(name[0])
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_'
}

// exportShell exports variables as shell script
func exportShell(cmd *cobra.Command, exec *CommandExecutor) {
	cmd.Println("# Variable export")
	exec.Variables.SortedForEach(func(k, v string) bool {
		if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
			varName := strings.TrimPrefix(k, "@")
			// Escape quotes in value
			escapedValue := strings.ReplaceAll(v, "\"", "\\\"")
			cmd.Printf("export %s=\"%s\"\n", strings.ToUpper(varName), escapedValue)
		}
		return false
	})
}

// exportJSON exports variables as JSON
func exportJSON(cmd *cobra.Command, exec *CommandExecutor) {
	vars := make(map[string]string)
	exec.Variables.ForEach(func(k, v string) bool {
		if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
			varName := strings.TrimPrefix(k, "@")
			vars[varName] = v
		}
		return false
	})

	data, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		cmd.PrintErrf("Error encoding JSON: %v\n", err)
		return
	}

	fmt.Fprintln(os.Stdout, string(data))
}
