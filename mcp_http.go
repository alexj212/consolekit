package consolekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type MCPHTTPServer struct {
	mcp *MCPServer

	sessionsMu sync.RWMutex
	sessions   map[string]chan []byte
}

func NewMCPHTTPServer(cli *CLI, appName, appVersion string) *MCPHTTPServer {
	return &MCPHTTPServer{
		mcp:      NewMCPServer(cli, appName, appVersion),
		sessions: make(map[string]chan []byte),
	}
}

func (s *MCPHTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/messages", s.handleMessages)

	// Non-SSE fallback: basic request/response JSON-RPC over HTTP POST.
	mux.HandleFunc("/mcp", s.handleRPC)
	return mux
}

func (s *MCPHTTPServer) ListenAndServe(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- err
			return
		}
		errCh <- httpServer.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *MCPHTTPServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *MCPHTTPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessionID, err := newSessionID()
	if err != nil {
		http.Error(w, "unable to create session", http.StatusInternalServerError)
		return
	}

	msgCh := make(chan []byte, 64)
	s.sessionsMu.Lock()
	s.sessions[sessionID] = msgCh
	s.sessionsMu.Unlock()
	defer func() {
		s.sessionsMu.Lock()
		delete(s.sessions, sessionID)
		s.sessionsMu.Unlock()
		close(msgCh)
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	postURL := fmt.Sprintf("%s/messages?sessionId=%s", baseURL(r), sessionID)
	writeSSE(w, "endpoint", []byte(postURL))
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			writeSSE(w, "message", msg)
			flusher.Flush()
		}
	}
}

func (s *MCPHTTPServer) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}

	s.sessionsMu.RLock()
	msgCh, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()
	if !ok {
		http.Error(w, "unknown sessionId", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "unable to read body", http.StatusBadRequest)
		return
	}

	responses, err := s.processRPCPayload(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, resp := range responses {
		select {
		case msgCh <- resp:
		case <-r.Context().Done():
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *MCPHTTPServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "unable to read body", http.StatusBadRequest)
		return
	}

	responses, err := s.processRPCPayload(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(responses) == 1 {
		_, _ = w.Write(responses[0])
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mustJSONMarshal(responses))
}

func (s *MCPHTTPServer) processRPCPayload(ctx context.Context, body []byte) ([][]byte, error) {
	body = bytesTrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty body")
	}

	// Batch support: array of requests.
	if body[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, fmt.Errorf("invalid batch JSON: %w", err)
		}
		var responses [][]byte
		for _, item := range batch {
			resp := s.mcp.ProcessBytes(ctx, item)
			if resp == nil {
				continue
			}
			b, err := json.Marshal(resp)
			if err != nil {
				return nil, fmt.Errorf("unable to marshal response: %w", err)
			}
			responses = append(responses, b)
		}
		return responses, nil
	}

	resp := s.mcp.ProcessBytes(ctx, body)
	if resp == nil {
		return nil, nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal response: %w", err)
	}
	return [][]byte{b}, nil
}

func writeSSE(w http.ResponseWriter, event string, data []byte) {
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func newSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

func mustJSONMarshal(items [][]byte) []byte {
	// items are already marshaled JSON objects; wrap them into a JSON array of objects.
	var raw []json.RawMessage
	for _, item := range items {
		raw = append(raw, json.RawMessage(item))
	}
	out, _ := json.Marshal(raw)
	return out
}
