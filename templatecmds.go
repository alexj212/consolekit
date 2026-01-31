package consolekit

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// AddTemplateCommands adds template management commands to the CLI
func AddTemplateCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var templateCmd = &cobra.Command{
			Use:   "template",
			Short: "Manage and execute script templates",
			Long:  "Create, list, show, execute, and manage script templates with variable substitution",
		}

		// template list
		var listCmd = &cobra.Command{
			Use:   "list",
			Short: "List available templates",
			Long:  "List all available templates from embedded FS and templates directory",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				templates, err := exec.TemplateManager.ListTemplates()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to list templates: %v", err))
					return
				}

				if len(templates) == 0 {
					cmd.Println(fmt.Sprintf("No templates found"))
					return
				}

				cmd.Println(fmt.Sprintf("Available templates:"))
				for _, tmpl := range templates {
					cmd.Printf("  - %s\n", tmpl)
				}
			},
		}

		// template show
		var showCmd = &cobra.Command{
			Use:   "show [name]",
			Short: "Show template content",
			Long:  "Display the raw content of a template",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				name := args[0]
				content, err := exec.TemplateManager.GetTemplateContent(name)
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to get template: %v", err))
					return
				}

				cmd.Println(content)
			},
		}

		// template exec
		var execCmd = &cobra.Command{
			Use:   "exec [name] [key=value...]",
			Short: "Execute a template",
			Long:  "Execute a template with variable substitution and run the resulting script",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				name := args[0]

				// Parse variables
				vars, err := ParseVariables(args[1:])
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to parse variables: %v", err))
					return
				}

				// Execute template
				script, err := exec.TemplateManager.ExecuteTemplate(name, vars)
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to execute template: %v", err))
					return
				}

				// Execute the generated script
				lines := strings.Split(script, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					output, err := exec.Execute(line, nil)
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Error executing line '%s': %v", line, err))
						return
					}

					if output != "" {
						cmd.Print(output)
					}
				}
			},
		}

		// template render
		var renderCmd = &cobra.Command{
			Use:   "render [name] [key=value...]",
			Short: "Render a template without executing",
			Long:  "Render a template with variable substitution and display the result",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				name := args[0]

				// Parse variables
				vars, err := ParseVariables(args[1:])
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to parse variables: %v", err))
					return
				}

				// Execute template
				script, err := exec.TemplateManager.ExecuteTemplate(name, vars)
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to execute template: %v", err))
					return
				}

				cmd.Println(script)
			},
		}

		// template create
		var createCmd = &cobra.Command{
			Use:   "create [name]",
			Short: "Create a new template",
			Long:  "Create a new template interactively or from stdin",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				name := args[0]

				cmd.Println(fmt.Sprintf("Enter template content (end with Ctrl+D on Unix or Ctrl+Z on Windows):"))

				// Read from stdin
				var content strings.Builder
				var line string
				for {
					line = exec.Prompt("")
					if line == "" {
						break
					}
					content.WriteString(line)
					content.WriteString("\n")
				}

				if err := exec.TemplateManager.SaveTemplate(name, content.String()); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to save template: %v", err))
					return
				}

				cmd.Println(fmt.Sprintf("Template '%s' created successfully", name))
			},
		}

		// template delete
		var deleteCmd = &cobra.Command{
			Use:   "delete [name]",
			Short: "Delete a template",
			Long:  "Delete a template from the file system",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				name := args[0]

				if !exec.Confirm(fmt.Sprintf("Delete template '%s'?", name)) {
					cmd.Println(fmt.Sprintf("Cancelled"))
					return
				}

				if err := exec.TemplateManager.DeleteTemplate(name); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to delete template: %v", err))
					return
				}

				cmd.Println(fmt.Sprintf("Template '%s' deleted successfully", name))
			},
		}

		// template clear-cache
		var clearCacheCmd = &cobra.Command{
			Use:   "clear-cache",
			Short: "Clear template cache",
			Long:  "Clear the in-memory template cache to force reload from disk",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.TemplateManager == nil {
					cmd.PrintErrln(fmt.Sprintf("Template manager not initialized"))
					return
				}

				exec.TemplateManager.ClearCache()
				cmd.Println(fmt.Sprintf("Template cache cleared"))
			},
		}

		// Add subcommands
		templateCmd.AddCommand(listCmd)
		templateCmd.AddCommand(showCmd)
		templateCmd.AddCommand(execCmd)
		templateCmd.AddCommand(renderCmd)
		templateCmd.AddCommand(createCmd)
		templateCmd.AddCommand(deleteCmd)
		templateCmd.AddCommand(clearCacheCmd)

		rootCmd.AddCommand(templateCmd)
	}
}
