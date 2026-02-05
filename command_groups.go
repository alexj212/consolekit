package consolekit

import (
	"github.com/spf13/cobra"
)

// This file provides convenience bundle functions for command registration.
// Applications can use these bundles or selectively include individual command groups.

// AddAllCmds registers all available built-in commands.
// This provides the complete ConsoleKit feature set.
func AddAllCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Core & Essential
		AddCoreCmds(exec)(rootCmd)

		// State Management
		AddVariableCmds(exec)(rootCmd)
		AddAliasCmds(exec)(rootCmd)
		AddHistoryCmds(exec)(rootCmd)
		AddConfigCmds(exec)(rootCmd)

		// Scripting & Control Flow
		AddScriptingCmds(exec)(rootCmd)
		AddControlFlowCmds(exec)(rootCmd)

		// OS Integration
		AddOSExecCmds(exec)(rootCmd)
		AddJobCmds(exec)(rootCmd)
		AddScheduleCmds(exec)(rootCmd)

		// File & Data
		AddFileUtilCmds(exec)(rootCmd)
		AddDataManipulationCmds(exec)(rootCmd)

		// Output & Formatting
		AddFormatCmds(exec)(rootCmd)
		AddPipelineCmds(exec)(rootCmd)
		AddClipboardCmds(exec)(rootCmd)

		// Advanced Features
		AddTemplateCmds(exec)(rootCmd)
		AddInteractiveCmds(exec)(rootCmd)
		AddLoggingCmds(exec)(rootCmd)

		// Integrations
		AddNetworkCmds(exec)(rootCmd)
		AddTimeCmds(exec)(rootCmd)
		AddNotificationCmds(exec)(rootCmd)
		AddMCPCmds(exec)(rootCmd)

		// Utilities
		AddUtilityCmds(exec)(rootCmd)
	}
}

// AddStandardCmds registers the recommended default command set.
// This includes all commonly-used commands except advanced integrations.
func AddStandardCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Core & Essential
		AddCoreCmds(exec)(rootCmd)

		// State Management
		AddVariableCmds(exec)(rootCmd)
		AddAliasCmds(exec)(rootCmd)
		AddHistoryCmds(exec)(rootCmd)
		AddConfigCmds(exec)(rootCmd)

		// Scripting & Control Flow
		AddScriptingCmds(exec)(rootCmd)
		AddControlFlowCmds(exec)(rootCmd)

		// OS Integration
		AddOSExecCmds(exec)(rootCmd)
		AddJobCmds(exec)(rootCmd)

		// File & Data
		AddFileUtilCmds(exec)(rootCmd)
		AddDataManipulationCmds(exec)(rootCmd)

		// Output & Formatting
		AddFormatCmds(exec)(rootCmd)
		AddPipelineCmds(exec)(rootCmd)

		// Basic integrations
		AddNetworkCmds(exec)(rootCmd)
		AddTimeCmds(exec)(rootCmd)

		// Utilities
		AddUtilityCmds(exec)(rootCmd)
	}
}

// AddMinimalCmds registers only the essential commands for a basic CLI.
// This includes core commands, variables, and basic scripting support.
func AddMinimalCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		AddCoreCmds(exec)(rootCmd)
		AddVariableCmds(exec)(rootCmd)
		AddScriptingCmds(exec)(rootCmd)
		AddControlFlowCmds(exec)(rootCmd)
	}
}

// AddDeveloperCmds registers commands useful for development and automation.
// This includes standard commands plus jobs, templates, and interactive prompts.
func AddDeveloperCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Include all standard commands
		AddStandardCmds(exec)(rootCmd)

		// Add developer-specific features
		AddScheduleCmds(exec)(rootCmd)
		AddTemplateCmds(exec)(rootCmd)
		AddInteractiveCmds(exec)(rootCmd)
		AddLoggingCmds(exec)(rootCmd)
		AddClipboardCmds(exec)(rootCmd)
	}
}

// AddAutomationCmds registers commands optimized for automation and scripting.
// This excludes interactive features but includes advanced control flow and data manipulation.
func AddAutomationCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Core & State
		AddCoreCmds(exec)(rootCmd)
		AddVariableCmds(exec)(rootCmd)
		AddConfigCmds(exec)(rootCmd)

		// Scripting
		AddScriptingCmds(exec)(rootCmd)
		AddControlFlowCmds(exec)(rootCmd)
		AddTemplateCmds(exec)(rootCmd)

		// OS & Jobs
		AddOSExecCmds(exec)(rootCmd)
		AddJobCmds(exec)(rootCmd)
		AddScheduleCmds(exec)(rootCmd)

		// Data manipulation
		AddFileUtilCmds(exec)(rootCmd)
		AddDataManipulationCmds(exec)(rootCmd)
		AddFormatCmds(exec)(rootCmd)
		AddPipelineCmds(exec)(rootCmd)

		// Network & Time
		AddNetworkCmds(exec)(rootCmd)
		AddTimeCmds(exec)(rootCmd)

		// Logging for audit trails
		AddLoggingCmds(exec)(rootCmd)
	}
}

