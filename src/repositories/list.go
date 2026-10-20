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
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// fetchRepos collects every page of repositories for owner.
// An empty owner lists every repository the authenticated user can access (including their orgs' repositories),
// otherwise the forge only returns that user's or organization's repositories (the filter is applied server-side)
func fetchRepos(ctx context.Context, c *forge.Client, owner string) ([]gol.Repo, error) {
	var repos []gol.Repo
	fetch := func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return c.Repos().List(ctx, owner, o)
	}
	for r, err := range gol.All(ctx, gol.ListOptions{PerPage: listPageSize}, fetch) {
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func ListRepositories(displayoutput bool) ([]gol.Repo, *cerr.CustomError) {
	ctx := context.Background()
	// An empty name means: use the default connection
	c, err := forge.Connect(ctx, shared.ConnectionName)
	if err != nil {
		return nil, err
	}

	repos, lerr := fetchRepos(ctx, c, OwnerOrg)
	// An unknown owner falls back to the default behaviour: list everything
	if lerr != nil && OwnerOrg != "" && gol.IsNotFound(lerr) {
		fmt.Fprintf(os.Stderr, "%s: no such user or organization, listing all repositories\n", hftx.Yellow(OwnerOrg))
		repos, lerr = fetchRepos(ctx, c, "")
	}
	if lerr != nil {
		return nil, forge.Error("Unable to list repositories", lerr)
	}

	if RepoListJSONOutput {
		jStream, jerr := json.Marshal(repos, jsontext.WithIndent("\t"))
		if jerr != nil {
			return nil, &cerr.CustomError{Title: "Unable to encode the repository list", Message: jerr.Error()}
		}
		if e := hfjson.Print(jStream); e != nil {
			return nil, &cerr.CustomError{Title: "Unable to print the repository list", Message: e.Error()}
		}
		return nil, nil
	}

	if shared.QuietOutput {
		for _, r := range repos {
			fmt.Println(r.FullName)
		}
		return nil, nil
	}

	if !displayoutput {
		return repos, nil
	}
	fmt.Printf("Number of repositories on %s: %s\n", hftx.Green(c.Connection.Name), hftx.Green(fmt.Sprintf("%d", len(repos))))
	if len(repos) == 0 {
		return nil, nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Owner", "Repository", "Visibility", "Created", "Last updated", "Empty", "URL", "Web site"})

	for _, r := range repos {
		t.AppendRow(table.Row{r.ID, r.Owner, r.Name, string(r.Visibility), r.CreatedAt.Local().Format("2006.01.02 15:04"),
			r.UpdatedAt.Local().Format("2006.01.02 15:04"), yesNo(r.Empty), r.HTMLURL, r.Website})
	}
	// determine the sort order
	sb, mode := "", table.Asc
	switch strings.TrimSpace(strings.ToLower(SortBy)) {
	case "owner", "organization", "org", "o":
		sb = "Owner"
	case "repository", "repo", "r":
		sb = "Repository"
	case "created", "creation", "c":
		sb = "Created"
	case "updated", "update", "u", "last", "l":
		sb = "Last updated"
	default:
		// table.Asc compares the rendered cells as strings (1, 10, 2...); IDs need a numeric sort
		sb, mode = "ID", table.AscNumeric
	}
	t.SortBy([]table.SortBy{
		{Name: sb, Mode: mode},
	})
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return nil, nil
}
