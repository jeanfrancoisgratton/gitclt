// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/jobs.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"context"
	"fmt"
	"gitclt/shared"

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
	renderJobTable(list)
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
	renderJob(j)
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

// JobLog prints a job's log, or saves it with -O (Forgejo 16 and later, GitHub)
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
	r, e := t.runs.WorkflowJobLogs(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return runError(fmt.Sprintf("Unable to get the log of job %d of %s", ids[0], t.where()), "job", e)
	}
	path := OutputFile
	if path == "" {
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
