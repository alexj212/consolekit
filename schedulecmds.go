package consolekit

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// ScheduledTask represents a scheduled command
type ScheduledTask struct {
	ID       int
	Command  string
	Time     time.Time
	Interval time.Duration
	Repeat   bool
	Enabled  bool
	timer    *time.Timer
	ticker   *time.Ticker
	done     chan bool
	mu       sync.RWMutex
}

// AddScheduleCommands adds command scheduling commands
func AddScheduleCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var scheduleCmd = &cobra.Command{
			Use:   "schedule",
			Short: "Schedule commands to run at specific times",
			Long:  "Schedule commands to run once, after a delay, or at regular intervals",
		}

		// schedule at - run command at specific time
		var atCmd = &cobra.Command{
			Use:   "at [time] [command]",
			Short: "Run command at a specific time",
			Long: `Schedule a command to run once at a specific time.
Time format: HH:MM (24-hour) or HH:MM:SS

Examples:
  schedule at 14:30 "print Afternoon reminder"
  schedule at 09:00:00 "print Good morning"`,
			Args: cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				timeStr := args[0]
				command := strings.Join(args[1:], " ")

				// Parse time
				var hour, min, sec int
				var err error

				parts := strings.Split(timeStr, ":")
				if len(parts) < 2 || len(parts) > 3 {
					cmd.PrintErrln(cli.ErrorString("Invalid time format. Use HH:MM or HH:MM:SS"))
					return
				}

				hour, err = strconv.Atoi(parts[0])
				if err != nil || hour < 0 || hour > 23 {
					cmd.PrintErrln(cli.ErrorString("Invalid hour"))
					return
				}

				min, err = strconv.Atoi(parts[1])
				if err != nil || min < 0 || min > 59 {
					cmd.PrintErrln(cli.ErrorString("Invalid minute"))
					return
				}

				if len(parts) == 3 {
					sec, err = strconv.Atoi(parts[2])
					if err != nil || sec < 0 || sec > 59 {
						cmd.PrintErrln(cli.ErrorString("Invalid second"))
						return
					}
				}

				// Calculate target time
				now := time.Now()
				targetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, 0, now.Location())

				// If time has passed today, schedule for tomorrow
				if targetTime.Before(now) {
					targetTime = targetTime.Add(24 * time.Hour)
				}

				duration := targetTime.Sub(now)

				task := &ScheduledTask{
					ID:      cli.JobManager.getNextID(),
					Command: command,
					Time:    targetTime,
					Repeat:  false,
					Enabled: true,
					done:    make(chan bool),
				}

				// Start timer
				task.timer = time.AfterFunc(duration, func() {
					output, err := cli.ExecuteLine(task.Command, nil)
					if output != "" {
						fmt.Print(output)
						if !strings.HasSuffix(output, "\n") {
							fmt.Println()
						}
					}
					if err != nil {
						fmt.Printf("Scheduled command error: %v\n", err)
					}
					// Remove from schedule list after execution
					cli.JobManager.removeScheduledTask(task.ID)
				})

				cli.JobManager.addScheduledTask(task)

				cmd.Println(cli.SuccessString(fmt.Sprintf("Scheduled task %d to run at %s", task.ID, targetTime.Format("15:04:05"))))
			},
		}

		// schedule in - run command after delay
		var inCmd = &cobra.Command{
			Use:   "in [duration] [command]",
			Short: "Run command after a delay",
			Long: `Schedule a command to run once after a specified delay.
Duration format: Xs (seconds), Xm (minutes), Xh (hours)

Examples:
  schedule in 30s "print Timer done"
  schedule in 5m "print 5 minutes elapsed"
  schedule in 1h "print 1 hour has passed"`,
			Args: cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				durationStr := args[0]
				command := strings.Join(args[1:], " ")

				duration, err := time.ParseDuration(durationStr)
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Invalid duration: %v", err)))
					return
				}

				targetTime := time.Now().Add(duration)

				task := &ScheduledTask{
					ID:      cli.JobManager.getNextID(),
					Command: command,
					Time:    targetTime,
					Repeat:  false,
					Enabled: true,
					done:    make(chan bool),
				}

				// Start timer
				task.timer = time.AfterFunc(duration, func() {
					output, err := cli.ExecuteLine(task.Command, nil)
					if output != "" {
						fmt.Print(output)
						if !strings.HasSuffix(output, "\n") {
							fmt.Println()
						}
					}
					if err != nil {
						fmt.Printf("Scheduled command error: %v\n", err)
					}
					// Remove from schedule list after execution
					cli.JobManager.removeScheduledTask(task.ID)
				})

				cli.JobManager.addScheduledTask(task)

				cmd.Println(cli.SuccessString(fmt.Sprintf("Scheduled task %d to run in %s", task.ID, duration)))
			},
		}

		// schedule every - run command at regular intervals
		var everyCmd = &cobra.Command{
			Use:   "every [interval] [command]",
			Short: "Run command at regular intervals",
			Long: `Schedule a command to run repeatedly at regular intervals.
Interval format: Xs (seconds), Xm (minutes), Xh (hours)

Examples:
  schedule every 10s "print Tick"
  schedule every 1m "print Minute passed"
  schedule every 1h "print Hourly check"`,
			Args: cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				intervalStr := args[0]
				command := strings.Join(args[1:], " ")

				interval, err := time.ParseDuration(intervalStr)
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Invalid interval: %v", err)))
					return
				}

				task := &ScheduledTask{
					ID:       cli.JobManager.getNextID(),
					Command:  command,
					Interval: interval,
					Repeat:   true,
					Enabled:  true,
					done:     make(chan bool),
				}

				// Start ticker
				task.ticker = time.NewTicker(interval)
				go func() {
					for {
						select {
						case <-task.ticker.C:
							task.mu.RLock()
							enabled := task.Enabled
							command := task.Command
							task.mu.RUnlock()

							if enabled {
								output, err := cli.ExecuteLine(command, nil)
								if output != "" {
									fmt.Print(output)
									if !strings.HasSuffix(output, "\n") {
										fmt.Println()
									}
								}
								if err != nil {
									fmt.Printf("Scheduled command error: %v\n", err)
								}
							}
						case <-task.done:
							return
						}
					}
				}()

				cli.JobManager.addScheduledTask(task)

				cmd.Println(cli.SuccessString(fmt.Sprintf("Scheduled repeating task %d every %s", task.ID, interval)))
			},
		}

		// schedule list - list all scheduled tasks
		var listCmd = &cobra.Command{
			Use:   "list",
			Short: "List all scheduled tasks",
			Run: func(cmd *cobra.Command, args []string) {
				tasks := cli.JobManager.getScheduledTasks()

				if len(tasks) == 0 {
					cmd.Println("No scheduled tasks")
					return
				}

				cmd.Println("Scheduled Tasks:")
				for _, task := range tasks {
					task.mu.RLock()
					enabled := task.Enabled
					repeat := task.Repeat
					interval := task.Interval
					command := task.Command
					taskTime := task.Time
					taskID := task.ID
					task.mu.RUnlock()

					status := "enabled"
					if !enabled {
						status = "disabled"
					}

					if repeat {
						cmd.Printf("[%d] Every %s - %s (%s)\n", taskID, interval, command, status)
					} else {
						cmd.Printf("[%d] At %s - %s (%s)\n", taskID, taskTime.Format("15:04:05"), command, status)
					}
				}
			},
		}

		// schedule cancel - cancel a scheduled task
		var cancelCmd = &cobra.Command{
			Use:   "cancel [id]",
			Short: "Cancel a scheduled task",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				id, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.PrintErrln(cli.ErrorString("Invalid task ID"))
					return
				}

				task := cli.JobManager.getScheduledTask(id)
				if task == nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Task %d not found", id)))
					return
				}

				// Stop timer/ticker and close done channel to signal goroutine exit
				if task.timer != nil {
					task.timer.Stop()
				}
				if task.ticker != nil {
					task.ticker.Stop()
				}
				if task.done != nil {
					close(task.done)
				}

				cli.JobManager.removeScheduledTask(id)

				cmd.Println(cli.SuccessString(fmt.Sprintf("Cancelled task %d", id)))
			},
		}

		// schedule pause - pause a repeating task
		var pauseCmd = &cobra.Command{
			Use:   "pause [id]",
			Short: "Pause a repeating task",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				id, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.PrintErrln(cli.ErrorString("Invalid task ID"))
					return
				}

				task := cli.JobManager.getScheduledTask(id)
				if task == nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Task %d not found", id)))
					return
				}

				if !task.Repeat {
					cmd.PrintErrln(cli.ErrorString("Cannot pause one-time tasks (use cancel instead)"))
					return
				}

				task.mu.Lock()
				task.Enabled = false
				task.mu.Unlock()
				cmd.Println(cli.SuccessString(fmt.Sprintf("Paused task %d", id)))
			},
		}

		// schedule resume - resume a paused task
		var resumeCmd = &cobra.Command{
			Use:   "resume [id]",
			Short: "Resume a paused task",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				id, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.PrintErrln(cli.ErrorString("Invalid task ID"))
					return
				}

				task := cli.JobManager.getScheduledTask(id)
				if task == nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Task %d not found", id)))
					return
				}

				if !task.Repeat {
					cmd.PrintErrln(cli.ErrorString("Cannot resume one-time tasks"))
					return
				}

				task.mu.Lock()
				task.Enabled = true
				task.mu.Unlock()
				cmd.Println(cli.SuccessString(fmt.Sprintf("Resumed task %d", id)))
			},
		}

		scheduleCmd.AddCommand(atCmd)
		scheduleCmd.AddCommand(inCmd)
		scheduleCmd.AddCommand(everyCmd)
		scheduleCmd.AddCommand(listCmd)
		scheduleCmd.AddCommand(cancelCmd)
		scheduleCmd.AddCommand(pauseCmd)
		scheduleCmd.AddCommand(resumeCmd)

		rootCmd.AddCommand(scheduleCmd)
	}
}
