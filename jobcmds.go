package consolekit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AddJobCommands adds job management commands to the CLI
func AddJobCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		// jobs command - list all jobs
		jobsCmd := &cobra.Command{
			Use:   "jobs",
			Short: "List all background jobs",
			Run: func(cmd *cobra.Command, args []string) {
				verbose, _ := cmd.Flags().GetBool("verbose")
				showAll, _ := cmd.Flags().GetBool("all")

				jobs := cli.JobManager.List()

				// Sort by ID
				sort.Slice(jobs, func(i, j int) bool {
					return jobs[i].ID < jobs[j].ID
				})

				if len(jobs) == 0 {
					cmd.Println("No jobs")
					return
				}

				// Filter out non-running jobs unless --all is specified
				displayJobs := jobs
				if !showAll {
					displayJobs = make([]*Job, 0)
					for _, job := range jobs {
						job := job // Create new variable for closure
						func() {
							job.mu.RLock()
							defer job.mu.RUnlock()
							if job.Status == JobRunning {
								displayJobs = append(displayJobs, job)
							}
						}()
					}
				}

				if len(displayJobs) == 0 {
					cmd.Println("No running jobs")
					return
				}

				cmd.Println("Background Jobs:")
				cmd.Println(strings.Repeat("-", 80))

				for _, job := range displayJobs {
					job.mu.RLock()
					id := job.ID
					command := job.Command
					status := job.Status
					pid := job.PID
					duration := job.Duration()
					job.mu.RUnlock()

					// Truncate long commands
					if len(command) > 60 && !verbose {
						command = command[:57] + "..."
					}

					statusStr := string(status)
					if status == JobRunning {
						statusStr = cli.InfoString("[%s]", status)
					} else if status == JobFailed {
						statusStr = cli.ErrorString("[%s]", status)
					} else {
						statusStr = fmt.Sprintf("[%s]", status)
					}

					cmd.Printf("[%d] %s PID:%d Duration:%s\n", id, statusStr, pid, duration.Round(time.Second))
					cmd.Printf("    %s\n", command)

					if verbose {
						job.mu.RLock()
						output := ""
						if job.Output != nil {
							output = job.Output.String()
						}
						job.mu.RUnlock()

						if output != "" {
							preview := strings.Split(output, "\n")
							maxLines := 3
							if len(preview) > maxLines {
								preview = preview[:maxLines]
							}
							cmd.Println("    Output preview:")
							for _, line := range preview {
								if line != "" {
									cmd.Printf("      %s\n", line)
								}
							}
							if len(strings.Split(output, "\n")) > maxLines {
								cmd.Println("      ...")
							}
						}
					}
					cmd.Println()
				}
			},
		}
		jobsCmd.Flags().BoolP("verbose", "v", false, "Show detailed information including output preview")
		jobsCmd.Flags().BoolP("all", "a", false, "Show all jobs including completed ones")

		// job command - manage individual jobs
		jobCmd := &cobra.Command{
			Use:   "job [id] [action]",
			Short: "Manage a specific job",
			Long: `Manage a specific background job.

Actions:
  (none)  - Show job details
  logs    - Show full output
  kill    - Kill the job
  wait    - Wait for job to complete`,
			Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				idStr := args[0]
				id, err := strconv.Atoi(idStr)
				if err != nil {
					cmd.PrintErrf("Invalid job ID: %s\n", idStr)
					return
				}

				job, ok := cli.JobManager.Get(id)
				if !ok {
					cmd.PrintErrf("Job %d not found\n", id)
					return
				}

				// Determine action
				action := ""
				if len(args) > 1 {
					action = args[1]
				}

				switch action {
				case "":
					// Show job details
					showJobDetails(cmd, cli, job)

				case "logs":
					// Show full output
					logs, err := cli.JobManager.Logs(id)
					if err != nil {
						cmd.PrintErrf("Error getting logs: %v\n", err)
						return
					}
					if logs == "" {
						cmd.Println("No output")
					} else {
						cmd.Print(logs)
					}

				case "kill":
					// Kill the job
					err := cli.JobManager.Kill(id)
					if err != nil {
						cmd.PrintErrf("Error killing job: %v\n", err)
						return
					}
					cmd.Printf("Job %d killed\n", id)

				case "wait":
					// Wait for job to complete
					cmd.Printf("Waiting for job %d...\n", id)
					err := cli.JobManager.Wait(id)
					if err != nil {
						cmd.PrintErrf("Job %d failed: %v\n", id, err)
						return
					}
					cmd.Printf("Job %d completed\n", id)

				default:
					cmd.PrintErrf("Unknown action: %s\n", action)
					cmd.Println("Valid actions: logs, kill, wait")
				}
			},
		}

		// killall command - kill all running jobs
		killallCmd := &cobra.Command{
			Use:   "killall",
			Short: "Kill all running background jobs",
			Run: func(cmd *cobra.Command, args []string) {
				errors := cli.JobManager.KillAll()
				if len(errors) > 0 {
					cmd.Println("Errors killing some jobs:")
					for _, err := range errors {
						cmd.PrintErrf("  %v\n", err)
					}
					return
				}
				cmd.Println("All jobs killed")
			},
		}

		// jobclean command - remove completed jobs
		jobcleanCmd := &cobra.Command{
			Use:   "jobclean",
			Short: "Remove completed/failed jobs from the list",
			Run: func(cmd *cobra.Command, args []string) {
				removed := cli.JobManager.Clean()
				cmd.Printf("Removed %d completed/failed job(s)\n", removed)
			},
		}

		rootCmd.AddCommand(jobsCmd)
		rootCmd.AddCommand(jobCmd)
		rootCmd.AddCommand(killallCmd)
		rootCmd.AddCommand(jobcleanCmd)
	}
}

func showJobDetails(cmd *cobra.Command, cli *CLI, job *Job) {
	job.mu.RLock()
	defer job.mu.RUnlock()

	cmd.Println(strings.Repeat("=", 80))
	cmd.Printf("Job ID: %d\n", job.ID)
	cmd.Printf("Command: %s\n", job.Command)
	cmd.Printf("Status: %s\n", job.Status)
	cmd.Printf("PID: %d\n", job.PID)
	cmd.Printf("Started: %s\n", job.StartTime.Format("2006-01-02 15:04:05"))

	if job.EndTime != nil {
		cmd.Printf("Ended: %s\n", job.EndTime.Format("2006-01-02 15:04:05"))
	}

	cmd.Printf("Duration: %s\n", job.Duration().Round(time.Second))

	if job.Error != nil {
		cmd.Printf("Error: %v\n", cli.ErrorString("%v", job.Error))
	}

	if job.Output != nil && job.Output.Len() > 0 {
		cmd.Println(strings.Repeat("-", 80))
		cmd.Println("Output (last 20 lines):")
		lines := strings.Split(job.Output.String(), "\n")
		start := 0
		if len(lines) > 20 {
			start = len(lines) - 20
			cmd.Println("...")
		}
		for _, line := range lines[start:] {
			if line != "" {
				cmd.Println(line)
			}
		}
	}
	cmd.Println(strings.Repeat("=", 80))
}
