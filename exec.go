package consolekit

//
//func executeLinea(c *Console, menu *Menu, line string) {
//	if line == "" {
//		return
//	}
//	outFile, commands, err := parser.ParseCommands(line)
//	if err != nil {
//		fmt.Printf("executeLine parsing line `%s` error: %s\n", line, err.Error())
//		return
//	}
//	if len(commands) == 0 {
//		return
//	}
//
//	rootCmd := menu.Root()
//
//	var outputBuffer bytes.Buffer
//	for _, command := range commands {
//		// Run user-provided pre-run line hooks,
//		// which may modify the input line args.
//		command.Args, err = c.runLineHooks(command.Args)
//		if err != nil {
//			fmt.Printf("executeLine runLineHooks error: %s\n", err.Error())
//			continue
//		}
//		menu.Command = rootCmd
//
//		// Run all pre-run hooks and the command itself
//		// Don't check the error: if its a cobra error,
//		// the library user is responsible for setting
//		// the cobra behavior.
//		// If it's an interrupt, we take care of it.
//		output, err := c.executeSingleCommand(menu, rootCmd, command)
//		if err != nil {
//			fmt.Printf("executeLine %v\n", err)
//			break
//		}
//		outputBuffer.WriteString(output)
//	}
//
//	// Print the output of the last command
//	output := outputBuffer.String()
//	lines := strings.Split(output, "\n")
//	for _, line := range lines {
//		fmt.Printf("%s\n", string(line))
//	}
//
//	// Handle output redirection if specified
//	if outFile != "" {
//		if err := os.WriteFile(outFile, outputBuffer.Bytes(), 0644); err != nil {
//			fmt.Printf("executeLine failed to write to file `%v` error: %v", outFile, err)
//		}
//	}
//
//}
