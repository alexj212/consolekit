package consolekit

import (
	"bytes"
	"strings"
	"testing"
)

// TestHelpOmitsGlobalFlags verifies that subcommand help does not render the
// cobra "Global Flags" section. In the REPL those inherited flags are the host's
// persistent startup flags (e.g. --config, --saveDir) and are pure noise on
// every command's help.
func TestHelpOmitsGlobalFlags(t *testing.T) {
	exec, err := NewCommandExecutor("test-app", func(*CommandExecutor) error { return nil })
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	exec.AddCommands(AddWatchCommand(exec))

	root := exec.RootCmd()
	var cfg string
	root.PersistentFlags().StringVarP(&cfg, "config", "c", "qa", "Mode to use: qa | prod")

	sub, _, err := root.Find([]string{"watch"})
	if err != nil || sub == root {
		t.Fatalf("could not find 'watch' subcommand: %v", err)
	}

	var buf bytes.Buffer
	sub.SetOut(&buf)
	if err := sub.Usage(); err != nil {
		t.Fatalf("Usage() error: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "Global Flags") {
		t.Errorf("subcommand help still renders a Global Flags section:\n%s", out)
	}
	if strings.Contains(out, "--config") {
		t.Errorf("inherited --config flag still shown in subcommand help:\n%s", out)
	}
	// Sanity: local flags must still render.
	if !strings.Contains(out, "--count") {
		t.Errorf("expected local --count flag in help, got:\n%s", out)
	}
}
