package main

import (
	"embed"
	"fmt"

	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
)

//go:embed *.run
var Data embed.FS

var (
	// BuildDate date string of when build was performed filled in by -X compile flag
	BuildDate string

	// LatestCommit date string of when build was performed filled in by -X compile flag
	LatestCommit string

	// Version string of build filled in by -X compile flag
	Version string

	// GitRepo string of the git repo url when build was performed filled in by -X compile flag
	GitRepo string

	// GitBranch string of branch in the git repo filled in by -X compile flag
	GitBranch string
)

func main() {
	customizer := func(cli *consolekit.CLI) error {

		cli.AddAll()
		cli.AddCommands(consolekit.AddRun(cli, Data))

		var verCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("BuildDate    : %s\n", BuildDate)
			cmd.Printf("LatestCommit : %s\n", LatestCommit)
			cmd.Printf("Version      : %s\n", Version)
			cmd.Printf("GitRepo      : %s\n", GitRepo)
			cmd.Printf("GitBranch    : %s\n", GitBranch)
		}

		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run:     verCmdFunc,
		}
		cli.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}
	cli, err := consolekit.NewCLI("simple", customizer)
	if err != nil {
		fmt.Printf("consolekit.NewCLI error, %v\n", err)
		return
	}
	cli.Scripts = Data
	err = cli.AppBlock()
	if err != nil {
		fmt.Printf("Error, %v\n", err)
	}
}
