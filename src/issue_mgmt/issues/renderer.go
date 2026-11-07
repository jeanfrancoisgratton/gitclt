// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/renderer.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"fmt"
	"os"
	"slices"
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

// A due date is a day: GitHub keeps only that, as midnight UTC, which would read as the day before anywhere west of Greenwich
func day(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	if u := t.UTC(); u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 {
		return u.Format("2006.01.02")
	}
	return t.Local().Format("2006.01.02")
}

func milestoneTitle(i gol.Issue) string {
	if i.Milestone == nil {
		return ""
	}
	return i.Milestone.Title
}

func newTable(header table.Row) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(header)
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(22)}
	t.Style().Format.Header = text.FormatDefault
	return t
}

func renderIssueTable(list []gol.Issue) {
	t := newTable(table.Row{"#", "Title", "State", "Author", "Labels", "Milestone", "Assignees", "Comments", "Updated"})
	for _, i := range list {
		t.AppendRow(table.Row{i.Number, i.Title, string(i.State), i.Author, strings.Join(i.Labels, ", "), milestoneTitle(i),
			strings.Join(i.Assignees, ", "), i.Comments, date(i.UpdatedAt)})
	}
	t.Render()
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

func renderIssue(i *Issue) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t#%d %s\n", hftx.Blue("Issue:"), i.Number, i.Title)
	state := string(i.State)
	if i.Locked {
		state += ", locked"
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("State:"), state)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Author:"), i.Author)
	printIfSet(w, "Labels:", strings.Join(i.Labels, ", "))
	printIfSet(w, "Milestone:", milestoneTitle(i.Issue))
	printIfSet(w, "Assignees:", strings.Join(i.Assignees, ", "))
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Comments:"), i.Issue.Comments)
	printIfSet(w, "Created:", date(i.CreatedAt))
	printIfSet(w, "Updated:", date(i.UpdatedAt))
	printIfSet(w, "Closed:", date(i.ClosedAt))
	printIfSet(w, "URL:", i.HTMLURL)
	_ = w.Flush()
	if i.Body != "" {
		fmt.Println()
		fmt.Println(i.Body)
	}
	if len(i.Discussion) > 0 {
		fmt.Println()
		renderComments(i.Discussion)
	}
}

// A comment is free text, often several lines long: a table would not suit it
func renderComments(list []gol.Comment) {
	for n, m := range list {
		if n > 0 {
			fmt.Println()
		}
		fmt.Printf("%s %d, %s, %s\n", hftx.Blue("Comment"), m.ID, m.Author, date(m.CreatedAt))
		fmt.Println(m.Body)
	}
}

// A label's color can be seen, not only read: the closest one of the 256-color palette, as go-pretty's own colors are
// (they turn themselves off where colors are not wanted)
func swatch(hex string) string {
	var r, g, b int
	if _, e := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); e != nil {
		return "  "
	}
	// The palette's color cube has six levels per channel, starting at 16
	level := func(v int) int { return (v*5 + 127) / 255 }
	return text.Colors{text.Bg256Color(16 + 36*level(r) + 6*level(g) + level(b))}.Sprint("  ")
}

// The scope column only appears when an organization's labels are listed too
func renderLabelTable(list []Label) {
	hasScope := slices.ContainsFunc(list, func(l Label) bool { return l.Scope != "repo" })
	header := table.Row{"ID", "Name", "Color", "Description"}
	if hasScope {
		header = table.Row{"ID", "Name", "Color", "Scope", "Description"}
	}
	t := newTable(header)
	for _, l := range list {
		row := table.Row{l.ID, l.Name, swatch(l.Color) + " #" + l.Color}
		if hasScope {
			scope := l.Scope
			if scope != "repo" {
				scope = "org " + scope
			}
			if l.Hidden {
				scope += " (hidden by the repo's)"
			}
			row = append(row, scope)
		}
		t.AppendRow(append(row, l.Description))
	}
	t.Render()
}

func renderLabel(l *gol.Label) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), l.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), l.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s #%s\n", hftx.Blue("Color:"), swatch(l.Color), l.Color)
	printIfSet(w, "Description:", l.Description)
	_ = w.Flush()
}

func renderMilestoneTable(list []gol.Milestone) {
	t := newTable(table.Row{"ID", "Title", "State", "Due", "Open", "Closed", "Description"})
	for _, m := range list {
		t.AppendRow(table.Row{m.ID, m.Title, string(m.State), day(m.DueOn), m.OpenIssues, m.ClosedIssues, m.Description})
	}
	t.Render()
}

func renderMilestone(m *gol.Milestone) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), m.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Title:"), m.Title)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("State:"), m.State)
	printIfSet(w, "Description:", m.Description)
	printIfSet(w, "Due:", day(m.DueOn))
	_, _ = fmt.Fprintf(w, "%s\t%d open, %d closed\n", hftx.Blue("Issues:"), m.OpenIssues, m.ClosedIssues)
	printIfSet(w, "Closed:", date(m.ClosedAt))
	_ = w.Flush()
}
