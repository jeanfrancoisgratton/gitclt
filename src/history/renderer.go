// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/renderer.go
// Original timestamp : 2026.10.30 15:10:00

package history

import (
	"fmt"
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

// title is the first line of a commit message
func title(msg string) string {
	t, _, _ := strings.Cut(msg, "\n")
	return t
}

// who names a commit's author or committer, with the forge account matched to the e-mail address, if any
func who(a gol.CommitAuthor) string {
	s := a.Name
	if a.Email != "" {
		s += " <" + a.Email + ">"
	}
	if a.Login != "" && a.Login != a.Name {
		s += " (" + a.Login + ")"
	}
	return s
}

// Merge commits are marked with M in front of their SHA
func renderCommitTable(list []gol.Commit) string {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Commit", "Date", "Author", "Title"})
	for _, c := range list {
		sha := "  " + short(c.SHA)
		if len(c.Parents) > 1 {
			sha = "M " + short(c.SHA)
		}
		t.AppendRow(table.Row{sha, date(c.Author.Date), c.Author.Name, title(c.Message)})
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(88)}
	t.Style().Format.Header = text.FormatDefault
	return t.Render()
}

// renderCommit shows one commit like git show --stat: its metadata, its full message, then its changed files
func renderCommit(c gol.Commit) string {
	parents := make([]string, len(c.Parents))
	for i, p := range c.Parents {
		parents[i] = short(p)
	}

	// tabulated output
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Commit:"), c.SHA)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Author:"), who(c.Author))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Date:"), date(c.Author.Date))
	// Bitbucket does not report the committer; it is only worth showing when it differs from the author
	if c.Committer.Name != "" && (c.Committer.Name != c.Author.Name || c.Committer.Email != c.Author.Email) {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Committer:"), who(c.Committer))
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Commit date:"), date(c.Committer.Date))
	}
	if len(parents) > 0 {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Parents:"), strings.Join(parents, ", "))
	}
	if c.Stats != nil {
		_, _ = fmt.Fprintf(w, "%s\t%d file(s), +%d -%d\n", hftx.Blue("Changes:"), len(c.Files), c.Stats.Additions, c.Stats.Deletions)
	}
	if c.HTMLURL != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("URL:"), c.HTMLURL)
	}
	_ = w.Flush()

	b.WriteString("\n")
	for _, l := range strings.Split(strings.TrimRight(c.Message, "\n"), "\n") {
		b.WriteString("    " + l + "\n")
	}
	if len(c.Files) > 0 {
		b.WriteString("\n" + renderFiles(c.Files))
	}
	return b.String()
}

// Forgejo only reports the commit's totals: the per-file line counts are then all 0, and not worth a column
func renderFiles(files []gol.ChangedFile) string {
	withCounts := false
	for _, f := range files {
		if f.Additions != 0 || f.Deletions != 0 {
			withCounts = true
			break
		}
	}

	t := table.NewWriter()
	header := table.Row{"Status", "File"}
	if withCounts {
		header = append(header, "+", "-")
	}
	t.AppendHeader(header)
	for _, f := range files {
		path := f.Path
		if f.PreviousPath != "" {
			path = f.PreviousPath + " → " + f.Path
		}
		row := table.Row{string(f.Status), path}
		if withCounts {
			row = append(row, f.Additions, f.Deletions)
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(33)}
	t.Style().Format.Header = text.FormatDefault
	return t.Render()
}
