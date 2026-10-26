// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/teams/renderer.go
// Original timestamp : 2026.10.26 08:05:37

package teams

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func renderTeamTable(list []gol.Team) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Organization", "Name", "Description", "Permission", "All repos", "Can create repos"})
	for _, tm := range list {
		t.AppendRow(table.Row{tm.ID, tm.Org, tm.Name, tm.Description, string(tm.Permission),
			tm.IncludesAllRepos, tm.CanCreateOrgRepo})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func printTeam(tm *gol.Team) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), tm.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Organization:"), tm.Org)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), tm.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), tm.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Permission:"), tm.Permission)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Access to all repos:"), tm.IncludesAllRepos)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Can create repos:"), tm.CanCreateOrgRepo)
	if len(tm.Units) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("Units"))
		// Sorted, as map order would change from one run to the next
		units := make([]string, 0, len(tm.Units))
		for u := range tm.Units {
			units = append(units, string(u))
		}
		slices.Sort(units)
		for _, u := range units {
			_, _ = fmt.Fprintf(w, "  %s\t%s\n", strings.TrimPrefix(u, "repo."), tm.Units[gol.TeamUnit(u)])
		}
	}
	_ = w.Flush()
}

func renderRepoTable(list []gol.Repo) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Repository", "Description", "Visibility", "Archived", "URL"})
	for _, r := range list {
		t.AppendRow(table.Row{r.ID, r.FullName, r.Description, string(r.Visibility), r.Archived, r.HTMLURL})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}
