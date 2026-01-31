package consolekit

import (
	"context"
	"strings"
	"testing"

	"github.com/alexj212/consolekit/safemap"
)

func TestExecute(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name    string
		line    string
		wantErr bool
		wantOut string // Substring to check in output
	}{
		{
			name:    "simple print",
			line:    "print hello",
			wantErr: false,
			wantOut: "hello",
		},
		{
			name:    "print with quotes",
			line:    `print "hello world"`,
			wantErr: false,
			wantOut: "hello world",
		},
		{
			name:    "empty command",
			line:    "",
			wantErr: false,
			wantOut: "",
		},
		{
			name:    "whitespace only",
			line:    "   \t  ",
			wantErr: false,
			wantOut: "",
		},
		{
			name:    "unknown command",
			line:    "unknowncommand123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(tt.line, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantOut != "" && !strings.Contains(output, tt.wantOut) {
				t.Errorf("Execute() output = %q, want to contain %q", output, tt.wantOut)
			}
		})
	}
}

func TestTokenReplacement(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		// Set a test variable
		exec.Variables.Set("@testvar", "testvalue")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name    string
		line    string
		defs    *safemap.SafeMap[string, string]
		wantOut string
	}{
		{
			name:    "basic variable replacement",
			line:    "print @testvar",
			defs:    nil,
			wantOut: "testvalue",
		},
		{
			name: "scoped variable",
			line: "print @scopedvar",
			defs: func() *safemap.SafeMap[string, string] {
				m := safemap.New[string, string]()
				m.Set("@scopedvar", "scopedvalue")
				return m
			}(),
			wantOut: "scopedvalue",
		},
		{
			name:    "no variable",
			line:    "print nomatch",
			defs:    nil,
			wantOut: "nomatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(tt.line, tt.defs)
			if err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
			}
			if !strings.Contains(output, tt.wantOut) {
				t.Errorf("Execute() output = %q, want to contain %q", output, tt.wantOut)
			}
		})
	}
}

func TestAliasExpansion(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Set up an alias
	executor.aliases.Set("hw", "print hello")

	output, err := executor.Execute("hw", nil)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Alias expansion failed: got %q, want to contain 'hello'", output)
	}
}

func TestPipeExecution(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{
			name:    "simple pipe",
			line:    `print "line1\nline2\nline3" | grep line2`,
			wantErr: false,
		},
		{
			name:    "multiple pipes",
			line:    `print "test\ndata\nhere" | grep test | grep test`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(tt.line, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Test context cancellation before command execution
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = executor.ExecuteWithContext(ctx, "print hello", nil)
	if err == nil {
		t.Error("Expected context cancelled error, got nil")
		return
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
		t.Errorf("Expected context/cancel error, got: %v", err)
	}
}

func TestRecursionProtection(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Create circular token reference using @exec:
	// When @recursive is expanded, it triggers another execution of the same variable
	executor.Variables.Set("@recursive", "@exec:print @recursive")

	output, err := executor.Execute("print @recursive", nil)
	// The recursion protection works by limiting depth (maxExecDepth = 10)
	// After 10 recursive calls, it stops but doesn't propagate the error through @exec: token replacement
	// So we check that it stopped after a reasonable number of iterations (10 newlines = 10 executions)
	newlineCount := strings.Count(output, "\n")
	if newlineCount != 10 {
		t.Errorf("Expected recursion to stop after 10 iterations, got %d newlines in output: %q", newlineCount, output)
	}
	if err != nil {
		t.Logf("Got error (expected behavior): %v", err)
	}
}

func TestSemicolonSeparator(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Multiple commands separated by semicolon
	output, err := executor.Execute("print hello; print world", nil)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("Semicolon separator failed: got %q", output)
	}
}

func TestFileRedirection(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Test output redirection
	_, err = executor.Execute("print test > /tmp/consolekit-test-output.txt", nil)
	if err != nil {
		t.Errorf("File redirection failed: %v", err)
	}

	// Read back to verify
	output, err := executor.Execute("cat /tmp/consolekit-test-output.txt", nil)
	if err != nil {
		t.Errorf("Failed to read redirected file: %v", err)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("File redirection content mismatch: got %q, want to contain 'test'", output)
	}
}

func BenchmarkExecute(b *testing.B) {
	executor, err := NewCommandExecutor("bench-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		b.Fatalf("Failed to create executor: %v", err)
	}

	commands := []string{
		"print hello",
		"print @test",
		`print "hello world"`,
		"print test | grep test",
	}

	for _, cmd := range commands {
		b.Run(cmd, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = executor.Execute(cmd, nil)
			}
		})
	}
}

func TestConcurrentExecution(t *testing.T) {
	executor, err := NewCommandExecutor("test-app", func(exec *CommandExecutor) error {
		exec.AddBuiltinCommands()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Test concurrent command execution (simulating multiple SSH sessions)
	const concurrent = 10
	done := make(chan error, concurrent)

	for i := 0; i < concurrent; i++ {
		go func(id int) {
			_, err := executor.Execute("print test", nil)
			done <- err
		}(i)
	}

	for i := 0; i < concurrent; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent execution %d failed: %v", i, err)
		}
	}
}
