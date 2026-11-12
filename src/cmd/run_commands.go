// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/run_commands.go
// Original timestamp : 2026.11.07 10:00:00

package cmd

import (
	"fmt"
	"gitclt/ci_mgmt/runs"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"runs"},
	Short:   "CI workflow run commands: Forgejo Actions and GitHub Actions",
	Long: `The runs of a repository's workflows, their jobs, logs and artifacts; Bitbucket Pipelines is not supported yet.
On Forgejo, listing and reading runs work from 15; cancel, delete, jobs, logs and artifacts need 16; a job's steps and reruns need 17.
A run's ID is the one the list shows, not its number (#), which counts the repository's runs. Starting a workflow by hand is not offered.`,
}

var runListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt run list [-c CONNECTION] [-o OWNER] [-w WORKFLOW] [-s STATUS] [-e EVENT] [-b BRANCH] [--sha SHA] [-n COUNT] [-j] [<REPO>]\n" +
		"  gitclt run ls -w build.yml -s failure",
	Short: "Lists the runs, newest first; -q outputs only their IDs",
	Long: `-w, --workflow is the workflow's file name (build.yml); one that does not exist selects nothing.
-s, --status is the forge's own word: on Forgejo waiting, blocked, running, or an outcome such as success or failure; on GitHub queued,
in_progress, completed, or an outcome. The status column shows the outcome once a run ended.
-n, --limit caps the count (default 30; 0: all of them).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := runs.ListRuns(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <RUN-ID>",
	Aliases: []string{"info", "show"},
	Example: "  gitclt run get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <RUN-ID>",
	Short:   "Shows a run, with its jobs",
	Long:    "Forgejo before 16 cannot list a run's jobs: the run is then shown alone.",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := runs.GetRun(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runCancelCmd = &cobra.Command{
	Use:     "cancel [<REPO>] <RUN-ID>...",
	Aliases: []string{"stop"},
	Example: "  gitclt run cancel [-c CONNECTION] [-o OWNER] [<REPO>] <RUN-ID>...",
	Short:   "Stops runs; those already finished are skipped (Forgejo 16 and later)",
	Long:    "Every run is checked first. The run stops at the first failure; the runs cancelled before it stay cancelled.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, ids := splitRepoNumber(args, 1)
		if e := runs.CancelRuns(repo, ids); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runRerunCmd = &cobra.Command{
	Use:     "rerun [<REPO>] <RUN-ID>",
	Example: "  gitclt run rerun [-c CONNECTION] [-o OWNER] [<REPO>] <RUN-ID>",
	Short:   "Starts a finished run's jobs again, under the same ID (Forgejo 17 and later)",
	Long:    "To start one job again, with those that depend on it, see gitclt run job rerun.",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := runs.RerunRun(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <RUN-ID>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt run delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <RUN-ID>...",
	Short:   "Deletes finished runs, with their logs and artifacts, after confirmation (Forgejo 16 and later)",
	Long: `Every run is checked before asking, and one still going is refused: cancel it first.
The run stops at the first failure; the runs deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, ids := splitRepoNumber(args, 1)
		if e := runs.DeleteRuns(repo, ids); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runLogsCmd = &cobra.Command{
	Use: "logs [<REPO>] <RUN-ID>",
	Example: "  gitclt run logs [-c CONNECTION] [-o OWNER] [-O FILE] [--force] [<REPO>] <RUN-ID>\n" +
		"  gitclt run logs -O - 42 | unzip -l /dev/stdin",
	Short: "Saves the logs of a run's jobs, as a ZIP archive (Forgejo 16 and later)",
	Long: `The archive is saved as run-<RUN-ID>-logs.zip, unless -O, --output names another file, or - for the standard output.
An existing file is only replaced with --force. Forgejo adds an empty .MISSING entry for a job with no log yet.
To read one job's log as text, see gitclt run job log.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := runs.RunLogs(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runJobCmd = &cobra.Command{
	Use:     "job",
	Aliases: []string{"jobs"},
	Short:   "Workflow job commands: the jobs of a run",
	Long:    "A job's ID is the one gitclt run get or gitclt run job list shows.",
}

var runJobListCmd = &cobra.Command{
	Use:     "list [<REPO>] <RUN-ID>",
	Aliases: []string{"ls"},
	Example: "  gitclt run job list [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <RUN-ID>",
	Short:   "Lists a run's jobs; -q outputs only their IDs (Forgejo 16 and later)",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := runs.ListJobs(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runJobGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <JOB-ID>",
	Aliases: []string{"info", "show"},
	Example: "  gitclt run job get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <JOB-ID>",
	Short:   "Shows a job, with its steps (Forgejo 17 and later)",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := runs.GetJob(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runJobRerunCmd = &cobra.Command{
	Use:     "rerun [<REPO>] <JOB-ID>",
	Example: "  gitclt run job rerun [-c CONNECTION] [-o OWNER] [<REPO>] <JOB-ID>",
	Short:   "Starts a finished job again, with the jobs that depend on it (Forgejo 17 and later)",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := runs.RerunJob(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runJobLogCmd = &cobra.Command{
	Use:     "log [<REPO>] <JOB-ID>",
	Aliases: []string{"logs"},
	Example: "  gitclt run job log [-c CONNECTION] [-o OWNER] [-f | -O FILE [--force]] [<REPO>] <JOB-ID>\n" +
		"  gitclt run job log -f 99",
	Short: "Shows a job's log, as text (Forgejo 16 and later)",
	Long: `On a terminal, the log is shown through a pager: Space for the next page, Enter for the next line, ^U for the previous
page, ^G for the position, Q to quit. Piped or redirected, it is written as is to the standard output.
-O, --output saves it to a file instead; an existing file is only replaced with --force.
-f, --follow writes the log as it grows, until the job ends (Ctrl-C stops following, without stopping the job); it needs
Forgejo 17. GitHub only has a job's log once the job finished: -f then waits, and writes it all at the end.
GitHub can take a moment to have a finished job's log.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := runs.JobLog(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runArtifactCmd = &cobra.Command{
	Use:     "artifact",
	Aliases: []string{"artifacts"},
	Short:   "Workflow artifact commands: the files runs store (Forgejo 16 and later)",
}

var runArtifactListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt run artifact list [-c CONNECTION] [-o OWNER] [--run RUN-ID] [--name NAME] [-j] [<REPO>]",
	Short:   "Lists the artifacts, of the repository or of one run; -q outputs only their IDs",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := runs.ListArtifacts(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runArtifactGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <ARTIFACT-ID>",
	Aliases: []string{"info"},
	Example: "  gitclt run artifact get [-c CONNECTION] [-o OWNER] [-j] [<REPO>] <ARTIFACT-ID>",
	Short:   "Shows an artifact",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if _, e := runs.GetArtifact(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runArtifactDownloadCmd = &cobra.Command{
	Use:     "download [<REPO>] <ARTIFACT-ID>",
	Aliases: []string{"dl"},
	Example: "  gitclt run artifact download [-c CONNECTION] [-o OWNER] [-O FILE] [--force] [<REPO>] <ARTIFACT-ID>",
	Short:   "Saves an artifact, as a ZIP archive",
	Long: `The archive is saved as <NAME>.zip in the current directory, unless -O, --output names another file, or - for the standard output.
An existing file is only replaced with --force; an expired artifact cannot be downloaded.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepo(args, 1)
		if e := runs.DownloadArtifact(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runArtifactDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <ARTIFACT-ID>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt run artifact delete [-c CONNECTION] [-o OWNER] [-y] [<REPO>] <ARTIFACT-ID>...",
	Short:   "Deletes artifacts, after confirmation",
	Long:    "Every artifact is checked before asking. The run stops at the first failure; the artifacts deleted before it stay deleted.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, ids := splitRepoNumber(args, 1)
		if e := runs.DeleteArtifacts(repo, ids); e != nil {
			fmt.Println(e.Error())
		}
	},
}

const runIDHelp = `An ID is never taken for <REPO>, so several need no . (gitclt run cancel 41 42);
a repository with a numeric name cannot be designated by this command.`

func init() {
	runCmd.AddCommand(runListCmd, runGetCmd, runCancelCmd, runRerunCmd, runDeleteCmd, runLogsCmd, runJobCmd, runArtifactCmd)
	runJobCmd.AddCommand(runJobListCmd, runJobGetCmd, runJobRerunCmd, runJobLogCmd)
	runArtifactCmd.AddCommand(runArtifactListCmd, runArtifactGetCmd, runArtifactDownloadCmd, runArtifactDeleteCmd)

	runCmd.PersistentFlags().StringVarP(&runs.OwnerOrg, "owner", "o", "", ownerFlagHelp)

	runListCmd.Flags().StringVarP(&runs.ListWorkflow, "workflow", "w", "", "only the runs of this workflow file (ex: build.yml)")
	runListCmd.Flags().StringVarP(&runs.ListStatus, "status", "s", "", "only the runs with this status or outcome, in the forge's words")
	runListCmd.Flags().StringVarP(&runs.ListEvent, "event", "e", "", "only the runs started by this event (ex: push)")
	runListCmd.Flags().StringVarP(&runs.ListBranch, "branch", "b", "", "only the runs of this branch")
	runListCmd.Flags().StringVar(&runs.ListSHA, "sha", "", "only the runs of this commit (its full SHA)")
	runListCmd.Flags().IntVarP(&runs.ListLimit, "limit", "n", 30, "at most this many runs (0: all of them)")

	for _, c := range []*cobra.Command{runListCmd, runGetCmd, runJobListCmd, runJobGetCmd, runArtifactListCmd, runArtifactGetCmd} {
		c.Flags().BoolVarP(&runs.JSONoutput, "json", "j", false, "output as JSON")
	}
	for _, c := range []*cobra.Command{runDeleteCmd, runArtifactDeleteCmd} {
		c.Flags().BoolVarP(&runs.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
	}
	for _, c := range []*cobra.Command{runLogsCmd, runJobLogCmd, runArtifactDownloadCmd} {
		c.Flags().StringVarP(&runs.OutputFile, "output", "O", "", "the file to write, or - for the standard output")
		c.Flags().BoolVar(&runs.Overwrite, "force", false, "replace the file if it exists")
	}
	runJobLogCmd.Flags().BoolVarP(&runs.FollowLog, "follow", "f", false, "write the log as it grows, until the job ends")
	runJobLogCmd.MarkFlagsMutuallyExclusive("follow", "output")
	runArtifactListCmd.Flags().Int64Var(&runs.ArtifactRun, "run", 0, "only the artifacts of this run")
	runArtifactListCmd.Flags().StringVar(&runs.ArtifactName, "name", "", "only the artifacts with this exact name")

	addHelp(runCmd, repoInferenceHelp)
	for _, c := range []*cobra.Command{runCancelCmd, runDeleteCmd, runArtifactDeleteCmd} {
		addHelp(c, runIDHelp)
	}
}
