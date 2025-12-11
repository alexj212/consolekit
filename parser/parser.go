package parser

import (
	"errors"
	"strings"

	"github.com/kballard/go-shellquote"
)

type ExecCmd struct {
	Cmd  string
	Args []string
	Pipe *ExecCmd
	Line int // Line number for error reporting
}

func (c *ExecCmd) String() string {
	if c.Pipe != nil {
		return c.Cmd + " " + strings.Join(c.Args, " ") + " | " + c.Pipe.String()
	}
	return c.Cmd + " " + strings.Join(c.Args, " ")
}

// ParseCommands processes multi-line input into executable commands
func ParseCommands(input string) (string, []*ExecCmd, error) {
	var outputFile string
	var commands []*ExecCmd

	// Remove comments and handle multi-line commands
	lines := strings.Split(input, "\n")
	filteredLines := []string{}
	var currentLine string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			// If we have accumulated content, save it before skipping comment
			if currentLine != "" {
				filteredLines = append(filteredLines, currentLine)
				currentLine = ""
			}
			continue
		}
		if strings.HasSuffix(line, "\\") {
			currentLine += strings.TrimSuffix(line, "\\") + " "
			continue
		}
		currentLine += line
		if currentLine != "" {
			filteredLines = append(filteredLines, currentLine)
			currentLine = ""
		}
	}

	if len(filteredLines) == 0 {
		return "", commands, nil
	}

	// Process each line
	for _, line := range filteredLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for output redirection using quote-aware parsing
		redirectIdx := findUnquotedChar(line, '>')
		if redirectIdx != -1 {
			if outputFile != "" {
				return "", nil, errors.New("multiple output redirections are not allowed")
			}
			outputFile = strings.TrimSpace(line[redirectIdx+1:])
			line = strings.TrimSpace(line[:redirectIdx])
		}

		// Split the input by ';' to handle multiple command chains (quote-aware)
		commandGroups := splitByUnquotedChar(line, ';')

		for _, group := range commandGroups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}

			// Parse piped commands (quote-aware)
			var prevCmd *ExecCmd
			pipeParts := splitByUnquotedChar(group, '|')
			for _, part := range pipeParts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				// Use shellquote to properly handle quoted arguments
				cmdParts, err := shellquote.Split(part)
				if err != nil {
					return "", nil, errors.New("invalid command syntax: " + err.Error())
				}
				if len(cmdParts) == 0 {
					return "", nil, errors.New("invalid command syntax")
				}

				cmd := &ExecCmd{
					Cmd:  cmdParts[0],
					Args: cmdParts[1:],
				}

				if prevCmd != nil {
					prevCmd.Pipe = cmd
				} else {
					commands = append(commands, cmd)
				}

				prevCmd = cmd
			}
		}
	}

	return outputFile, commands, nil
}

// findUnquotedChar finds the first occurrence of char that's not inside quotes
// Returns -1 if not found
func findUnquotedChar(s string, char rune) int {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i, c := range s {
		if escaped {
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if !inSingleQuote && !inDoubleQuote && c == char {
			return i
		}
	}

	return -1
}

// splitByUnquotedChar splits a string by char, but only at positions where char is not quoted
func splitByUnquotedChar(s string, char rune) []string {
	var result []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, c := range s {
		if escaped {
			current.WriteRune(c)
			escaped = false
			continue
		}

		if c == '\\' {
			current.WriteRune(c)
			escaped = true
			continue
		}

		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteRune(c)
			continue
		}

		if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteRune(c)
			continue
		}

		if !inSingleQuote && !inDoubleQuote && c == char {
			result = append(result, current.String())
			current.Reset()
			continue
		}

		current.WriteRune(c)
	}

	// Add the last part
	if current.Len() > 0 || len(result) > 0 {
		result = append(result, current.String())
	}

	return result
}
