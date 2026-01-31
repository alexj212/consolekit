package consolekit

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("HOME", "/home/test")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("HOME")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple env var",
			input:    "$TEST_VAR",
			expected: "test_value",
		},
		{
			name:     "Braced env var",
			input:    "${TEST_VAR}",
			expected: "test_value",
		},
		{
			name:     "Env var in string",
			input:    "Value is $TEST_VAR here",
			expected: "Value is test_value here",
		},
		{
			name:     "Multiple env vars",
			input:    "$HOME/test/$TEST_VAR",
			expected: "/home/test/test/test_value",
		},
		{
			name:     "Undefined env var",
			input:    "$UNDEFINED_VAR",
			expected: "",
		},
		{
			name:     "Mixed braced and simple",
			input:    "${HOME}/$TEST_VAR",
			expected: "/home/test/test_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandConsoleKitVars(t *testing.T) {
	// Create test executor with variables
	exec, err := NewCommandExecutor("test", nil)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	exec.Variables.Set("@name", "John")
	exec.Variables.Set("@count", "42")
	exec.Variables.Set("@path", "/usr/bin")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple variable",
			input:    "@name",
			expected: "John",
		},
		{
			name:     "Variable in string",
			input:    "Hello @name!",
			expected: "Hello John!",
		},
		{
			name:     "Multiple variables",
			input:    "@name has @count items",
			expected: "John has 42 items",
		},
		{
			name:     "Variable in path",
			input:    "@path/executable",
			expected: "/usr/bin/executable",
		},
		{
			name:     "Undefined variable",
			input:    "@undefined",
			expected: "@undefined",
		},
		{
			name:     "Mixed defined and undefined",
			input:    "@name and @undefined",
			expected: "John and @undefined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandConsoleKitVars(tt.input, exec)
			if result != tt.expected {
				t.Errorf("expandConsoleKitVars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandArithmeticVars(t *testing.T) {
	exec, err := NewCommandExecutor("test", nil)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	exec.Variables.Set("@x", "10")
	exec.Variables.Set("@y", "20")
	exec.Variables.Set("@count", "5")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single variable",
			input:    "x",
			expected: "10",
		},
		{
			name:     "Variable in expression",
			input:    "x + y",
			expected: "10 + 20",
		},
		{
			name:     "Multiple variables",
			input:    "x * y + count",
			expected: "10 * 20 + 5",
		},
		{
			name:     "Variable with operators",
			input:    "(x+y)*count",
			expected: "(10+20)*5",
		},
		{
			name:     "Undefined variable",
			input:    "x + undefined",
			expected: "10 + undefined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandArithmeticVars(tt.input, exec)
			if result != tt.expected {
				t.Errorf("expandArithmeticVars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandArithmetic(t *testing.T) {
	exec, err := NewCommandExecutor("test", nil)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	exec.Variables.Set("@x", "10")
	exec.Variables.Set("@y", "5")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple arithmetic",
			input:    "Result: $((5+3))",
			expected: "Result: 8",
		},
		{
			name:     "Multiplication",
			input:    "$((10*5))",
			expected: "50",
		},
		{
			name:     "Complex expression",
			input:    "$((5*3+10/2-1))",
			expected: "19",
		},
		{
			name:     "Variable in arithmetic",
			input:    "$((x*2))",
			expected: "20",
		},
		{
			name:     "Multiple variables",
			input:    "$((x+y))",
			expected: "15",
		},
		{
			name:     "Parentheses",
			input:    "$(( (x+y)*2 ))",
			expected: "30",
		},
		{
			name:     "Division",
			input:    "$((x/y))",
			expected: "2",
		},
		{
			name:     "Modulo",
			input:    "$((x%y))",
			expected: "0",
		},
		{
			name:     "Multiple arithmetic in string",
			input:    "Sum: $((x+y)), Product: $((x*y))",
			expected: "Sum: 15, Product: 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandArithmetic(tt.input, exec)
			if result != tt.expected {
				t.Errorf("expandArithmetic() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEvaluateArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected int
		wantErr  bool
	}{
		{
			name:     "Simple addition",
			expr:     "5+3",
			expected: 8,
		},
		{
			name:     "Subtraction",
			expr:     "10-7",
			expected: 3,
		},
		{
			name:     "Multiplication",
			expr:     "6*7",
			expected: 42,
		},
		{
			name:     "Division",
			expr:     "20/4",
			expected: 5,
		},
		{
			name:     "Modulo",
			expr:     "17%5",
			expected: 2,
		},
		{
			name:     "Complex expression",
			expr:     "5*3+10/2-1",
			expected: 19,
		},
		{
			name:     "Parentheses",
			expr:     "(5+3)*2",
			expected: 16,
		},
		{
			name:     "Nested parentheses",
			expr:     "((5+3)*(2+1))",
			expected: 24,
		},
		{
			name:     "Unary minus",
			expr:     "-5+10",
			expected: 5,
		},
		{
			name:     "Multiple operations",
			expr:     "100/10+5*2-3",
			expected: 17,
		},
		{
			name:    "Division by zero",
			expr:    "10/0",
			wantErr: true,
		},
		{
			name:    "Invalid expression",
			expr:    "5++3",
			wantErr: true,
		},
		{
			name:    "Missing closing paren",
			expr:    "(5+3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateArithmetic(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluateArithmetic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("evaluateArithmetic() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProcessValueExpansions(t *testing.T) {
	// Set test environment
	os.Setenv("TEST_HOME", "/home/test")
	os.Setenv("TEST_USER", "testuser")
	defer os.Unsetenv("TEST_HOME")
	defer os.Unsetenv("TEST_USER")

	exec, err := NewCommandExecutor("test", nil)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	exec.Variables.Set("@counter", "5")
	exec.Variables.Set("@name", "World")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Environment variable",
			input:    "$TEST_HOME/data",
			expected: "/home/test/data",
		},
		{
			name:     "ConsoleKit variable",
			input:    "Hello @name",
			expected: "Hello World",
		},
		{
			name:     "Simple arithmetic",
			input:    "$((5*3))",
			expected: "15",
		},
		{
			name:     "Arithmetic with variable",
			input:    "$((counter*2))",
			expected: "10",
		},
		{
			name:     "Complex mixed",
			input:    "$TEST_USER: @name, count=$((counter+5))",
			expected: "testuser: World, count=10",
		},
		{
			name:     "Quoted string",
			input:    "\"User: $TEST_USER, Home: $TEST_HOME\"",
			expected: "User: testuser, Home: /home/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processValueExpansions(tt.input, exec)
			if err != nil {
				t.Errorf("processValueExpansions() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processValueExpansions() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTokenizeArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected []string
	}{
		{
			name:     "Simple expression",
			expr:     "5+3",
			expected: []string{"5", "+", "3"},
		},
		{
			name:     "With spaces",
			expr:     "5 + 3",
			expected: []string{"5", "+", "3"},
		},
		{
			name:     "Complex",
			expr:     "(10+20)*5",
			expected: []string{"(", "10", "+", "20", ")", "*", "5"},
		},
		{
			name:     "Multiple operators",
			expr:     "100/10-5*2",
			expected: []string{"100", "/", "10", "-", "5", "*", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizeArithmetic(tt.expr)
			if len(result) != len(tt.expected) {
				t.Errorf("tokenizeArithmetic() returned %d tokens, want %d", len(result), len(tt.expected))
				return
			}
			for i, tok := range result {
				if tok != tt.expected[i] {
					t.Errorf("tokenizeArithmetic() token[%d] = %q, want %q", i, tok, tt.expected[i])
				}
			}
		})
	}
}
