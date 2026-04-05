// simple_reeflective demonstrates using reeflective/console directly
// (without consolekit) to build a basic interactive REPL with cobra commands.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func main() {
	// Create console application
	app := console.New("demo")

	// Get the default menu and configure prompt
	menu := app.ActiveMenu()
	prompt := menu.Prompt()
	prompt.Primary = func() string { return "demo > " }

	// Handle Ctrl+D (EOF) to exit cleanly
	menu.AddInterrupt(io.EOF, func(c *console.Console) {
		fmt.Println("Goodbye!")
		os.Exit(0)
	})

	// Register commands - this function is called before each execution
	// so commands get a fresh state (important for flag reuse in REPL)
	menu.SetCommands(func() *cobra.Command {
		root := &cobra.Command{}

		root.AddCommand(helloCmd())
		root.AddCommand(echoCmd())
		root.AddCommand(dateCmd())
		root.AddCommand(exitCmd())

		return root
	})

	fmt.Println("Simple reeflective/console demo. Type 'help' for commands.")

	// Start the REPL (blocking)
	if err := app.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func helloCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hello [name]",
		Short: "Greet someone",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "World"
			if len(args) > 0 {
				name = args[0]
			}
			greeting, _ := cmd.Flags().GetString("greeting")
			fmt.Printf("%s, %s!\n", greeting, name)
		},
	}
	cmd.Flags().String("greeting", "Hello", "Custom greeting text")
	return cmd
}

func echoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "echo [text...]",
		Short: "Echo back the provided text",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(strings.Join(args, " "))
		},
	}
}

func dateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "date",
		Short: "Show current date and time",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
		},
	}
}

func exitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exit",
		Short: "Exit the application",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Goodbye!")
			os.Exit(0)
		},
	}
}
