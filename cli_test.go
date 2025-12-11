package consolekit

import (
	"strings"
	"testing"
)

// TestNewCLI tests CLI initialization
func TestNewCLI(t *testing.T) {
	cli, err := NewCLI("test", nil)
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	if cli.AppName != "test" {
		t.Errorf("Expected AppName 'test', got '%s'", cli.AppName)
	}

	if cli.JobManager == nil {
		t.Error("JobManager should be initialized")
	}

	if cli.Config == nil {
		t.Error("Config should be initialized")
	}

	if cli.Defaults == nil {
		t.Error("Defaults should be initialized")
	}
}

// TestExecuteLinePrint tests basic command execution
func TestExecuteLinePrint(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	output, err := cli.ExecuteLine("print hello", nil)
	if err != nil {
		t.Fatalf("ExecuteLine failed: %v", err)
	}

	if !strings.Contains(output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", output)
	}
}

// TestExecuteLinePiping tests command piping
func TestExecuteLinePiping(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		AddMisc()
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Test piping print to grep
	output, err := cli.ExecuteLine("print \"line1\\nline2\\nline3\" | grep line2", nil)
	if err != nil {
		t.Fatalf("ExecuteLine with piping failed: %v", err)
	}

	if !strings.Contains(output, "line2") {
		t.Errorf("Expected output to contain 'line2', got: %s", output)
	}

	if strings.Contains(output, "line1") || strings.Contains(output, "line3") {
		t.Errorf("Output should only contain 'line2', got: %s", output)
	}
}

// TestTokenReplacement tests variable token replacement
func TestTokenReplacement(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		AddVariableCommands(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Set a variable
	_, err = cli.ExecuteLine("let myvar=hello", nil)
	if err != nil {
		t.Fatalf("Setting variable failed: %v", err)
	}

	// Use the variable
	output, err := cli.ExecuteLine("print @myvar", nil)
	if err != nil {
		t.Fatalf("ExecuteLine with token replacement failed: %v", err)
	}

	if !strings.Contains(output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", output)
	}
}

// TestCommandChaining tests command chaining with semicolons
func TestCommandChaining(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		AddVariableCommands(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Execute multiple commands
	output, err := cli.ExecuteLine("let x=5 ; print @x", nil)
	if err != nil {
		t.Fatalf("Command chaining failed: %v", err)
	}

	if !strings.Contains(output, "5") {
		t.Errorf("Expected output to contain '5', got: %s", output)
	}
}

// TestRecursionProtection tests that recursion is limited
func TestRecursionProtection(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Set up a recursive token
	cli.Defaults.Set("@test", "@exec:print @test")

	// Try to execute - should hit recursion limit
	_, err = cli.ExecuteLine("print @test", nil)
	if err == nil {
		t.Error("Expected recursion error, got nil")
	}

	if !strings.Contains(err.Error(), "recursion") {
		t.Errorf("Expected recursion error, got: %v", err)
	}
}

// TestAliasReplacement tests alias replacement
func TestAliasReplacement(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		AddAlias(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Set an alias
	_, err = cli.ExecuteLine("alias hw=\"print hello world\"", nil)
	if err != nil {
		t.Fatalf("Setting alias failed: %v", err)
	}

	// Use the alias
	output, err := cli.ExecuteLine("hw", nil)
	if err != nil {
		t.Fatalf("Executing alias failed: %v", err)
	}

	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", output)
	}
}

// TestEnvTokenReplacement tests environment variable replacement
func TestEnvTokenReplacement(t *testing.T) {
	cli, err := NewCLI("test", func(c *CLI) error {
		AddBaseCmds(c)
		return nil
	})
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	// Set an environment variable
	t.Setenv("TEST_VAR", "testvalue")

	// Use the environment variable
	output, err := cli.ExecuteLine("print @env:TEST_VAR", nil)
	if err != nil {
		t.Fatalf("ExecuteLine with env token failed: %v", err)
	}

	if !strings.Contains(output, "testvalue") {
		t.Errorf("Expected output to contain 'testvalue', got: %s", output)
	}
}
