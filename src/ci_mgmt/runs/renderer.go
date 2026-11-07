// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/renderer.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006.01.02 15:04")
}

// short abbreviates a commit, as git does
func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// outcome is how something ended, or its status while it goes on: the forges only set the conclusion once it ends
func outcome(status, conclusion string) string {
	if conclusion != "" {
		return conclusion
	}
	return status
}

// elapsed is how long something took, or has been going; empty when the forge does not tell
func elapsed(start, stop time.Time) string {
	if start.IsZero() {
		return ""
	}
	if stop.IsZero() {
		stop = time.Now()
	}
	return stop.Sub(start).Round(time.Second).String()
}

func newTable(header table.Row) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(header)
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(77)}
	t.Style().Format.Header = text.FormatDefault
	return t
}

func renderRunTable(list []gol.WorkflowRun) {
	t := newTable(table.Row{"ID", "#", "Workflow", "Title", "Branch", "Event", "Status", "Actor", "Created"})
	for _, r := range list {
		t.AppendRow(table.Row{r.ID, r.Number, r.Workflow, r.Title, r.Branch, r.Event, outcome(r.Status, r.Conclusion), r.Actor, date(r.CreatedAt)})
	}
	t.Render()
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

func renderRun(r *Run) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d (#%d)\n", hftx.Blue("Run:"), r.ID, r.Number)
	printIfSet(w, "Title:", r.Title)
	printIfSet(w, "Workflow:", r.Workflow)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), outcome(r.Status, r.Conclusion))
	printIfSet(w, "Event:", r.Event)
	printIfSet(w, "Branch:", r.Branch)
	printIfSet(w, "Commit:", short(r.SHA))
	printIfSet(w, "Actor:", r.Actor)
	printIfSet(w, "Created:", date(r.CreatedAt))
	printIfSet(w, "Started:", date(r.StartedAt))
	printIfSet(w, "Stopped:", date(r.StoppedAt))
	if !r.StoppedAt.IsZero() || r.Conclusion == "" {
		printIfSet(w, "Duration:", elapsed(r.StartedAt, r.StoppedAt))
	}
	printIfSet(w, "URL:", r.HTMLURL)
	_ = w.Flush()
	switch {
	case r.JobsError != "":
		fmt.Println()
		fmt.Println(hftx.InfoSign("No jobs: " + r.JobsError))
	case len(r.Jobs) > 0:
		fmt.Println()
		renderJobTable(r.Jobs)
	}
}

func renderJobTable(list []gol.WorkflowJob) {
	t := newTable(table.Row{"Job ID", "Name", "Status", "Attempt", "Runs on", "Duration"})
	for _, j := range list {
		t.AppendRow(table.Row{j.ID, j.Name, outcome(j.Status, j.Conclusion), j.Attempt, strings.Join(j.Labels, ", "), elapsed(j.StartedAt, j.CompletedAt)})
	}
	t.Render()
}

func renderJob(j *gol.WorkflowJob) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Job:"), j.ID)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Run:"), j.RunID)
	printIfSet(w, "Name:", j.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), outcome(j.Status, j.Conclusion))
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Attempt:"), j.Attempt)
	printIfSet(w, "Runs on:", strings.Join(j.Labels, ", "))
	printIfSet(w, "Started:", date(j.StartedAt))
	printIfSet(w, "Completed:", date(j.CompletedAt))
	printIfSet(w, "URL:", j.HTMLURL)
	_ = w.Flush()
	if len(j.Steps) > 0 {
		fmt.Println()
		t := newTable(table.Row{"#", "Step", "Status", "Duration"})
		for _, s := range j.Steps {
			t.AppendRow(table.Row{s.Number, s.Name, outcome(s.Status, s.Conclusion), elapsed(s.StartedAt, s.CompletedAt)})
		}
		t.Render()
	}
}

func renderArtifactTable(list []gol.WorkflowArtifact) {
	t := newTable(table.Row{"ID", "Name", "Size", "Run", "Created", "Expires"})
	for _, a := range list {
		expires := date(a.ExpiresAt)
		if a.Expired {
			expires = "expired"
		}
		t.AppendRow(table.Row{a.ID, a.Name, size(a.Size), a.RunID, date(a.CreatedAt), expires})
	}
	t.Render()
}

func renderArtifact(a *gol.WorkflowArtifact) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), a.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), a.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Size:"), size(a.Size))
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Run:"), a.RunID)
	printIfSet(w, "Created:", date(a.CreatedAt))
	if a.Expired {
		_, _ = fmt.Fprintf(w, "%s\texpired\n", hftx.Blue("Expires:"))
	} else {
		printIfSet(w, "Expires:", date(a.ExpiresAt))
	}
	_ = w.Flush()
}
