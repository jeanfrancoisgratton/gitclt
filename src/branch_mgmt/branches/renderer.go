// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/renderer.go
// Original timestamp : 2026.10.26 16:40:12

package branches

import (
	"fmt"
	"gitclt/shared"
	"os"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Commits are shown abbreviated, like git does
func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// status compares both sides of a branch, from the local branch's point of view (like git status)
func status(b Branch) string {
	switch {
	case b.LocalSHA == "":
		return "server only"
	case b.ServerSHA == "":
		return "local only"
	case b.LocalSHA == b.ServerSHA:
		return "in sync"
	case b.Unpushed:
		return "unpushed commits"
	}
	return aheadBehind(b.Ahead, b.Behind)
}

// aheadBehind words a commit count the way git status does
func aheadBehind(ahead, behind int) string {
	switch {
	case ahead > 0 && behind > 0:
		return fmt.Sprintf("ahead %d, behind %d", ahead, behind)
	case ahead > 0:
		return fmt.Sprintf("ahead %d", ahead)
	case behind > 0:
		return fmt.Sprintf("behind %d", behind)
	}
	return "in sync"
}

// The server's last commit: its date, and its author
func lastCommit(lc *gol.CommitSummary) string {
	if lc == nil {
		return ""
	}
	return lc.Date.Local().Format("2006.01.02 15:04") + ", " + lc.Author
}

// parent is the branch it was created from, when the clone's reflogs tell it (a guess does not fit in a column)
func parent(b Branch) string {
	if b.Parent == nil || !b.Parent.Exact {
		return ""
	}
	return b.Parent.Branch
}

// upstream is the branch it tracks, as git names it: remote/branch, or a local branch's own name; gone when the
// last fetch no longer saw it on the remote
func upstream(b Branch) string {
	u := b.Upstream
	if u == nil {
		return ""
	}
	s := u.Remote + "/" + u.Branch
	if u.Remote == "." {
		s = u.Branch
	}
	if u.SHA == "" {
		s += " (gone)"
	}
	return s
}

// vsDefault is how the branch relates to the default one; the default branch itself is not compared
func vsDefault(b Branch) (count, merged string) {
	switch {
	case b.IsDefault:
		return "default", ""
	case b.VsDefault == nil:
		return "", ""
	}
	count = aheadBehind(b.VsDefault.Ahead, b.VsDefault.Behind)
	if b.Merged != nil {
		merged = yesNo(*b.Merged)
	}
	return count, merged
}

func ciState(b Branch) string {
	if b.CI == nil {
		return ""
	}
	return string(b.CI.State)
}

func pullRequestNumbers(list []PullRequest) string {
	n := make([]string, len(list))
	for i, pr := range list {
		n[i] = fmt.Sprintf("#%d", pr.Number)
	}
	return strings.Join(n, ", ")
}

// The columns depend on which sides are listed; the checked out branch is marked with *, like git branch does
// def is the default branch, which the branches are compared with
func renderBranchTable(list []Branch, def string, withLocal, withServer bool) string {
	vs := "Vs default"
	if def != "" {
		vs = "Vs " + def
	}
	t := table.NewWriter()
	header := table.Row{"Branch"}
	if withLocal {
		header = append(header, "Parent", "Upstream", "Local")
	}
	if withServer {
		header = append(header, "Server")
	}
	header = append(header, vs, "Merged")
	if withLocal {
		header = append(header, "Merge base")
	}
	if withServer {
		header = append(header, "CI state")
	}
	header = append(header, "Last commit")
	if withServer {
		header = append(header, "PRs")
	}
	if withLocal && withServer {
		header = append(header, "Status")
	}
	t.AppendHeader(header)

	for _, b := range list {
		name := "  " + b.Name
		if b.Current {
			name = "* " + b.Name
		}
		row := table.Row{name}
		if withLocal {
			row = append(row, parent(b), upstream(b), short(b.LocalSHA))
		}
		if withServer {
			row = append(row, short(b.ServerSHA))
		}
		count, merged := vsDefault(b)
		row = append(row, count, merged)
		if withLocal {
			row = append(row, short(b.MergeBase))
		}
		if withServer {
			row = append(row, ciState(b))
		}
		row = append(row, lastCommit(b.LastCommit))
		if withServer {
			row = append(row, pullRequestNumbers(b.PullRequests))
		}
		if withLocal && withServer {
			row = append(row, status(b))
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	// White on brown
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(52)}
	t.Style().Format.Header = text.FormatDefault
	return t.Render()
}

// parentDetail words where the branch came from: exact from the reflogs (with when), or a guess and its ties
func parentDetail(o *local.Origin) string {
	switch {
	case o == nil:
		return ""
	case o.Exact && o.Branch == "":
		return "a commit (" + short(o.SHA) + ")"
	case o.Exact:
		s := o.Branch + ", at " + short(o.SHA)
		if !o.At.IsZero() {
			s += ", on " + o.At.Local().Format("2006.01.02 15:04")
		}
		return s
	case o.Branch == "":
		return ""
	}
	s := o.Branch + " (a guess from the history"
	if len(o.Others) > 0 {
		s += "; as likely: " + strings.Join(o.Others, ", ")
	}
	return s + ")"
}

// renderBranch shows both sides of one branch, as far as they exist
// noLocal, when set, tells why the local side could not be read (not in a working copy of the repo...); def is the
// default branch, withLocal and withServer the sides asked for
func renderBranch(b Branch, def, noLocal string, withLocal, withServer bool) string {
	local := short(b.LocalSHA)
	switch {
	case noLocal != "":
		local = "n/a (" + noLocal + ")"
	case b.LocalSHA == "":
		local = "not in the local working copy"
	case b.Current:
		local += " (checked out)"
	}

	// tabulated output
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Branch name:"), b.Name)
	if withLocal {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Local commit:"), local)
		printIfSet(w, "Upstream:", upstream(b))
		printIfSet(w, "Created from:", parentDetail(b.Parent))
	}
	if withServer {
		switch {
		case b.ServerSHA == "":
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Server commit:"), "not on the server")
		case b.ServerName != "":
			_, _ = fmt.Fprintf(w, "%s\t%s (as %s)\n", hftx.Blue("Server commit:"), short(b.ServerSHA), b.ServerName)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Server commit:"), short(b.ServerSHA))
		}
	}
	if b.LastCommit != nil {
		title, _, _ := strings.Cut(b.LastCommit.Message, "\n")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Last commit:"), lastCommit(b.LastCommit))
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Commit title:"), title)
	}
	if b.ServerSHA != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Default:"), shared.Sign(b.IsDefault))
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Protected:"), shared.Sign(b.Protected))
	}
	if !b.IsDefault && b.VsDefault != nil {
		label := "Vs default:"
		if def != "" {
			label = "Vs " + def + ":"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), aheadBehind(b.VsDefault.Ahead, b.VsDefault.Behind))
		if b.Merged != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Merged:"), shared.Sign(*b.Merged))
		}
	}
	printIfSet(w, "Merge base:", short(b.MergeBase))
	if b.CI != nil {
		state := string(b.CI.State)
		if state == "" {
			state = "nothing reported"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("CI state:"), state)
		for _, st := range b.CI.Statuses {
			line := st.Context + ": " + string(st.State)
			if st.Description != "" {
				line += ", " + st.Description
			}
			if st.TargetURL != "" {
				line += " (" + st.TargetURL + ")"
			}
			_, _ = fmt.Fprintf(w, "\t%s\n", line)
		}
	}
	if withServer && b.ServerSHA != "" {
		if len(b.PullRequests) == 0 {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Pull requests:"), "none open")
		}
		for i, pr := range b.PullRequests {
			label := ""
			if i == 0 {
				label = "Pull requests:"
			}
			_, _ = fmt.Fprintf(w, "%s\t#%d %s → %s, by %s\n", hftx.Blue(label), pr.Number, pr.Title, pr.Base, pr.Author)
		}
	}
	if withLocal && withServer && noLocal == "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), status(b))
	}
	_ = w.Flush()

	if notes := missingNotes(b.Missing); len(notes) > 0 {
		sb.WriteString("\n" + hftx.Yellow("Could not be read:") + "\n")
		for _, n := range notes {
			sb.WriteString("  - " + n + "\n")
		}
	}
	return sb.String()
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

// renderComparison shows how head relates to base; a head that is only ahead can be merged into base as a fast-forward
func renderComparison(cmp *gol.Comparison) {
	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Base:"), cmp.Base)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Head:"), cmp.Head)
	// From head's point of view, like the status of branch ls is from the local branch's
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), aheadBehind(cmp.Ahead, cmp.Behind))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Fast-forward:"), shared.Sign(cmp.Ahead > 0 && cmp.Behind == 0))
	_ = w.Flush()
}
