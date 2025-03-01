# ConsoleKit

**ConsoleKit** is a modular CLI (Command Line Interface) library for building powerful console applications in Go. It supports advanced features like REPL (Read-Eval-Print Loop), command chaining (`;`), piping (`|`), file redirection (`>`), and modular command registration.

## Features
- 🖥️ **REPL Interface:** Provides an interactive shell-like environment.
- 🔗 **Command Chaining:** Run multiple commands sequentially using `;`.
- 🚦 **Piping:** Pipe the output of one command to another using `|`.
- 📁 **File Redirection:** Redirect output to a file using `>`, while also displaying in stdout.
- 📦 **Modular Commands:** Easily add and integrate new commands.
- 💬 **Comment Handling:** Supports `#` for comments in scripts and commands.

## Installation
```sh
go get github.com/alexj212/consolekit
```

## Example Usage

### **1. Create a CLI Application**

**main.go:**
```go
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

//go:embed *
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
		fmt.Printf("Error, %v\n", err)
		return
	}
	cli.Scripts = Data
	err = cli.AppBlock()
	if err != nil {
		fmt.Printf("Error, %v\n", err)
	}
}


```

### **2. Build and Run**
```sh
go run main.go
```

### **3. Example Commands in REPL**
```sh
myapp> echo "Hello World"
Hello World

myapp> echo "Write to file" > output.txt
Write to file

myapp> echo "hello\nworld" | grep hello
hello

myapp> # This is a comment
myapp> echo "Command chaining" ; echo "With comments"
Command chaining
With comments
```

## Contributing
Contributions are welcome! Feel free to submit issues or pull requests to help improve **ConsoleKit**.

## License
**ConsoleKit** is licensed under the [MIT License](LICENSE).
