package consolekit

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestJobManager_Add(t *testing.T) {
	jm := NewJobManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	jobID := jm.Add("sleep 1", ctx, cancel, cmd)

	if jobID != 1 {
		t.Errorf("Expected job ID 1, got %d", jobID)
	}

	job, ok := jm.Get(jobID)
	if !ok {
		t.Fatal("Job not found")
	}

	job.mu.RLock()
	status := job.Status
	command := job.Command
	job.mu.RUnlock()

	if status != JobRunning {
		t.Errorf("Expected status %s, got %s", JobRunning, status)
	}

	if command != "sleep 1" {
		t.Errorf("Expected command 'sleep 1', got %s", command)
	}
}

func TestJobManager_List(t *testing.T) {
	jm := NewJobManager()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	cmd1 := exec.CommandContext(ctx1, "sleep", "1")
	if err := cmd1.Start(); err != nil {
		t.Fatalf("Failed to start command 1: %v", err)
	}
	jm.Add("sleep 1", ctx1, cancel1, cmd1)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	cmd2 := exec.CommandContext(ctx2, "sleep", "2")
	if err := cmd2.Start(); err != nil {
		t.Fatalf("Failed to start command 2: %v", err)
	}
	jm.Add("sleep 2", ctx2, cancel2, cmd2)

	jobs := jm.List()

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
}

func TestJobManager_Kill(t *testing.T) {
	jm := NewJobManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	jobID := jm.Add("sleep 10", ctx, cancel, cmd)

	// Give the job a moment to start
	time.Sleep(100 * time.Millisecond)

	err := jm.Kill(jobID)
	if err != nil {
		t.Fatalf("Failed to kill job: %v", err)
	}

	// Wait a moment for status to update
	time.Sleep(200 * time.Millisecond)

	job, ok := jm.Get(jobID)
	if !ok {
		t.Fatal("Job not found")
	}

	job.mu.RLock()
	status := job.Status
	job.mu.RUnlock()

	if status != JobKilled {
		t.Errorf("Expected status %s, got %s", JobKilled, status)
	}
}

func TestJobManager_Wait(t *testing.T) {
	jm := NewJobManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	jobID := jm.Add("sleep 0.1", ctx, cancel, cmd)

	err := jm.Wait(jobID)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	job, ok := jm.Get(jobID)
	if !ok {
		t.Fatal("Job not found")
	}

	job.mu.RLock()
	status := job.Status
	job.mu.RUnlock()

	if status != JobCompleted {
		t.Errorf("Expected status %s, got %s", JobCompleted, status)
	}
}

func TestJobManager_Clean(t *testing.T) {
	jm := NewJobManager()

	// Add a job that will complete quickly
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "echo", "test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}
	jm.Add("echo test", ctx, cancel, cmd)

	// Wait for job to complete
	time.Sleep(200 * time.Millisecond)

	removed := jm.Clean()

	if removed != 1 {
		t.Errorf("Expected 1 job removed, got %d", removed)
	}

	jobs := jm.List()
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after clean, got %d", len(jobs))
	}
}

func TestJob_Duration(t *testing.T) {
	jm := NewJobManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "0.2")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	jobID := jm.Add("sleep 0.2", ctx, cancel, cmd)

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	job, ok := jm.Get(jobID)
	if !ok {
		t.Fatal("Job not found")
	}

	duration := job.Duration()

	if duration < 50*time.Millisecond {
		t.Errorf("Expected duration > 50ms, got %s", duration)
	}
}
