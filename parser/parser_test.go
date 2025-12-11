package parser

import (
	"testing"
)

func TestParseCommands_SimpleCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmds int
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "simple command no args",
			input:    "help",
			wantCmds: 1,
			wantCmd:  "help",
			wantArgs: []string{},
		},
		{
			name:     "command with single arg",
			input:    "print hello",
			wantCmds: 1,
			wantCmd:  "print",
			wantArgs: []string{"hello"},
		},
		{
			name:     "command with multiple args",
			input:    "print hello world",
			wantCmds: 1,
			wantCmd:  "print",
			wantArgs: []string{"hello", "world"},
		},
		{
			name:     "command with quoted arg",
			input:    `print "hello world"`,
			wantCmds: 1,
			wantCmd:  "print",
			wantArgs: []string{"hello world"},
		},
		{
			name:     "command with single quoted arg",
			input:    `print 'hello world'`,
			wantCmds: 1,
			wantCmd:  "print",
			wantArgs: []string{"hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if outputFile != "" {
				t.Errorf("expected no output file, got %q", outputFile)
			}
			if len(commands) != tt.wantCmds {
				t.Fatalf("got %d commands, want %d", len(commands), tt.wantCmds)
			}
			cmd := commands[0]
			if cmd.Cmd != tt.wantCmd {
				t.Errorf("got cmd %q, want %q", cmd.Cmd, tt.wantCmd)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("got %d args, want %d", len(cmd.Args), len(tt.wantArgs))
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseCommands_QuotedSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "pipe in quotes",
			input:    `print "hello | world"`,
			wantCmd:  "print",
			wantArgs: []string{"hello | world"},
		},
		{
			name:     "redirect in quotes",
			input:    `print "value > 5"`,
			wantCmd:  "print",
			wantArgs: []string{"value > 5"},
		},
		{
			name:     "semicolon in quotes",
			input:    `print "cmd1; cmd2"`,
			wantCmd:  "print",
			wantArgs: []string{"cmd1; cmd2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(commands) != 1 {
				t.Fatalf("got %d commands, want 1", len(commands))
			}
			cmd := commands[0]
			if cmd.Cmd != tt.wantCmd {
				t.Errorf("got cmd %q, want %q", cmd.Cmd, tt.wantCmd)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("got %d args, want %d", len(cmd.Args), len(tt.wantArgs))
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseCommands_Piping(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCmds  []string
		wantPipes int
	}{
		{
			name:      "two commands piped",
			input:     "print hello | grep he",
			wantCmds:  []string{"print", "grep"},
			wantPipes: 1,
		},
		{
			name:      "three commands piped",
			input:     "print hello world | grep hello | grep world",
			wantCmds:  []string{"print", "grep", "grep"},
			wantPipes: 2,
		},
		{
			name:      "pipe with quoted args",
			input:     `print "hello world" | grep "hello"`,
			wantCmds:  []string{"print", "grep"},
			wantPipes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(commands) != 1 {
				t.Fatalf("expected 1 command group, got %d", len(commands))
			}

			// Count pipes
			pipeCount := 0
			current := commands[0]
			cmdIndex := 0

			for current != nil {
				if cmdIndex >= len(tt.wantCmds) {
					t.Fatalf("more commands than expected")
				}
				if current.Cmd != tt.wantCmds[cmdIndex] {
					t.Errorf("command[%d] = %q, want %q", cmdIndex, current.Cmd, tt.wantCmds[cmdIndex])
				}
				if current.Pipe != nil {
					pipeCount++
				}
				current = current.Pipe
				cmdIndex++
			}

			if pipeCount != tt.wantPipes {
				t.Errorf("got %d pipes, want %d", pipeCount, tt.wantPipes)
			}
			if cmdIndex != len(tt.wantCmds) {
				t.Errorf("got %d commands, want %d", cmdIndex, len(tt.wantCmds))
			}
		})
	}
}

func TestParseCommands_Redirection(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantOutputFile string
		wantCmd        string
	}{
		{
			name:           "simple redirect",
			input:          "print hello > output.txt",
			wantOutputFile: "output.txt",
			wantCmd:        "print",
		},
		{
			name:           "redirect with path",
			input:          "print test > /tmp/output.log",
			wantOutputFile: "/tmp/output.log",
			wantCmd:        "print",
		},
		{
			name:           "redirect with spaces",
			input:          "print hello world > result.txt",
			wantOutputFile: "result.txt",
			wantCmd:        "print",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if outputFile != tt.wantOutputFile {
				t.Errorf("got output file %q, want %q", outputFile, tt.wantOutputFile)
			}
			if len(commands) != 1 {
				t.Fatalf("got %d commands, want 1", len(commands))
			}
			if commands[0].Cmd != tt.wantCmd {
				t.Errorf("got cmd %q, want %q", commands[0].Cmd, tt.wantCmd)
			}
		})
	}
}

