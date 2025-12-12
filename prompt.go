package consolekit

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Confirm prompts the user for a yes/no confirmation
// Returns true if user confirms (y/yes), false otherwise
func (c *CLI) Confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// Prompt prompts the user for a string input
// Returns the user's input as a string
func (c *CLI) Prompt(message string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response)
}

// PromptDefault prompts the user for a string input with a default value
// Returns the user's input or the default if empty
func (c *CLI) PromptDefault(message string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	return response
}

// PromptPassword prompts the user for a password (hidden input)
// Note: This is a simplified version. For production use, consider using
// golang.org/x/term for proper terminal control
func (c *CLI) PromptPassword(message string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response)
}

// Select prompts the user to select one option from a list
// Returns the selected option or empty string if cancelled
func (c *CLI) Select(message string, options []string) string {
	if len(options) == 0 {
		return ""
	}

	reader := bufio.NewReader(os.Stdin)

	// Display options
	fmt.Println(message)
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select an option (1-", len(options), ", or 0 to cancel): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	response = strings.TrimSpace(response)
	choice, err := strconv.Atoi(response)
	if err != nil || choice < 0 || choice > len(options) {
		return ""
	}

	if choice == 0 {
		return "" // Cancelled
	}

	return options[choice-1]
}

// SelectWithDefault prompts the user to select one option with a default
// Returns the selected option or the default if empty/cancelled
func (c *CLI) SelectWithDefault(message string, options []string, defaultIdx int) string {
	if len(options) == 0 {
		return ""
	}

	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}

	reader := bufio.NewReader(os.Stdin)

	// Display options
	fmt.Println(message)
	for i, option := range options {
		if i == defaultIdx {
			fmt.Printf("  %d) %s (default)\n", i+1, option)
		} else {
			fmt.Printf("  %d) %s\n", i+1, option)
		}
	}
	fmt.Printf("Select an option [%d]: ", defaultIdx+1)

	response, err := reader.ReadString('\n')
	if err != nil {
		return options[defaultIdx]
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return options[defaultIdx]
	}

	choice, err := strconv.Atoi(response)
	if err != nil || choice < 1 || choice > len(options) {
		return options[defaultIdx]
	}

	return options[choice-1]
}

// MultiSelect prompts the user to select multiple options from a list
// Returns the selected options as a slice
func (c *CLI) MultiSelect(message string, options []string) []string {
	if len(options) == 0 {
		return []string{}
	}

	reader := bufio.NewReader(os.Stdin)

	// Display options
	fmt.Println(message)
	fmt.Println("  (Enter numbers separated by spaces, e.g., '1 3 5', or 0 to cancel)")
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select options: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return []string{}
	}

	response = strings.TrimSpace(response)
	if response == "" || response == "0" {
		return []string{}
	}

	// Parse selected indices
	parts := strings.Fields(response)
	selected := make([]string, 0)
	seen := make(map[int]bool)

	for _, part := range parts {
		choice, err := strconv.Atoi(part)
		if err != nil || choice < 1 || choice > len(options) {
			continue
		}

		// Avoid duplicates
		if !seen[choice] {
			selected = append(selected, options[choice-1])
			seen[choice] = true
		}
	}

	return selected
}

// ConfirmDestructive prompts for confirmation of a destructive operation
// Requires the user to type "yes" in full (not just "y")
func (c *CLI) ConfirmDestructive(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s\n", c.ErrorString("WARNING: This is a destructive operation!"))
	fmt.Printf("%s [yes/NO]: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes"
}

// PromptInteger prompts the user for an integer value
// Returns the integer value or the default if invalid
func (c *CLI) PromptInteger(message string, defaultValue int) int {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [%d]: ", message, defaultValue)

	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(response)
	if err != nil {
		return defaultValue
	}

	return value
}
