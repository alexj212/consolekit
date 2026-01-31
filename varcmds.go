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
  let path="$HOME/data"    - Environment variable expansion
  let counter=$((counter+1)) - Arithmetic operations
  let "result=$((5 * 3))"  - Use quotes for expressions with spaces`,
			Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				// Join args back together to handle cases where spaces split the expression
				// Then split by unquoted = to find assignments
				fullLine := strings.Join(args, " ")
				assignments := parseAssignments(fullLine)

				for _, assignment := range assignments {
					parts := strings.SplitN(assignment, "=", 2)
					if len(parts) != 2 {
						cmd.PrintErrf("Invalid assignment: %s (expected name=value)\n", assignment)
						continue
					}

					name := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Validate variable name
					if !isValidVarName(name) {
						cmd.PrintErrf("Invalid variable name: %s (must start with letter/underscore)\n", name)
						continue
					}

					// Process value expansions in order:
					// 1. Arithmetic expansion $((...))
					// 2. Command substitution $(...)
					// 3. Environment variable expansion $VAR
					// 4. ConsoleKit variable expansion @var

					var err error
					value, err = processValueExpansions(value, exec)
					if err != nil {
						cmd.PrintErrf("Error processing '%s': %v\n", assignment, err)
						continue
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

// parseAssignments splits a line into individual assignments, handling quotes
func parseAssignments(line string) []string {
	var assignments []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range line {
		switch ch {
		case '"', '\'':
			if inQuote && ch == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = ch
			}
			current.WriteRune(ch)
		case ' ', '\t':
			if inQuote {
				current.WriteRune(ch)
			} else if current.Len() > 0 && i+1 < len(line) && !strings.ContainsRune("=", rune(line[i+1])) {
				// Check if this might be part of a multi-word assignment
				// If we've seen an = and we're in the value part, keep spaces
				if strings.Contains(current.String(), "=") {
					current.WriteRune(ch)
				} else {
					// End of current assignment
					if current.Len() > 0 {
						assignments = append(assignments, current.String())
						current.Reset()
					}
				}
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		assignments = append(assignments, current.String())
	}

	return assignments
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


// evaluateArithmetic evaluates a simple arithmetic expression
// Supports: +, -, *, /, %, (, )
func evaluateArithmetic(expr string) (int, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	// Simple recursive descent parser for arithmetic
	tokens := tokenizeArithmetic(expr)
	if len(tokens) == 0 {
		return 0, fmt.Errorf("no tokens in expression")
	}

	parser := &arithmeticParser{tokens: tokens, pos: 0}
	result, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}

	if parser.pos < len(parser.tokens) {
		return 0, fmt.Errorf("unexpected tokens after expression")
	}

	return result, nil
}

// tokenizeArithmetic splits an arithmetic expression into tokens
func tokenizeArithmetic(expr string) []string {
	var tokens []string
	var current strings.Builder

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		switch ch {
		case ' ', '\t':
			// Skip whitespace
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case '+', '-', '*', '/', '%', '(', ')':
			// Operators and parentheses
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(ch))
		default:
			// Numbers and variable names
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// arithmeticParser implements a simple recursive descent parser
type arithmeticParser struct {
	tokens []string
	pos    int
}

// parseExpression parses addition and subtraction (lowest precedence)
func (p *arithmeticParser) parseExpression() (int, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
		op := p.tokens[p.pos]
		if op != "+" && op != "-" {
			break
		}
		p.pos++

		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}

		if op == "+" {
			left = left + right
		} else {
			left = left - right
		}
	}

	return left, nil
}

// parseTerm parses multiplication, division, and modulo (higher precedence)
func (p *arithmeticParser) parseTerm() (int, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}

	for p.pos < len(p.tokens) {
		op := p.tokens[p.pos]
		if op != "*" && op != "/" && op != "%" {
			break
		}
		p.pos++

		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}

		switch op {
		case "*":
			left = left * right
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left = left / right
		case "%":
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			left = left % right
		}
	}

	return left, nil
}

// parseFactor parses numbers and parenthesized expressions
func (p *arithmeticParser) parseFactor() (int, error) {
	if p.pos >= len(p.tokens) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	token := p.tokens[p.pos]

	// Handle parentheses
	if token == "(" {
		p.pos++
		result, err := p.parseExpression()
		if err != nil {
			return 0, err
		}

		if p.pos >= len(p.tokens) || p.tokens[p.pos] != ")" {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return result, nil
	}

	// Handle unary minus
	if token == "-" {
		p.pos++
		val, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}

	// Handle numbers
	p.pos++
	val, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", token)
	}

	return val, nil
}
