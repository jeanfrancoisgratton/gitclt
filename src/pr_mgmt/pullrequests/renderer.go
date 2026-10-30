// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/renderer.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

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

// Commits are shown abbreviated, like git does
func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// branches shows head → base; a head from a fork is prefixed with its repository
func branches(pr gol.PullRequest) string {
	head := pr.Head.Ref
	if pr.Head.Repo != "" && pr.Base.Repo != "" && pr.Head.Repo != pr.Base.Repo {
		head = pr.Head.Repo + ":" + head
	}
	return head + " → " + pr.Base.Ref
}

// mergeable is unknown while the server computes it (GitHub), and never reported by Bitbucket
func mergeable(pr gol.PullRequest) string {
	switch {
	case pr.State != gol.PRStateOpen:
		return ""
	case pr.Mergeable == nil:
		return "unknown"
	case *pr.Mergeable:
		return "yes"
	}
	return "no (conflicts)"
}

func renderPRTable(list []gol.PullRequest) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Title", "Branches", "Author", "Status", "Updated"})
	for _, pr := range list {
		t.AppendRow(table.Row{pr.Number, pr.Title, branches(pr), pr.Author, status(pr), date(pr.UpdatedAt)})
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderPR(pr PullRequest) {
	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t#%d %s\n", hftx.Blue("Pull request:"), pr.Number, pr.Title)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Branches:"), branches(pr.PullRequest))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Head commit:"), short(pr.Head.SHA))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Author:"), pr.Author)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), status(pr.PullRequest))
	if m := mergeable(pr.PullRequest); m != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Mergeable:"), m)
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), date(pr.CreatedAt))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), date(pr.UpdatedAt))
	if pr.Merged {
		_, _ = fmt.Fprintf(w, "%s\t%s, by %s (%s)\n", hftx.Blue("Merged:"), date(pr.MergedAt), pr.MergedBy, short(pr.MergeCommitSHA))
	} else if !pr.ClosedAt.IsZero() {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Closed:"), date(pr.ClosedAt))
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("URL:"), pr.HTMLURL)
	if len(pr.Reviews) == 0 {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Reviews:"), "none")
	}
	for i, r := range pr.Reviews {
		label := ""
		if i == 0 {
			label = "Reviews:"
		}
		line := r.Reviewer + ": " + strings.ReplaceAll(string(r.State), "_", " ")
		if !r.SubmittedAt.IsZero() {
			line += " (" + date(r.SubmittedAt) + ")"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), line)
	}
	_ = w.Flush()

	if b := strings.TrimSpace(pr.Body); b != "" {
		fmt.Println()
		fmt.Println(b)
	}
}
