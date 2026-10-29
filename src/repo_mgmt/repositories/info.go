// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/info.go
// Original timestamp : 2026.10.21 00:13:23

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
)

// RepositoryInfo provides extended repository information, provided that the userpass / token allows for it.
// The result can be displayed as json, or in a key:value format.
func RepositoryInfo(reponame string, displayoutput bool) (*gol.Repo, *cerr.CustomError) {
	var r *gol.Repo
	var e error
	var c *forge.Client
	var ce *cerr.CustomError

	ctx := context.Background()
	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return nil, ce
	}

	if r, e = c.Repos().Get(ctx, owner, reponame); e != nil {
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

	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), r.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Full name:"), r.FullName)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Owner/Org:"), r.Owner)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), r.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), r.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Default branch:"), r.DefaultBranch)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Visibility:"), r.Visibility)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), r.CreatedAt.Local().Format("2006.01.02 15:04"))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), r.UpdatedAt.Local().Format("2006.01.02 15:04"))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("HTML URL:"), r.HTMLURL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("SSH URL:"), r.SSHURL)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Fork:"), r.Fork)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Mirror:"), r.Mirror)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Archived:"), r.Archived)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Template:"), r.Template)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Empty:"), r.Empty)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Web site:"), r.Website)
	_ = w.Flush()

	return r, nil
}
