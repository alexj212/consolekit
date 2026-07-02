package consolekit

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestBuiltinsCoexistWithHostPersistentDashC guards against builtin subcommands
// binding a -c shorthand. Host apps commonly bind a persistent -c (e.g.
// --config qa|prod); cobra merges that persistent flag into every subcommand's
// flagset (via Find -> stripFlags -> mergePersistentFlags), and pflag panics if
// the subcommand already binds -c. This reproduces the path the REPL's syntax
// highlighter takes when the user types one of these commands.
func TestBuiltinsCoexistWithHostPersistentDashC(t *testing.T) {
	exec, err := NewCommandExecutor("test-app", func(*CommandExecutor) error { return nil })
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Register the groups that hold the commands which historically bound -c:
	// repeat (base), highlight + col (format), dedupe (history), watch.
	exec.AddCommands(AddBaseCmds(exec))
	exec.AddCommands(AddFormatCommands(exec))
	exec.AddCommands(AddHistory(exec))
	exec.AddCommands(AddWatchCommand(exec))

	root := exec.RootCmd()

	// Simulate a host app binding a persistent -c, as genrmi2/botserver/ptutils do.
	var cfg string
	root.PersistentFlags().StringVarP(&cfg, "config", "c", "qa", "Mode to use: qa | prod")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a builtin flag shorthand collides with the host persistent -c: %v", r)
		}
	}()

	// Walk every command by its full path and Find it with a trailing probe
	// token. cobra's stripFlags only merges persistent flags into a command when
	// args remain after its name, so the probe forces the merge into that
	// command's flagset — the operation that panics on a shorthand collision.
	var visit func(c *cobra.Command, path []string)
	visit = func(c *cobra.Command, path []string) {
		if len(path) > 0 {
			probe := append(append([]string{}, path...), "__probe__")
			_, _, _ = root.Find(probe) // a collision surfaces as a panic, not an error
		}
		for _, sub := range c.Commands() {
			visit(sub, append(append([]string{}, path...), sub.Name()))
		}
	}
	visit(root, nil)
}
