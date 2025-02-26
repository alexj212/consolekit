package consolekit

import (
	"bufio"
	"fmt"
	"github.com/alexj212/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ResetAllFlags Function to reset all flags to their default values
func ResetAllFlags(cmd *cobra.Command) {
	//Printf("LocalRootReplCmd resetAllFlags %s\n", cmd.Use)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
	})
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
	})

	for _, subCmd := range cmd.Commands() {
		ResetAllFlags(subCmd)
	}
}

// SetRecursiveHelpFunc function to set custom HelpFunc for a command and all its subcommands
func SetRecursiveHelpFunc(cmd *cobra.Command) {
	// Store the original help function
	originalHelpFunc := cmd.HelpFunc()

	// Set a new HelpFunc that includes resetting flags after help is shown
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		// Display the original help message
		originalHelpFunc(c, args)
		// After help is displayed, reset all flags to their default values
		ResetAllFlags(c)
	})

	// Recursively apply this to all subcommands
	for _, subCmd := range cmd.Commands() {
		SetRecursiveHelpFunc(subCmd)
	}
}

// ResetHelpFlag resets the help flag for a single command
func ResetHelpFlag(cmd *cobra.Command) {
	helpFlag := cmd.Flags().Lookup("help")
	if helpFlag != nil && helpFlag.Changed {
		helpFlag.Value.Set("false")
		helpFlag.Changed = false
	}
}

// ResetHelpFlagRecursively resets the help flag for a command and all of its children
func ResetHelpFlagRecursively(cmd *cobra.Command) {
	ResetHelpFlag(cmd) // Reset help flag for the current command

	for _, childCmd := range cmd.Commands() {
		ResetHelpFlagRecursively(childCmd) // Recursively reset help flag for each child
	}
}

func (c *CLI) AppBlock() error {

	return c.Repl.Start()
}

// ExitCtrlD is a custom interrupt handler to use when the shell
// readline receives an io.EOF error, which is returned with CtrlD.
func (c *CLI) ExitCtrlD(conc *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	conc.Printf("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		c.Exit("exitCtrlD", 0)
	}
}

func (c *CLI) Exit(caller string, code int) {
	if c.OnExit != nil {
		c.OnExit(caller, code)
	}

	if code != 0 {
		c.Repl.Printf("%s: exiting with code %d\n", caller, code)
	}
	time.Sleep(250 * time.Millisecond)
	os.Exit(code)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func (c *CLI) Execute() {

	// Channel to signal shutdown
	quit := make(chan struct{})

	// Start console in a separate goroutine
	go startConsole(quit)
	//err = display.Start("genrmi2", quit)
	//if err != nil {
	//	Printf("error starting display: %v\n", err)
	//}
	<-quit
	RestoreTerminal()

	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func RestoreTerminal() {
	echoCmd := exec.Command("stty", "echo") // Multiple stty arguments
	echoCmd.Stdin = os.Stdin
	if err := echoCmd.Run(); err != nil {
		log.Println("Failed to restore terminal:", err)
	}
}

func startConsole(quit chan struct{}) {
	err := cli.AppBlock()
	if err != nil {
		fmt.Printf(cli.ErrorString("error executing root cmd, %v\n", err))
	}
	time.Sleep(1 * time.Second)
	close(quit)
}
