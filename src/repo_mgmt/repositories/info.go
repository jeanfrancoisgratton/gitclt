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
	"io"
	"os"
	"slices"
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
	owner, reponame, ce := c.Target(ctx, OwnerOrg, reponame)
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
	printSettings(w, r)
	_ = w.Flush()

	return r, nil
}

// infoRow is a feature or setting; an empty val is one the forge lacks or does not report, shown as n/a, never as off
type infoRow struct {
	label, val string
}

func boolVal(b *bool) string {
	if b == nil {
		return ""
	}
	return yesNo(*b)
}

func strVal[T ~string](s *T, empty string) string {
	switch {
	case s == nil:
		return ""
	case *s == "":
		return empty
	}
	return string(*s)
}

// printGroup prints a group of features or settings, unless the forge reports none of them
// It returns false when nothing was printed
func printGroup(w io.Writer, title string, rows []infoRow) bool {
	if !slices.ContainsFunc(rows, func(r infoRow) bool { return r.val != "" }) {
		return false
	}
	_, _ = fmt.Fprintf(w, "%s\t\n", hftx.Blue(title))
	for _, r := range rows {
		val := r.val
		if val == "" {
			val = "n/a"
		}
		// Coloured like the other labels: tabwriter counts the escape codes, so all labels need the same
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("  "+r.label+":"), val)
	}
	return true
}

// printSettings shows the features, then the settings groups the forge has (Bitbucket has none)
func printSettings(w io.Writer, r *gol.Repo) {
	f := r.Features
	if !printGroup(w, "Features:", []infoRow{
		{"Actions", boolVal(f.Actions)}, {"Issues", boolVal(f.Issues)}, {"Wiki", boolVal(f.Wiki)},
		{"Pull requests", boolVal(f.PullRequests)}, {"Projects", boolVal(f.Projects)}, {"Releases", boolVal(f.Releases)},
		{"Packages", boolVal(f.Packages)}, {"Discussions", boolVal(f.Discussions)},
	}) {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Features:"), "not reported by this forge")
	}

	m := r.Merge
	printGroup(w, "Merge:", []infoRow{
		{"Merge commit", boolVal(m.MergeCommit)}, {"Rebase", boolVal(m.Rebase)}, {"Rebase then merge", boolVal(m.RebaseMerge)},
		{"Squash", boolVal(m.Squash)}, {"Fast-forward only", boolVal(m.FastForwardOnly)},
		{"Default method", strVal(m.DefaultMethod, "none")}, {"Delete branch", boolVal(m.DeleteBranch)},
		{"Auto-merge", boolVal(m.AutoMerge)}, {"Update branch", boolVal(m.UpdateBranch)}, {"Rebase update", boolVal(m.RebaseUpdate)},
	})

	t := r.IssueTracker
	external := ""
	if t.External != nil {
		// A zero URL is the built-in tracker
		external = "none (built-in)"
		if t.External.URL != "" {
			external = t.External.URL
			if t.External.Style != "" {
				external += " (" + string(t.External.Style) + ")"
			}
		}
	}
	printGroup(w, "Issue tracker:", []infoRow{
		{"Time tracking", boolVal(t.TimeTracking)}, {"Contributors only", boolVal(t.TimeTrackingContributorsOnly)},
		{"Dependencies", boolVal(t.Dependencies)}, {"External tracker", external},
	})

	printGroup(w, "Wiki:", []infoRow{
		{"External wiki", strVal(r.Wiki.ExternalURL, "none (built-in)")}, {"Editable by all", boolVal(r.Wiki.EditableByAll)},
	})

	a := r.Actions
	printGroup(w, "Actions:", []infoRow{
		{"Allowed", strVal(a.Allowed, "")}, {"Token permission", strVal(a.TokenPermission, "")},
		{"Approve pull requests", boolVal(a.ApprovePullRequests)},
	})
}
