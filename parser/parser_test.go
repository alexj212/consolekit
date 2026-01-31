package parser

import (
	"reflect"
	"testing"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantOutputFile string
		wantCmds       int // Number of top-level commands
		wantErr        bool
	}{
		{
			name:           "simple command",
			input:          "echo hello",
			wantOutputFile: "",
			wantCmds:       1,
			wantErr:        false,
		},
		{
			name:           "pipe chain",
			input:          "echo hello | grep h | wc",
			wantOutputFile: "",
			wantCmds:       1, // One chain
			wantErr:        false,
		},
		{
			name:           "output redirection",
			input:          "echo test > output.txt",
			wantOutputFile: "output.txt",
			wantCmds:       1,
			wantErr:        false,
		},
		{
			name:           "quoted string with pipe",
			input:          `echo "hello | world"`,
			wantOutputFile: "",
			wantCmds:       1,
			wantErr:        false,
		},
		{
			name:           "semicolon separator",
			input:          "cmd1; cmd2; cmd3",
			wantOutputFile: "",
			wantCmds:       3,
			wantErr:        false,
		},
		{
			name:           "empty input",
			input:          "",
			wantOutputFile: "",
			wantCmds:       0,
			wantErr:        false,
		},
		{
			name:           "whitespace only",
			input:          "   \t\n  ",
			wantOutputFile: "",
			wantCmds:       0,
			wantErr:        false,
		},
		{
			name:           "comment line",
			input:          "# this is a comment",
			wantOutputFile: "",
			wantCmds:       0,
			wantErr:        false,
		},
		{
			name:           "command with comment",
			input:          "echo test\n# comment\necho test2",
			wantOutputFile: "",
			wantCmds:       2,
			wantErr:        false,
		},
		{
			name:           "backslash continuation",
			input:          "echo hello \\\nworld",
			wantOutputFile: "",
			wantCmds:       1,
			wantErr:        false,
		},
		{
			name:           "multiple output redirections should error",
			input:          "echo test > file1.txt\necho test2 > file2.txt",
			wantOutputFile: "",
			wantCmds:       0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutputFile, gotCmds, err := ParseCommands(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOutputFile != tt.wantOutputFile {
				t.Errorf("ParseCommands() outputFile = %v, want %v", gotOutputFile, tt.wantOutputFile)
			}
			if len(gotCmds) != tt.wantCmds {
				t.Errorf("ParseCommands() got %d commands, want %d", len(gotCmds), tt.wantCmds)
			}
		})
	}
}

func TestPipeChain(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantChain []string // Expected command names in pipe chain
	}{
		{
			name:      "no pipes",
			input:     "echo hello",
			wantChain: []string{"echo"},
		},
		{
			name:      "two commands",
			input:     "echo hello | grep h",
			wantChain: []string{"echo", "grep"},
		},
		{
			name:      "three commands",
			input:     "cat file | grep pattern | wc -l",
			wantChain: []string{"cat", "grep", "wc"},
		},
		{
			name:      "quoted pipe character",
			input:     `echo "test | pipe"`,
			wantChain: []string{"echo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmds, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(cmds) != 1 {
				t.Fatalf("Expected 1 command chain, got %d", len(cmds))
			}

			// Walk the pipe chain
			var got []string
			cmd := cmds[0]
			for cmd != nil {
				got = append(got, cmd.Cmd)
				cmd = cmd.Pipe
			}

			if !reflect.DeepEqual(got, tt.wantChain) {
				t.Errorf("Pipe chain = %v, want %v", got, tt.wantChain)
			}
		})
	}
}

func TestQuoteHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs []string // Expected args for first command
	}{
		{
			name:     "double quotes",
			input:    `echo "hello world"`,
			wantArgs: []string{"hello world"},
		},
		{
			name:     "single quotes",
			input:    `echo 'hello world'`,
			wantArgs: []string{"hello world"},
		},
		{
			name:     "mixed quotes",
			input:    `echo "double" 'single' plain`,
			wantArgs: []string{"double", "single", "plain"},
		},
		{
			name:     "escaped quote",
			input:    `echo \"test\"`,
			wantArgs: []string{`"test"`},
		},
		{
			name:     "nested quotes",
			input:    `echo "outer 'inner' outer"`,
			wantArgs: []string{"outer 'inner' outer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmds, err := ParseCommands(tt.input)
			if err != nil {
				t.Fatalf("ParseCommands() error = %v", err)
			}
			if len(cmds) == 0 {
				t.Fatal("Expected at least one command")
			}

			if !reflect.DeepEqual(cmds[0].Args, tt.wantArgs) {
				t.Errorf("Args = %v, want %v", cmds[0].Args, tt.wantArgs)
			}
		})
	}
}

func TestFindUnquotedChar(t *testing.T) {
	tests := []struct {
		name  string
		input string
		char  rune
		want  int
	}{
		{
			name:  "not quoted",
			input: "echo | grep",
			char:  '|',
			want:  5,
		},
		{
			name:  "in double quotes",
			input: `echo "test | pipe"`,
			char:  '|',
			want:  -1,
		},
		{
			name:  "in single quotes",
			input: `echo 'test | pipe'`,
			char:  '|',
			want:  -1,
		},
		{
			name:  "escaped",
			input: `echo test \| pipe`,
			char:  '|',
			want:  -1,
		},
		{
			name:  "after quotes",
			input: `echo "test" | grep`,
			char:  '|',
			want:  12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUnquotedChar(tt.input, tt.char)
			if got != tt.want {
				t.Errorf("findUnquotedChar() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSplitByUnquotedChar(t *testing.T) {
	tests := []struct {
		name  string
		input string
		char  rune
		want  []string
	}{
		{
			name:  "simple split",
			input: "cmd1; cmd2; cmd3",
			char:  ';',
			want:  []string{"cmd1", " cmd2", " cmd3"},
		},
		{
			name:  "quoted separator",
			input: `echo "test;test"; cmd2`,
			char:  ';',
			want:  []string{`echo "test;test"`, ` cmd2`},
		},
		{
			name:  "no separator",
			input: "echo hello",
			char:  ';',
			want:  []string{"echo hello"},
		},
		{
			name:  "pipe split",
			input: "echo test | grep t | wc",
			char:  '|',
			want:  []string{"echo test ", " grep t ", " wc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByUnquotedChar(tt.input, tt.char)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitByUnquotedChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParseCommands(b *testing.B) {
	inputs := []string{
		"echo hello",
		"echo hello | grep h | wc",
		"echo test > output.txt",
		`echo "hello | world" | grep hello`,
		"cmd1; cmd2; cmd3",
	}

	for _, input := range inputs {
		b.Run(input, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _ = ParseCommands(input)
			}
		})
	}
}
