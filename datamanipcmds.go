package consolekit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AddDataManipulationCommands adds JSON, CSV, and YAML manipulation commands
func AddDataManipulationCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// JSON command
		var jsonCmd = &cobra.Command{
			Use:   "json",
			Short: "JSON manipulation commands",
			Long:  "Parse, query, format, and manipulate JSON data",
		}

		// json parse
		var jsonPretty bool
		var jsonParseCmd = &cobra.Command{
			Use:   "parse [file]",
			Short: "Parse and format JSON",
			Long:  "Parse JSON from file or stdin and optionally pretty-print",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error

				if len(args) > 0 {
					data, err = os.ReadFile(args[0])
				} else {
					// Read from stdin
					data, err = os.ReadFile("/dev/stdin")
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := json.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid JSON: %v", err))
					return
				}

				var output []byte
				if jsonPretty {
					output, err = json.MarshalIndent(parsed, "", "  ")
				} else {
					output, err = json.Marshal(parsed)
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to format JSON: %v", err))
					return
				}

				cmd.Println(string(output))
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		jsonParseCmd.Flags().BoolVar(&jsonPretty, "pretty", true, "Pretty-print JSON output")

		// json get - extract value from JSON using path notation
		var jsonGetCmd = &cobra.Command{
			Use:   "get [file] [path]",
			Short: "Get value from JSON",
			Long:  "Extract a value from JSON using dot notation (e.g., 'users.0.name')",
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error
				var path string

				if len(args) == 2 {
					data, err = os.ReadFile(args[0])
					path = args[1]
				} else {
					// Read from stdin
					data, err = os.ReadFile("/dev/stdin")
					path = args[0]
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := json.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid JSON: %v", err))
					return
				}

				value := getJSONPath(parsed, path)
				if value == nil {
					cmd.PrintErrln(fmt.Sprintf("Path not found: %s", path))
					return
				}

				output, err := json.MarshalIndent(value, "", "  ")
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to format output: %v", err))
					return
				}

				cmd.Println(string(output))
			},
		}

		// json validate
		var jsonValidateCmd = &cobra.Command{
			Use:   "validate [file]",
			Short: "Validate JSON syntax",
			Long:  "Check if JSON is valid",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error

				if len(args) > 0 {
					data, err = os.ReadFile(args[0])
				} else {
					data, err = os.ReadFile("/dev/stdin")
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := json.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid JSON: %v", err))
					return
				}

				cmd.Println(fmt.Sprintf("Valid JSON"))
			},
		}

		jsonCmd.AddCommand(jsonParseCmd)
		jsonCmd.AddCommand(jsonGetCmd)
		jsonCmd.AddCommand(jsonValidateCmd)

		// YAML command
		var yamlCmd = &cobra.Command{
			Use:   "yaml",
			Short: "YAML manipulation commands",
			Long:  "Parse, format, and convert YAML data",
		}

		// yaml parse
		var yamlParseCmd = &cobra.Command{
			Use:   "parse [file]",
			Short: "Parse and format YAML",
			Long:  "Parse YAML from file or stdin",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error

				if len(args) > 0 {
					data, err = os.ReadFile(args[0])
				} else {
					data, err = os.ReadFile("/dev/stdin")
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid YAML: %v", err))
					return
				}

				output, err := yaml.Marshal(parsed)
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to format YAML: %v", err))
					return
				}

				cmd.Print(string(output))
			},
		}

		// yaml to-json
		var yamlToJSONCmd = &cobra.Command{
			Use:   "to-json [file]",
			Short: "Convert YAML to JSON",
			Long:  "Convert YAML input to JSON format",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error

				if len(args) > 0 {
					data, err = os.ReadFile(args[0])
				} else {
					data, err = os.ReadFile("/dev/stdin")
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid YAML: %v", err))
					return
				}

				output, err := json.MarshalIndent(parsed, "", "  ")
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to convert to JSON: %v", err))
					return
				}

				cmd.Println(string(output))
			},
		}

		// yaml from-json
		var yamlFromJSONCmd = &cobra.Command{
			Use:   "from-json [file]",
			Short: "Convert JSON to YAML",
			Long:  "Convert JSON input to YAML format",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var data []byte
				var err error

				if len(args) > 0 {
					data, err = os.ReadFile(args[0])
				} else {
					data, err = os.ReadFile("/dev/stdin")
				}

				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to read input: %v", err))
					return
				}

				var parsed interface{}
				if err := json.Unmarshal(data, &parsed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Invalid JSON: %v", err))
					return
				}

				output, err := yaml.Marshal(parsed)
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to convert to YAML: %v", err))
					return
				}

				cmd.Print(string(output))
			},
		}

		yamlCmd.AddCommand(yamlParseCmd)
		yamlCmd.AddCommand(yamlToJSONCmd)
		yamlCmd.AddCommand(yamlFromJSONCmd)

		// CSV command
		var csvCmd = &cobra.Command{
			Use:   "csv",
			Short: "CSV manipulation commands",
			Long:  "Parse and manipulate CSV data",
		}

		// csv parse
		var csvHeader bool
		var csvParseCmd = &cobra.Command{
			Use:   "parse [file]",
			Short: "Parse CSV file",
			Long:  "Parse CSV from file or stdin and display as table",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var file *os.File
				var err error

				if len(args) > 0 {
					file, err = os.Open(args[0])
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Failed to open file: %v", err))
						return
					}
					defer file.Close()
				} else {
					file = os.Stdin
				}

				reader := csv.NewReader(file)
				records, err := reader.ReadAll()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to parse CSV: %v", err))
					return
				}

				if len(records) == 0 {
					cmd.Println(fmt.Sprintf("Empty CSV"))
					return
				}

				// Print as table
				for i, record := range records {
					if csvHeader && i == 0 {
						cmd.Println(strings.Join(record, " | "))
						cmd.Println(strings.Repeat("-", len(strings.Join(record, " | "))))
					} else {
						cmd.Println(strings.Join(record, " | "))
					}
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		csvParseCmd.Flags().BoolVar(&csvHeader, "header", true, "First row is header")

		// csv to-json
		var csvToJSONCmd = &cobra.Command{
			Use:   "to-json [file]",
			Short: "Convert CSV to JSON",
			Long:  "Convert CSV input to JSON format (assumes first row is header)",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				var file *os.File
				var err error

				if len(args) > 0 {
					file, err = os.Open(args[0])
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Failed to open file: %v", err))
						return
					}
					defer file.Close()
				} else {
					file = os.Stdin
				}

				reader := csv.NewReader(file)
				records, err := reader.ReadAll()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to parse CSV: %v", err))
					return
				}

				if len(records) < 2 {
					cmd.PrintErrln(fmt.Sprintf("CSV must have at least a header and one data row"))
					return
				}

				headers := records[0]
				var result []map[string]string

				for i := 1; i < len(records); i++ {
					row := make(map[string]string)
					for j, value := range records[i] {
						if j < len(headers) {
							row[headers[j]] = value
						}
					}
					result = append(result, row)
				}

				output, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to convert to JSON: %v", err))
					return
				}

				cmd.Println(string(output))
			},
		}

		csvCmd.AddCommand(csvParseCmd)
		csvCmd.AddCommand(csvToJSONCmd)

		rootCmd.AddCommand(jsonCmd)
		rootCmd.AddCommand(yamlCmd)
		rootCmd.AddCommand(csvCmd)
	}
}

// getJSONPath extracts a value from parsed JSON using dot notation
func getJSONPath(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			// Handle array index
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return nil
			}
			if idx < 0 || idx >= len(v) {
				return nil
			}
			current = v[idx]
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}
