package consolekit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobKilled    JobStatus = "killed"
)

// Job represents a background process
type Job struct {
	ID        int
	Command   string
	StartTime time.Time
	EndTime   *time.Time
	Status    JobStatus
	PID       int
	Output    *bytes.Buffer
	Error     error
	Cancel    context.CancelFunc
	cmd       *exec.Cmd
	mu        sync.RWMutex
}

// JobManager manages background jobs and scheduled tasks
type JobManager struct {
	jobs           map[int]*Job
	scheduledTasks map[int]*ScheduledTask
	nextID         int
	mu             sync.RWMutex
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:           make(map[int]*Job),
		scheduledTasks: make(map[int]*ScheduledTask),
		nextID:         1,
	}
}

// Add creates a new job and starts tracking it
func (jm *JobManager) Add(command string, ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd) int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        jm.nextID,
		Command:   command,
		StartTime: time.Now(),
		Status:    JobRunning,
		PID:       -1,
		Output:    &bytes.Buffer{},
		Cancel:    cancel,
		cmd:       cmd,
	}

	jm.jobs[jm.nextID] = job
	jm.nextID++

	// Start a goroutine to update job status when it completes
	go func(j *Job) {
		// Wait for the command to complete
		err := cmd.Wait()

		j.mu.Lock()
		defer j.mu.Unlock()

		now := time.Now()
		j.EndTime = &now

		if err != nil {
			j.Error = err
			if ctx.Err() == context.Canceled {
				j.Status = JobKilled
			} else {
				j.Status = JobFailed
			}
		} else {
			j.Status = JobCompleted
		}
	}(job)

	// Set PID after process starts (if available)
	if cmd.Process != nil {
		job.PID = cmd.Process.Pid
	}

	return job.ID
}

// Get retrieves a job by ID
func (jm *JobManager) Get(id int) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, ok := jm.jobs[id]
	return job, ok
}

// List returns all jobs
func (jm *JobManager) List() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// Kill terminates a job by ID
func (jm *JobManager) Kill(id int) error {
	jm.mu.RLock()
	job, ok := jm.jobs[id]
	jm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job %d not found", id)
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != JobRunning {
		return fmt.Errorf("job %d is not running (status: %s)", id, job.Status)
	}

	// Cancel the context
	if job.Cancel != nil {
		job.Cancel()
	}

	// Also try to kill the process directly
	if job.cmd != nil && job.cmd.Process != nil {
		return job.cmd.Process.Kill()
	}

	return nil
}

// KillAll terminates all running jobs
func (jm *JobManager) KillAll() []error {
	jm.mu.RLock()
	runningJobs := make([]*Job, 0)
	for _, job := range jm.jobs {
		job.mu.RLock()
		if job.Status == JobRunning {
			runningJobs = append(runningJobs, job)
		}
		job.mu.RUnlock()
	}
	jm.mu.RUnlock()

	var errors []error
	for _, job := range runningJobs {
		if err := jm.Kill(job.ID); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// Wait blocks until a job completes
func (jm *JobManager) Wait(id int) error {
	job, ok := jm.Get(id)
	if !ok {
		return fmt.Errorf("job %d not found", id)
	}

	// Poll until job is no longer running
	for {
		job.mu.RLock()
		status := job.Status
		job.mu.RUnlock()

		if status != JobRunning {
			job.mu.RLock()
			err := job.Error
			job.mu.RUnlock()
			return err
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Logs returns the output for a job
func (jm *JobManager) Logs(id int) (string, error) {
	job, ok := jm.Get(id)
	if !ok {
		return "", fmt.Errorf("job %d not found", id)
	}

	job.mu.RLock()
	defer job.mu.RUnlock()

	if job.Output != nil {
		return job.Output.String(), nil
	}

	return "", nil
}

// Clean removes completed/failed jobs from the list
func (jm *JobManager) Clean() int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	removed := 0
	for id, job := range jm.jobs {
		job.mu.RLock()
		status := job.Status
		job.mu.RUnlock()

		if status != JobRunning {
			delete(jm.jobs, id)
			removed++
		}
	}

	return removed
}

// Duration returns the duration of a job
func (j *Job) Duration() time.Duration {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if j.EndTime != nil {
		return j.EndTime.Sub(j.StartTime)
	}

	return time.Since(j.StartTime)
}

// getNextID returns the next available ID for tasks (used internally by schedule commands)
func (jm *JobManager) getNextID() int {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	id := jm.nextID
	jm.nextID++
	return id
}

// addScheduledTask adds a scheduled task to tracking
func (jm *JobManager) addScheduledTask(task *ScheduledTask) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.scheduledTasks[task.ID] = task
}

// getScheduledTask retrieves a scheduled task by ID
func (jm *JobManager) getScheduledTask(id int) *ScheduledTask {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	return jm.scheduledTasks[id]
}

// getScheduledTasks returns all scheduled tasks
func (jm *JobManager) getScheduledTasks() []*ScheduledTask {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(jm.scheduledTasks))
	for _, task := range jm.scheduledTasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// removeScheduledTask removes a scheduled task from tracking
func (jm *JobManager) removeScheduledTask(id int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	delete(jm.scheduledTasks, id)
}
