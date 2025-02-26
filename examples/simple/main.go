package main

import (
	"fmt"
	"github.com/alexj212/consolekit"
	"github.com/alexj212/consolekit/cmds"
)

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
		cli.AddAlias()
		cli.AddExec()
		cli.AddHistory()
		cli.AddMiscCmds()

		cli.AddCommand(cmds.EchoCommand(cli))
		cli.AddCommand(cmds.GrepCommand(cli))
		return nil
	}
	cli := consolekit.NewCLI("simple", BuildDate, LatestCommit, Version, GitRepo, GitBranch, customizer)
	err := cli.AppBlock()
	if err != nil {
		fmt.Printf("Error, %v\n", err)
	}
}
