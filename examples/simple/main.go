package main

import (
	"embed"
	"fmt"
	"github.com/alexj212/consolekit"
	"github.com/alexj212/consolekit/cmds"
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
