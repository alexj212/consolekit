package consolekit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSocketHandler_UnixSocket(t *testing.T) {
	// Create a minimal executor
	executor, err := NewCommandExecutor("socket-test", func(exec *CommandExecutor) error {
		exec.AddCommands(AddCoreCmds(exec))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Use a temp socket path
	sockPath := filepath.Join(os.TempDir(), "consolekit-test.sock")
	defer os.Remove(sockPath)

	handler := NewSocketHandler(executor, "unix", sockPath)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start()
	}()

	// Wait for server to start
	var conn net.Conn
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send a command
	req := SocketRequest{ID: "test-1", Command: "print hello"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	_, err = conn.Write(data)
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}

	var resp SocketResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got error: %s", resp.Error)
	}
	if resp.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", resp.ID)
	}
	if resp.Output != "hello\n" {
		t.Errorf("Expected output 'hello\\n', got '%s'", resp.Output)
	}

	// Stop server
	handler.Stop()
}

func TestSocketHandler_TCPWithAuth(t *testing.T) {
	executor, err := NewCommandExecutor("socket-test-tcp", func(exec *CommandExecutor) error {
		exec.AddCommands(AddCoreCmds(exec))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	handler := NewSocketHandler(executor, "tcp", "127.0.0.1:0")
	handler.SetAuthToken("test-secret")

	// Use port 0 to let OS assign a free port
	// We need to start and capture the actual address
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Start()
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Get the actual address from the listener
	handler.mu.Lock()
	addr := handler.listener.Addr().String()
	handler.mu.Unlock()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send command without token - should fail
	req := SocketRequest{ID: "no-auth", Command: "print test"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	conn.Write(data)

	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	var resp SocketResponse
	json.Unmarshal(scanner.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected auth failure, got success")
	}
	if resp.Error != "authentication required" {
		t.Errorf("Expected 'authentication required', got '%s'", resp.Error)
	}

	// Send command with correct token - should succeed
	req = SocketRequest{ID: "with-auth", Command: "print hello", Token: "test-secret"}
	data, _ = json.Marshal(req)
	data = append(data, '\n')
	conn.Write(data)

	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	json.Unmarshal(scanner.Bytes(), &resp)

	if !resp.Success {
		t.Errorf("Expected success, got error: %s", resp.Error)
	}
	if resp.ID != "with-auth" {
		t.Errorf("Expected ID 'with-auth', got '%s'", resp.ID)
	}

	handler.Stop()
}

func TestSocketHandler_MultipleCommands(t *testing.T) {
	executor, err := NewCommandExecutor("socket-test-multi", func(exec *CommandExecutor) error {
		exec.AddCommands(AddCoreCmds(exec))
		exec.AddCommands(AddVariableCmds(exec))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	sockPath := filepath.Join(os.TempDir(), "consolekit-test-multi.sock")
	defer os.Remove(sockPath)

	handler := NewSocketHandler(executor, "unix", sockPath)

	go handler.Start()

	var conn net.Conn
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send multiple commands on same connection
	commands := []string{"print first", "print second", "print third"}
	for i, cmd := range commands {
		req := SocketRequest{ID: fmt.Sprintf("cmd-%d", i), Command: cmd}
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		conn.Write(data)

		if !scanner.Scan() {
			t.Fatalf("Failed to read response for command %d: %v", i, scanner.Err())
		}

		var resp SocketResponse
		json.Unmarshal(scanner.Bytes(), &resp)
		if !resp.Success {
			t.Errorf("Command %d failed: %s", i, resp.Error)
		}
	}

	handler.Stop()
}