func TestParseCommands_PipeAndRedirect(t *testing.T) {
	input := "print hello world | grep hello > output.txt"
	outputFile, commands, err := ParseCommands(input)
	if err != nil {
		t.Fatalf("ParseCommands() error = %v", err)
	}

	if outputFile != "output.txt" {
		t.Errorf("got output file %q, want %q", outputFile, "output.txt")
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 command group, got %d", len(commands))
	}

	// Check first command
	if commands[0].Cmd != "print" {
		t.Errorf("first command = %q, want %q", commands[0].Cmd, "print")
	}

	// Check piped command
	if commands[0].Pipe == nil {
		t.Fatal("expected pipe, got nil")
	}
	if commands[0].Pipe.Cmd != "grep" {
		t.Errorf("piped command = %q, want %q", commands[0].Pipe.Cmd, "grep")
	}
}

func TestParseCommands_CommandChaining(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmds []string
	}{
		{
			name:     "two commands with semicolon",
			input:    "print hello ; print world",
			wantCmds: []string{"print", "print"},
		},
		{
			name:     "three commands with semicolon",
			input:    "print a ; print b ; print c",
			wantCmds: []string{"print", "print", "print"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(commands) != len(tt.wantCmds) {
				t.Fatalf("got %d commands, want %d", len(commands), len(tt.wantCmds))
			}
			for i, cmd := range commands {
				if cmd.Cmd != tt.wantCmds[i] {
					t.Errorf("command[%d] = %q, want %q", i, cmd.Cmd, tt.wantCmds[i])
				}
			}
		})
	}
}

func TestParseCommands_Comments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmds int
	}{
		{
			name:     "comment only",
			input:    "# this is a comment",
			wantCmds: 0,
		},
		{
			name:     "command with comment on next line",
			input:    "print hello\n# comment",
			wantCmds: 1,
		},
		{
			name:     "multiple lines with comments",
			input:    "# comment\nprint hello\n# another comment\nprint world",
			wantCmds: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(commands) != tt.wantCmds {
				t.Errorf("got %d commands, want %d", len(commands), tt.wantCmds)
			}
		})
	}
}

func TestParseCommands_MultiLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "backslash continuation",
			input:    "print hello \\\nworld",
			wantCmd:  "print",
			wantArgs: []string{"hello", "world"},
		},
		{
			name:     "multiple backslash continuations",
			input:    "print \\\nhello \\\nworld",
			wantCmd:  "print",
			wantArgs: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, commands, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(commands) != 1 {
				t.Fatalf("got %d commands, want 1", len(commands))
			}
			cmd := commands[0]
			if cmd.Cmd != tt.wantCmd {
				t.Errorf("got cmd %q, want %q", cmd.Cmd, tt.wantCmd)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("got %d args, want %d", len(cmd.Args), len(tt.wantArgs))
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseCommands_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty command",
			input:   "",
			wantErr: false, // Should return empty commands slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseCommands(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
