// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/list.go
// Original timestamp : 2026.10.18 00:53:36

package repositories

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func yesNo(b bool) string {
	if b {
		return hftx.Yellow("yes")
	}
	return hftx.Green("no")
}

func ListRepositories() *cerr.CustomError {
	ctx := context.Background()
	// An empty name means: use the default connection
	c, err := forge.Connect(ctx, shared.ConnectionName)
	if err != nil {
		return err
	}

	// An empty owner lists every repository the authenticated user can access (including their orgs' repositories)
	var repos []gol.Repo
	fetch := func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return c.Repos().List(ctx, "", o)
	}
	for r, lerr := range gol.All(ctx, gol.ListOptions{PerPage: listPageSize}, fetch) {
		if lerr != nil {
			return forge.Error("Unable to list repositories", lerr)
		}
		repos = append(repos, r)
	}

	if RepoListJSONOutput {
		jStream, jerr := json.Marshal(repos, jsontext.WithIndent("\t"))
		if jerr != nil {
			return &cerr.CustomError{Title: "Unable to encode the repository list", Message: jerr.Error()}
		}
		fmt.Println(string(jStream))
		return nil
	}

	if shared.QuietOutput {
		for _, r := range repos {
			fmt.Println(r.FullName)
		}
		return nil
	}

	fmt.Printf("Number of repositories on %s: %s\n", hftx.Green(c.Connection.Name), hftx.Green(fmt.Sprintf("%d", len(repos))))
	if len(repos) == 0 {
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Repository", "Visibility", "Fork", "Mirror", "Archived", "Default branch", "Last update", "URL"})

	for _, r := range repos {
		t.AppendRow(table.Row{hftx.Green(r.FullName), hftx.Green(string(r.Visibility)), yesNo(r.Fork), yesNo(r.Mirror), yesNo(r.Archived),
			hftx.Green(r.DefaultBranch), hftx.Green(r.UpdatedAt.Local().Format("2006.01.02 15:04")), hftx.Green(r.HTMLURL)})
	}
	t.SortBy([]table.SortBy{
		{Name: "Repository", Mode: table.Asc},
	})
	t.SetStyle(table.StyleBold)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return nil
}
