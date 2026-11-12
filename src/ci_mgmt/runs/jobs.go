// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/jobs.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"context"
	"errors"
	"fmt"
	"gitclt/shared"
	"os"
	"os/signal"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListJobs lists a run's jobs (Forgejo 16 and later, GitHub)
func ListJobs(repo, arg string, displayoutput bool) ([]gol.WorkflowJob, *cerr.CustomError) {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "run")
	if ce != nil {
		return nil, ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.WorkflowJob], error) {
		return t.runs.ListWorkflowJobs(ctx, t.owner, t.repo, ids[0], o)
	})
	if e != nil {
		return nil, runError(fmt.Sprintf("Unable to list the jobs of run %d of %s", ids[0], t.where()), "run", e)
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "job list")
	}
	if shared.QuietOutput {
		for _, j := range list {
			fmt.Println(j.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("Run %d of %s has no job\n", ids[0], t.where())
		return list, nil
	}
	shared.Page(renderJobTable(list), shared.TableHeader)
	return list, nil
}

// GetJob shows a job, with its steps (Forgejo 17 and later, GitHub)
func GetJob(repo, arg string, displayoutput bool) (*gol.WorkflowJob, *cerr.CustomError) {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "job")
	if ce != nil {
		return nil, ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	j, e := t.runs.GetWorkflowJob(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return nil, runError(fmt.Sprintf("Unable to get job %d of %s", ids[0], t.where()), "job", e)
	}
	if !displayoutput {
		return j, nil
	}
	if JSONoutput {
		return j, shared.PrintJSON(j, "job")
	}
	// The job's line stays on top
	shared.Page(renderJob(j), 1)
	return j, nil
}

// RerunJob starts a finished job again, with the jobs that depend on it (Forgejo 17 and later, GitHub)
func RerunJob(repo, arg string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "job")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	if e := t.runs.RerunWorkflowJob(ctx, t.owner, t.repo, ids[0]); e != nil {
		return runError(fmt.Sprintf("Unable to rerun job %d of %s", ids[0], t.where()), "job", e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Started job %d of %s again, with the jobs that depend on it", ids[0], t.where())))
	}
	return nil
}

// JobLog shows a job's log, paged on a terminal; saves it with -O, or follows it with -f (Forgejo 16 and later, GitHub)
func JobLog(repo, arg string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "job")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	if FollowLog {
		return followJobLog(ctx, t, ids[0])
	}
	r, e := t.runs.WorkflowJobLogs(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return runError(fmt.Sprintf("Unable to get the log of job %d of %s", ids[0], t.where()), "job", e)
	}

	path := OutputFile
	if path == "" {
		if hftx.Interactive() {
			return pageLog(r, fmt.Sprintf("Log of job %d of %s", ids[0], t.where()))
		}
		path = "-"
	}
	n, ce := save(r, path)
	if ce != nil {
		return ce
	}
	if path != "-" && !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Saved the log of job %d to %s (%s)", ids[0], hftx.Brown(path), size(n))))
	}
	return nil
}

// followJobLog writes the job's log as it grows, until the job ends; Ctrl-C stops following, and is no error
// GitHub only has a log once the job ended: the wait is said, on the standard error, or it would look hung
func followJobLog(ctx context.Context, t target, jobID int64) *cerr.CustomError {
	f, ok := t.c.Client.(gol.WorkflowLogFollower)
	if !ok {
		return &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: following a job's log is only available on Forgejo and GitHub", t.c.Connection.Name)}
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	if t.c.Backend() == gol.BackendGitHub && !shared.QuietOutput {
		fmt.Fprintln(os.Stderr, hftx.InProgressSign(fmt.Sprintf("Waiting for job %d to end: GitHub only has a job's log once it finished", jobID)))
	}
	e := f.FollowWorkflowJobLog(ctx, t.owner, t.repo, jobID, os.Stdout, gol.FollowLogOptions{})
	switch {
	case errors.Is(e, context.Canceled):
		return nil
	case e != nil:
		return runError(fmt.Sprintf("Unable to follow the log of job %d of %s", jobID, t.where()), "job", e)
	}
	return nil
}
