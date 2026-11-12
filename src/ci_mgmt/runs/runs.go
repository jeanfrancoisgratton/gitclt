// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/runs.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// Run is what run get shows: the run, and its jobs where the forge can list them
type Run struct {
	gol.WorkflowRun
	Jobs []gol.WorkflowJob `json:"jobs,omitzero"`
	// JobsError tells why the jobs are missing (ex: a Forgejo older than 16)
	JobsError string `json:"jobsError,omitzero"`
}

// ListRuns lists the runs of owner/repo, newest first; at most ListLimit of them, so that a long history is not read whole
func ListRuns(repo string, displayoutput bool) ([]gol.WorkflowRun, *cerr.CustomError) {
	ctx := context.Background()
	if ListLimit < 0 {
		return nil, &cerr.CustomError{Title: "Invalid limit", Message: fmt.Sprintf("%d is not a run count", ListLimit)}
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}

	opts := gol.ListWorkflowRunsOptions{Workflow: ListWorkflow, Status: ListStatus, Event: ListEvent, Branch: ListBranch, SHA: ListSHA}
	opts.PerPage = shared.ListPageSize
	if ListLimit > 0 {
		opts.PerPage = min(ListLimit, shared.ListPageSize)
	}
	list := []gol.WorkflowRun{}
	for run, e := range gol.All(ctx, opts.ListOptions, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.WorkflowRun], error) {
		opts.ListOptions = o
		return t.runs.ListWorkflowRuns(ctx, t.owner, t.repo, opts)
	}) {
		if e != nil {
			return nil, runError("Unable to list the runs of "+t.where(), "run", e)
		}
		list = append(list, run)
		if ListLimit > 0 && len(list) == ListLimit {
			break
		}
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "run list")
	}
	if shared.QuietOutput {
		for _, r := range list {
			fmt.Println(r.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("No run of %s matches\n", t.where())
		return list, nil
	}
	shared.Page(renderRunTable(list), shared.TableHeader)
	return list, nil
}

// GetRun shows a run, with its jobs; an older Forgejo cannot list them, which does not hide the run
func GetRun(repo, arg string, displayoutput bool) (*Run, *cerr.CustomError) {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "run")
	if ce != nil {
		return nil, ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	r, e := t.runs.GetWorkflowRun(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return nil, runError(fmt.Sprintf("Unable to get run %d of %s", ids[0], t.where()), "run", e)
	}
	out := &Run{WorkflowRun: *r}
	jobs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.WorkflowJob], error) {
		return t.runs.ListWorkflowJobs(ctx, t.owner, t.repo, r.ID, o)
	})
	switch {
	case errors.Is(e, gol.ErrNotSupported):
		out.JobsError = e.Error()
	case e != nil:
		return nil, forge.Error(fmt.Sprintf("Unable to list the jobs of run %d", r.ID), e)
	default:
		out.Jobs = jobs
	}

	if !displayoutput {
		return out, nil
	}
	if JSONoutput {
		return out, shared.PrintJSON(out, "run")
	}
	// The run's line stays on top
	shared.Page(renderRun(out), 1)
	return out, nil
}

// CancelRuns stops runs; those already finished are skipped, as the forges take cancelling one for a success
// Every run is checked first; the run stops at the first failure, and the runs cancelled before it stay cancelled
func CancelRuns(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs(args, "run")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	var todo []int64
	for _, id := range ids {
		r, e := t.runs.GetWorkflowRun(ctx, t.owner, t.repo, id)
		if e != nil {
			return runError(fmt.Sprintf("Unable to get run %d of %s", id, t.where()), "run", e)
		}
		if r.Conclusion != "" {
			if !shared.QuietOutput {
				fmt.Printf("Run %d of %s already finished: %s\n", id, t.where(), r.Conclusion)
			}
			continue
		}
		todo = append(todo, id)
	}
	for _, id := range todo {
		if e := t.runs.CancelWorkflowRun(ctx, t.owner, t.repo, id); e != nil {
			return runError(fmt.Sprintf("Unable to cancel run %d of %s", id, t.where()), "run", e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Cancelled run %d of %s", id, t.where())))
		}
	}
	return nil
}

// RerunRun starts a finished run's jobs again, under the same ID
func RerunRun(repo, arg string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "run")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	if e := t.runs.RerunWorkflowRun(ctx, t.owner, t.repo, ids[0]); e != nil {
		return runError(fmt.Sprintf("Unable to rerun run %d of %s", ids[0], t.where()), "run", e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Started run %d of %s again", ids[0], t.where())))
	}
	return nil
}

// DeleteRuns removes finished runs, after confirmation; every one is checked before asking, and one still going is refused
// The run stops at the first failure; the runs deleted before it stay deleted
func DeleteRuns(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs(args, "run")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	shown := make([]string, len(ids))
	for i, id := range ids {
		r, e := t.runs.GetWorkflowRun(ctx, t.owner, t.repo, id)
		if e != nil {
			return runError(fmt.Sprintf("Unable to get run %d of %s", id, t.where()), "run", e)
		}
		if r.Conclusion == "" {
			return &cerr.CustomError{Title: "Run not finished",
				Message: fmt.Sprintf("run %d of %s is %s: cancel it first (gitclt run cancel %d); nothing was deleted", id, t.where(), r.Status, id)}
		}
		shown[i] = fmt.Sprintf("%d (%s, %s)", id, r.Title, r.Conclusion)
	}
	if !confirm(fmt.Sprintf("Delete run(s) %s of %s, with their logs and artifacts ? [y/N]: ", strings.Join(shown, ", "), t.where())) {
		fmt.Println("Aborted, no run is deleted")
		return nil
	}
	for _, id := range ids {
		if e := t.runs.DeleteWorkflowRun(ctx, t.owner, t.repo, id); e != nil {
			if errors.Is(e, gol.ErrWorkflowRunActive) {
				return &cerr.CustomError{Title: "Run not finished", Message: fmt.Sprintf("run %d started again in the meantime: cancel it first", id)}
			}
			return runError(fmt.Sprintf("Unable to delete run %d of %s", id, t.where()), "run", e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted run %d of %s", id, t.where())))
		}
	}
	return nil
}

// RunLogs saves the logs of a run's jobs, as the ZIP archive the forge makes
func RunLogs(repo, arg string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "run")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	path := OutputFile
	if path == "" {
		path = "run-" + strconv.FormatInt(ids[0], 10) + "-logs.zip"
	}
	r, e := t.runs.WorkflowRunLogs(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return runError(fmt.Sprintf("Unable to get the logs of run %d of %s", ids[0], t.where()), "run", e)
	}
	n, ce := save(r, path)
	if ce != nil {
		return ce
	}
	if path != "-" && !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Saved the logs of run %d to %s (%s)", ids[0], hftx.Brown(path), size(n))))
	}
	return nil
}
