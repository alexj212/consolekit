package main

import (
	"testing"

	"github.com/alexj212/consolekit"
)

func TestBubbletteaAdapter_Creation(t *testing.T) {
	adapter := NewBubbletteaAdapter("test")

	if adapter == nil {
		t.Fatal("NewBubbletteaAdapter returned nil")
	}

	if adapter.appName != "test" {
		t.Errorf("Expected appName 'test', got '%s'", adapter.appName)
	}

	// Verify it implements DisplayAdapter interface
	var _ consolekit.DisplayAdapter = adapter
}

func TestBubbletteaAdapter_SetPrompt(t *testing.T) {
	adapter := NewBubbletteaAdapter("test")

	customPrompt := "custom > "
	adapter.SetPrompt(func() string {
		return customPrompt
	})

	result := adapter.promptFunc()
	if result != customPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", customPrompt, result)
	}
}

func TestBubbletteaAdapter_SetHistoryFile(t *testing.T) {
	adapter := NewBubbletteaAdapter("test")

	histFile := "/tmp/test.history"
	adapter.SetHistoryFile(histFile)

	if adapter.historyFile != histFile {
		t.Errorf("Expected historyFile '%s', got '%s'", histFile, adapter.historyFile)
	}
}

func TestBubbletteaAdapter_SetExecutor(t *testing.T) {
	adapter := NewBubbletteaAdapter("test")

	executor, err := consolekit.NewCommandExecutor("test", nil)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	adapter.SetExecutor(executor)

	if adapter.executor != executor {
		t.Error("Executor not set correctly")
	}
}

func TestBubbletteaAdapter_Configure(t *testing.T) {
	adapter := NewBubbletteaAdapter("test")

	config := consolekit.DisplayConfig{
		AppName: "custom_app",
	}

	adapter.Configure(config)

	if adapter.config.AppName != "custom_app" {
		t.Errorf("Expected AppName 'custom_app', got '%s'", adapter.config.AppName)
	}
}
