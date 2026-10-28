// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/renderer.go
// Original timestamp : 2026.10.26 16:40:12

package branches

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
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
	case b.Ahead > 0 && b.Behind > 0:
		return fmt.Sprintf("ahead %d, behind %d", b.Ahead, b.Behind)
	case b.Ahead > 0:
		return fmt.Sprintf("ahead %d", b.Ahead)
	case b.Behind > 0:
		return fmt.Sprintf("behind %d", b.Behind)
	}
	return "differs"
}

// The server's last commit: its date, and its author
func lastCommit(lc *gol.CommitSummary) string {
	if lc == nil {
		return ""
	}
	return lc.Date.Local().Format("2006.01.02 15:04") + ", " + lc.Author
}

// The columns depend on which sides are listed; the checked out branch is marked with *, like git branch does
func renderBranchTable(list []Branch, withLocal, withServer bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	header := table.Row{"Branch"}
	if withLocal {
		header = append(header, "Local")
	}
	if withServer {
		header = append(header, "Server", "Last commit", "Default", "Protected")
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
			row = append(row, short(b.LocalSHA))
		}
		if withServer {
			def, prot := "", ""
			if b.ServerSHA != "" {
				def, prot = yesNo(b.IsDefault), yesNo(b.Protected)
			}
			row = append(row, short(b.ServerSHA), lastCommit(b.LastCommit), def, prot)
		}
		if withLocal && withServer {
			row = append(row, status(b))
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

// renderBranch shows both sides of one branch, as far as they exist
// noLocal, when set, tells why the local side could not be read (not in a working copy of the repo...)
func renderBranch(b Branch, noLocal string) {
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
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Branch name:"), b.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Local commit:"), local)
	if b.ServerSHA == "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Server commit:"), "not on the server")
	} else {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Server commit:"), short(b.ServerSHA))
		if b.LastCommit != nil {
			title, _, _ := strings.Cut(b.LastCommit.Message, "\n")
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Last commit:"), lastCommit(b.LastCommit))
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Commit title:"), title)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Default:"), yesNo(b.IsDefault))
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Protected:"), yesNo(b.Protected))
	}
	if noLocal == "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Status:"), status(b))
	}
	_ = w.Flush()
}
