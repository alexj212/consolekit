package consolekit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// MCP Protocol Types (JSON-RPC 2.0)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Specific Types

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// MCPServer is the MCP protocol server
type MCPServer struct {
	cli        *CommandExecutor
	appName    string
	appVersion string
	reader     *bufio.Reader
	writer     io.Writer
}

// NewMCPServer creates a new MCP server
func NewMCPServer(cli *CommandExecutor, appName, appVersion string) *MCPServer {
	return &MCPServer{
		cli:        cli,
		appName:    appName,
		appVersion: appVersion,
		reader:     bufio.NewReader(os.Stdin),
		writer:     os.Stdout,
	}
}

// Run starts the MCP server loop
func (s *MCPServer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read next JSON-RPC request
			line, err := s.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error reading request: %w", err)
			}

			resp := s.ProcessBytes(ctx, line)
			if resp != nil {
				if err := s.writeResponse(resp); err != nil {
					return err
				}
			}
		}
	}
}

// ProcessBytes processes a single JSON-RPC message and returns a response (or nil for notifications).
func (s *MCPServer) ProcessBytes(ctx context.Context, data []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return s.newErrorResponse(nil, -32700, "Parse error", err.Error())
	}
	return s.Process(ctx, &req)
}

// Process processes a single request and returns a response (or nil for notifications).
func (s *MCPServer) Process(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(req)
	default:
		return s.newErrorResponse(req.ID, -32601, "Method not found", fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	if req.ID == nil {
		return nil
	}
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    s.appName,
			Version: s.appVersion,
		},
	}

	return s.newResultResponse(req.ID, result)
}

// handleToolsList returns the list of available tools (CLI commands)
func (s *MCPServer) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	if req.ID == nil {
		return nil
	}
	tools := []Tool{}

	// Get the root command
	rootCmd := s.cli.RootCmd()

	// Recursively collect all commands
	s.collectCommands(rootCmd, "", &tools)

	result := ToolsListResult{
		Tools: tools,
	}

	return s.newResultResponse(req.ID, result)
}

// collectCommands recursively collects commands and converts them to MCP tools
func (s *MCPServer) collectCommands(cmd *cobra.Command, prefix string, tools *[]Tool) {
	// Skip hidden commands
	if cmd.Hidden {
		return
	}

	// Handle root command (empty name) - just recurse into subcommands
	if cmd.Name() == "" {
		for _, subCmd := range cmd.Commands() {
			s.collectCommands(subCmd, prefix, tools)
		}
		return
	}

	// Build the full command name
	fullName := cmd.Name()
	if prefix != "" {
		fullName = prefix + " " + cmd.Name()
	}

	// If this is a parent-only command with no Run function, recurse into subcommands
	if cmd.Run == nil && cmd.RunE == nil && len(cmd.Commands()) > 0 {
		// Recurse into subcommands with updated prefix
		for _, subCmd := range cmd.Commands() {
			s.collectCommands(subCmd, fullName, tools)
		}
		return
	}

	// If command has no Run function and no subcommands, skip it
	if cmd.Run == nil && cmd.RunE == nil {
		return
	}

	// Create input schema from flags
	properties := make(map[string]interface{})
	required := []string{}

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagSchema := map[string]interface{}{
			"type":        "string",
			"description": flag.Usage,
		}

		if flag.DefValue != "" {
			flagSchema["default"] = flag.DefValue
		}

		properties[flag.Name] = flagSchema

		// Mark as required if no default and not a boolean flag
		if flag.DefValue == "" && flag.Value.Type() != "bool" {
			required = append(required, flag.Name)
		}
	})

	// Add _args property for positional arguments
	if cmd.Args != nil || cmd.Use != "" {
		properties["_args"] = map[string]interface{}{
			"type":        "string",
			"description": "Positional arguments for the command",
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}

	// Create the tool
	tool := Tool{
		Name:        fullName,
		Description: cmd.Short,
		InputSchema: inputSchema,
	}

	*tools = append(*tools, tool)

	// Recurse into subcommands
	for _, subCmd := range cmd.Commands() {
		s.collectCommands(subCmd, fullName, tools)
	}
}

// handleToolsCall executes a CLI command
func (s *MCPServer) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	if req.ID == nil {
		return nil
	}
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.newErrorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	// Build command line from tool name and arguments
	cmdLine := params.Name

	// Add flag arguments
	for key, value := range params.Arguments {
		if key == "_args" {
			// Positional arguments
			if strVal, ok := value.(string); ok && strVal != "" {
				cmdLine += " " + strVal
			}
		} else {
			// Flag arguments
			cmdLine += fmt.Sprintf(" --%s=%v", key, value)
		}
	}

	// Execute the command
	output, err := s.cli.ExecuteWithContext(ctx, cmdLine, nil)

	result := CallToolResult{
		Content: []ContentItem{
			{
				Type: "text",
				Text: output,
			},
		},
		IsError: err != nil,
	}

	if err != nil {
		result.Content = append(result.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("\n\nError: %v", err),
		})
	}

	return s.newResultResponse(req.ID, result)
}

// handleResourcesList returns available resources (templates, scripts, etc.)
func (s *MCPServer) handleResourcesList(req *JSONRPCRequest) *JSONRPCResponse {
	if req.ID == nil {
		return nil
	}
	resources := []Resource{}

	// Add templates as resources
	if s.cli.TemplateManager != nil {
		templates, _ := s.cli.TemplateManager.ListTemplates()
		for _, tmpl := range templates {
			resources = append(resources, Resource{
				URI:         "template://" + tmpl,
				Name:        tmpl,
				Description: "ConsoleKit template",
				MimeType:    "text/plain",
			})
		}
	}

	result := ResourcesListResult{
		Resources: resources,
	}

	return s.newResultResponse(req.ID, result)
}

func (s *MCPServer) newResultResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *MCPServer) newErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func (s *MCPServer) writeResponse(resp *JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshaling response: %w", err)
	}
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	if err != nil {
		return fmt.Errorf("error writing response: %w", err)
	}
	return nil
}