// Individual command group functions (these will be implemented in their respective files)

// AddCoreCmds registers essential core commands: exit, cls, print, date
func AddCoreCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddBaseCmds(exec) // Implemented in base.go
}

// AddVariableCmds registers variable management commands: let, unset, vars, inc, dec
func AddVariableCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddVariableCommands(exec) // Implemented in varcmds.go
}

// AddAliasCmds registers alias management commands
func AddAliasCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddAlias(exec) // Implemented in alias.go
}

// AddHistoryCmds registers history management commands
func AddHistoryCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddHistory(exec) // Implemented in history.go
}

// AddConfigCmds registers configuration management commands
func AddConfigCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddConfigCommands(exec) // Implemented in configcmds.go
}

// AddScriptingCmds registers script execution commands: run
// Note: The run command requires an embed.FS parameter, so applications must call
// AddRun(exec, scripts) directly when they have embedded scripts.
func AddScriptingCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	// No-op placeholder - run command is added via AddRun(exec, scripts embed.FS)
	return func(rootCmd *cobra.Command) {
		// Applications should call AddRun(exec, scripts) separately
	}
}

// AddControlFlowCmds registers all control flow commands: if, repeat, while, for, case, test
func AddControlFlowCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Basic control flow commands from base.go
		AddControlFlowBasicCmds(exec)(rootCmd)
		// Advanced control flow commands from controlflowcmds.go
		AddControlFlowCommands(exec)(rootCmd)
	}
}

// AddOSExecCmds registers OS command execution: osexec
func AddOSExecCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddOSExec(exec) // Implemented in exec.go
}

// AddJobCmds registers job management commands: jobs, job, killall, jobclean
func AddJobCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddJobCommands(exec) // Implemented in jobcmds.go
}

// AddScheduleCmds registers task scheduling commands: schedule
func AddScheduleCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddScheduleCommands(exec) // Implemented in schedulecmds.go
}

// AddFileUtilCmds registers file utility commands: cat, grep, env
func AddFileUtilCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddMisc(exec) // Implemented in misc.go
}

// AddDataManipulationCmds registers data manipulation commands: json, yaml, csv
func AddDataManipulationCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddDataManipulationCommands(exec) // Implemented in datamanipcmds.go
}

// AddFormatCmds registers output formatting commands: table, column, highlight, page
func AddFormatCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddFormatCommands(exec) // Implemented in formatcmds.go
}

// AddPipelineCmds registers pipeline utility commands: tee
func AddPipelineCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddPipelineCommands(exec) // Implemented in pipelinecmds.go
}

// AddClipboardCmds registers clipboard commands: clip, paste
func AddClipboardCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddClipboardCommands(exec) // Implemented in clipboardcmds.go
}

// AddTemplateCmds registers template management commands
func AddTemplateCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddTemplateCommands(exec) // Implemented in templatecmds.go
}

// AddInteractiveCmds registers interactive prompt commands
func AddInteractiveCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddPromptCommands(exec) // Implemented in promptcmds.go
}

// AddLoggingCmds registers logging and audit commands
func AddLoggingCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddLogCommands(exec) // Implemented in logcmds.go
}

// AddNetworkCmds registers network commands: http
func AddNetworkCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddNetworkCommands(exec) // Implemented in base.go
}

// AddTimeCmds registers time-related commands: sleep, wait, waitfor, watch
func AddTimeCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Time commands from base.go
		AddTimeCommands(exec)(rootCmd)
		// Watch command from watchcmds.go
		AddWatchCommand(exec)(rootCmd)
	}
}

// AddNotificationCmds registers notification commands
func AddNotificationCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddNotifyCommands(exec) // Implemented in notifycmds.go
}

// AddMCPCmds registers MCP integration commands
func AddMCPCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddMCPCommands(exec) // Implemented in mcpcmds.go
}

// AddUtilityCmds registers utility commands
func AddUtilityCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return AddUtilityCommands(exec) // Implemented in utilcmds.go
}
