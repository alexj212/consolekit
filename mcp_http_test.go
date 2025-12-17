package consolekit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMCPHTTPRPCInitialize(t *testing.T) {
	cli, err := NewCLI("testapp", nil)
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	srv := NewMCPHTTPServer(cli, cli.AppName, "9.9.9")

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	req := httptest.NewRequest(http.MethodPost, "http://example.com/mcp", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error response: %+v", rpcResp.Error)
	}

	resultBytes, err := json.Marshal(rpcResp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if initResult.ServerInfo.Name != cli.AppName {
		t.Fatalf("expected server name %q, got %q", cli.AppName, initResult.ServerInfo.Name)
	}
	if initResult.ServerInfo.Version != "9.9.9" {
		t.Fatalf("expected server version %q, got %q", "9.9.9", initResult.ServerInfo.Version)
	}
}

func TestMCPHTTPSSEInitialize(t *testing.T) {
	cli, err := NewCLI("testapp", nil)
	if err != nil {
		t.Fatalf("NewCLI failed: %v", err)
	}

	srv := NewMCPHTTPServer(cli, cli.AppName, "1.2.3")
	sessionID := "test-session"
	msgCh := make(chan []byte, 1)
	srv.sessionsMu.Lock()
	srv.sessions[sessionID] = msgCh
	srv.sessionsMu.Unlock()
	t.Cleanup(func() {
		srv.sessionsMu.Lock()
		delete(srv.sessions, sessionID)
		srv.sessionsMu.Unlock()
		close(msgCh)
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	req := httptest.NewRequest(http.MethodPost, "http://example.com/messages?sessionId="+sessionID, strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var data []byte
	select {
	case data = <-msgCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for message")
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		t.Fatalf("unmarshal rpc response: %v", err)
	}
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error response: %+v", rpcResp.Error)
	}
}
