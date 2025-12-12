package consolekit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditLog represents a single command execution log entry
type AuditLog struct {
	Timestamp time.Time     `json:"timestamp"`
	User      string        `json:"user"`
	Command   string        `json:"command"`
	Output    string        `json:"output,omitempty"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

// LogManager handles command logging and audit trail
type LogManager struct {
	enabled      bool
	logFile      string
	logSuccess   bool
	logFailures  bool
	maxSizeMB    int64
	retentionDays int
	logs         []AuditLog
	mu           sync.RWMutex
}

// NewLogManager creates a new log manager
func NewLogManager(logFile string) *LogManager {
	return &LogManager{
		enabled:      false,
		logFile:      logFile,
		logSuccess:   true,
		logFailures:  true,
		maxSizeMB:    100,
		retentionDays: 90,
		logs:         make([]AuditLog, 0),
	}
}

// Enable enables command logging
func (lm *LogManager) Enable() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.enabled = true
}

// Disable disables command logging
func (lm *LogManager) Disable() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.enabled = false
}

// IsEnabled returns whether logging is enabled
func (lm *LogManager) IsEnabled() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.enabled
}

// SetLogFile sets the log file path
func (lm *LogManager) SetLogFile(path string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.logFile = path
}

// GetLogFile returns the current log file path
func (lm *LogManager) GetLogFile() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.logFile
}

// SetMaxSize sets the maximum log file size in MB
func (lm *LogManager) SetMaxSize(mb int64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.maxSizeMB = mb
}

// SetRetention sets the log retention period in days
func (lm *LogManager) SetRetention(days int) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.retentionDays = days
}

// SetLogSuccess sets whether to log successful commands
func (lm *LogManager) SetLogSuccess(enable bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.logSuccess = enable
}

// SetLogFailures sets whether to log failed commands
func (lm *LogManager) SetLogFailures(enable bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.logFailures = enable
}

// Log records a command execution
func (lm *LogManager) Log(entry AuditLog) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if !lm.enabled {
		return nil
	}

	// Check if we should log based on success/failure
	if entry.Success && !lm.logSuccess {
		return nil
	}
	if !entry.Success && !lm.logFailures {
		return nil
	}

	// Add to in-memory logs
	lm.logs = append(lm.logs, entry)

	// Append to log file if configured
	if lm.logFile != "" {
		return lm.appendToFile(entry)
	}

	return nil
}

// appendToFile appends a log entry to the log file
func (lm *LogManager) appendToFile(entry AuditLog) error {
	// Ensure directory exists
	dir := filepath.Dir(lm.logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Check file size and rotate if needed
	if err := lm.rotateIfNeeded(); err != nil {
		return fmt.Errorf("failed to rotate log: %w", err)
	}

	// Open file for append
	f, err := os.OpenFile(lm.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Write JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// rotateIfNeeded rotates the log file if it exceeds max size
func (lm *LogManager) rotateIfNeeded() error {
	info, err := os.Stat(lm.logFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Check if file exceeds max size
	maxBytes := lm.maxSizeMB * 1024 * 1024
	if info.Size() < maxBytes {
		return nil
	}

	// Rotate: rename current to .old
	oldFile := lm.logFile + ".old"
	if err := os.Rename(lm.logFile, oldFile); err != nil {
		return err
	}

	return nil
}

// GetLogs returns all in-memory logs
func (lm *LogManager) GetLogs() []AuditLog {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]AuditLog, len(lm.logs))
	copy(result, lm.logs)
	return result
}

// GetRecentLogs returns the last N logs
func (lm *LogManager) GetRecentLogs(n int) []AuditLog {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if n <= 0 || n >= len(lm.logs) {
		result := make([]AuditLog, len(lm.logs))
		copy(result, lm.logs)
		return result
	}

	start := len(lm.logs) - n
	result := make([]AuditLog, n)
	copy(result, lm.logs[start:])
	return result
}

// GetFailedLogs returns only failed command logs
func (lm *LogManager) GetFailedLogs() []AuditLog {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]AuditLog, 0)
	for _, log := range lm.logs {
		if !log.Success {
			result = append(result, log)
		}
	}
	return result
}

// SearchLogs searches logs by command text
func (lm *LogManager) SearchLogs(query string) []AuditLog {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]AuditLog, 0)
	for _, log := range lm.logs {
		if strings.Contains(log.Command, query) {
			result = append(result, log)
		}
	}
	return result
}

// GetLogsSince returns logs since a specific time
func (lm *LogManager) GetLogsSince(since time.Time) []AuditLog {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]AuditLog, 0)
	for _, log := range lm.logs {
		if log.Timestamp.After(since) || log.Timestamp.Equal(since) {
			result = append(result, log)
		}
	}
	return result
}

// Clear clears all in-memory logs
func (lm *LogManager) Clear() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.logs = make([]AuditLog, 0)
}

// LoadFromFile loads logs from the log file
func (lm *LogManager) LoadFromFile() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.logFile == "" {
		return nil
	}

	data, err := os.ReadFile(lm.logFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	// Parse JSON lines
	logs := make([]AuditLog, 0)
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry AuditLog
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		logs = append(logs, entry)
	}

	lm.logs = logs
	return nil
}

// ExportJSON exports logs to JSON format
func (lm *LogManager) ExportJSON() ([]byte, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return json.MarshalIndent(lm.logs, "", "  ")
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}

	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
