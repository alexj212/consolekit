package consolekit

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// processValueExpansions handles all value expansions in order
func processValueExpansions(value string, exec *CommandExecutor) (string, error) {
	// Remove surrounding quotes if present
	value = strings.Trim(value, "\"'")

	// Process in order, but only once each to avoid infinite loops
	// 1. Arithmetic expansion $((...))
	value = expandArithmetic(value, exec)

	// 2. Command substitution $(...)
	value = expandCommandSubstitution(value, exec)

	// 3. Environment variable expansion $VAR or ${VAR}
	value = expandEnvVars(value)

	// 4. ConsoleKit variable expansion @var
	value = expandConsoleKitVars(value, exec)

	return value, nil
}

// expandArithmetic handles $((...)) arithmetic expressions
func expandArithmetic(value string, exec *CommandExecutor) string {
	// Find all $((...)) patterns - need to handle nested parentheses
	// Use a manual parser instead of regex for better handling
	result := value
	for {
		start := strings.Index(result, "$((")
		if start == -1 {
			break
		}

		// Find matching ))
		depth := 0
		end := -1
		for i := start + 3; i < len(result)-1; i++ {
			if result[i] == '(' {
				depth++
			} else if result[i] == ')' {
				if result[i+1] == ')' && depth == 0 {
					end = i + 2
					break
				}
				if depth > 0 {
					depth--
				}
			}
		}

		if end == -1 {
			// No matching )), skip this one
			break
		}

		// Extract and evaluate
		expr := result[start+3 : end-2]
		expr = expandArithmeticVars(expr, exec)

		evalResult, err := evaluateArithmetic(expr)
		if err != nil {
			evalResult = 0 // Return 0 on error
		}

		result = result[:start] + strconv.Itoa(evalResult) + result[end:]
	}

	return result
}

// expandArithmeticVars expands variable names (without @) in arithmetic expressions
func expandArithmeticVars(expr string, exec *CommandExecutor) string {
	// Pattern for variable names (letters/underscore followed by alphanumeric)
	// that are NOT already prefixed with @
	pattern := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\b`)
	return pattern.ReplaceAllStringFunc(expr, func(match string) string {
		// Try to get the variable with @ prefix
		varName := "@" + match
		if val, ok := exec.Variables.Get(varName); ok {
			return val
		}
		// If not found, return original (might be a number or operator)
		return match
	})
}

// expandCommandSubstitution handles $(...) command substitution
func expandCommandSubstitution(value string, exec *CommandExecutor) string {
	// Pattern that doesn't match $((...))
	pattern := regexp.MustCompile(`\$\(([^(][^)]*)\)`)

	// Find all matches first
	matches := pattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value
	}

	// Process in reverse to maintain indices
	result := value
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		cmdToExec := value[match[2]:match[3]]

		cmdResult, err := exec.Execute(cmdToExec, nil)
		if err != nil {
			continue // Skip on error
		}

		result = result[:match[0]] + strings.TrimSpace(cmdResult) + result[match[1]:]
	}

	return result
}

// expandEnvVars expands environment variables in the format $VAR or ${VAR}
func expandEnvVars(value string) string {
	// Pattern for ${VAR}
	bracePattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	value = bracePattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1]
		return os.Getenv(varName)
	})

	// Pattern for $VAR (simpler, just alphanumeric after $)
	simplePattern := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	value = simplePattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[1:]
		return os.Getenv(varName)
	})

	return value
}

// expandConsoleKitVars expands ConsoleKit variables in the format @var
func expandConsoleKitVars(value string, exec *CommandExecutor) string {
	pattern := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)`)
	return pattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := match // Keep @ prefix
		if val, ok := exec.Variables.Get(varName); ok {
			return val
		}
		return match // Return original if not found
	})
}
