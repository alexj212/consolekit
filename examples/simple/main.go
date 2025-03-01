package main

import (
	"embed"
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

//go:embed *.run
var Data embed.FS

func main() {
	customizer := func(cli *consolekit.CLI) error {
		cmds.AddAlias(cli)
		cmds.AddExec(cli)
		cmds.AddHistory(cli)
		cmds.AddMisc(cli)
		cmds.AddBaseCmds(cli)
		cmds.AddRun(cli, Data)
		return nil
	}
	cli, err := consolekit.NewCLI("simple", BuildDate, LatestCommit, Version, GitRepo, GitBranch, customizer)
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
