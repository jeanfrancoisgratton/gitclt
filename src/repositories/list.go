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
	"text/tabwriter"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ListRepositories List all repositories that this userpass or token allows.
// The result can be displayed as json, or in a tabulated format. if displayoutput is set to false, this means that
// This function was called from another function, not a user, and we just need its result for internal consumption.
// The result can be filtered (the -o flag) and/or sorted (-s flag)
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
	// Bitbucket repositories have no numeric ID (r.ID is always 0)
	hasIDs := c.Backend() != gol.BackendBitbucket
	// Sorted before any output, so the table, -q and --json all show the same order
	sortRepos(repos, hasIDs)

	if JSONoutput {
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
	// No IDs on Bitbucket: the column is dropped
	header := table.Row{"Owner", "Repository", "Visibility", "Created", "Last updated", "Empty", "URL", "Web site"}
	if hasIDs {
		header = append(table.Row{"ID"}, header...)
	}
	t.AppendHeader(header)

	for _, r := range repos {
		row := table.Row{r.Owner, r.Name, string(r.Visibility), r.CreatedAt.Local().Format("2006.01.02 15:04"),
			r.UpdatedAt.Local().Format("2006.01.02 15:04"), yesNo(r.Empty), r.HTMLURL, r.Website}
		if hasIDs {
			row = append(table.Row{r.ID}, row...)
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return nil, nil
}

// RepositoryInfo provides extended repository information, provided that the userpass / token allows for it.
// The result can be displayed as json, or in a key:value format.
func RepositoryInfo(reponame string, displayoutput bool) (*gol.Repo, *cerr.CustomError) {
	if OwnerOrg == "" {
		return nil, &cerr.CustomError{Title: "Owner / organization required", Message: "You must specify the -o flag for this command"}
	}
	var r *gol.Repo
	var e error
	var c *forge.Client
	var ce *cerr.CustomError

	ctx := context.Background()
	c, ce = forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, ce
	}

	if r, e = c.Repos().Get(ctx, OwnerOrg, reponame); e != nil {
		return nil, forge.Error("Unable to fetch repo", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return r, nil
	}

	if JSONoutput {
		jStream, jerr := json.Marshal(r, jsontext.WithIndent("\t"))
		if jerr != nil {
			return nil, &cerr.CustomError{Title: "Unable to encode the repository info", Message: jerr.Error()}
		}
		if e := hfjson.Print(jStream); e != nil {
			return nil, &cerr.CustomError{Title: "Unable to print the repository info", Message: e.Error()}
		}
		return r, nil
	}

	// pretty output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), r.ID)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Full name:"), r.FullName)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Owner/Org:"), r.Owner)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), r.Name)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), r.Description)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Default branch:"), r.DefaultBranch)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Visibility:"), r.Visibility)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), r.CreatedAt.Local().Format("2006.01.02 15:04"))
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), r.UpdatedAt.Local().Format("2006.01.02 15:04"))
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("HTML URL:"), r.HTMLURL)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("SSH URL:"), r.SSHURL)
	fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Fork:"), r.Fork)
	fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Mirror:"), r.Mirror)
	fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Archived:"), r.Archived)
	fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Template:"), r.Template)
	fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Empty:"), r.Empty)
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Web site:"), r.Website)
	w.Flush()

	return r, nil
}
