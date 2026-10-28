// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/list.go
// Original timestamp : 2026.10.26 16:40:12

package branches

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListBranches lists the branches of OwnerOrg/reponame: those of the local working copy (the current directory),
// those on the server, or both, merged by name (the default)
// --local alone needs neither the owner nor the repo name; when they are given, the working copy must be that repo's
// With both, a current directory that is not a working copy of the repo only drops the local side, with a notice
func ListBranches(reponame string, displayoutput bool) ([]Branch, *cerr.CustomError) {
	withLocal, withServer := !RemoteOnly, !LocalOnly
	if AllBranches || (LocalOnly && RemoteOnly) {
		withLocal, withServer = true, true
	}

	if withServer && (OwnerOrg == "" || reponame == "") {
		return nil, &cerr.CustomError{Title: "Missing repository",
			Message: "the server's branches need the owner (-o OWNER) and the repository name; use --local for the local branches only"}
	}

	rows := map[string]*Branch{}
	row := func(name string) *Branch {
		if rows[name] == nil {
			rows[name] = &Branch{Name: name}
		}
		return rows[name]
	}

	if withLocal {
		lr, ce := localBranches(reponame)
		if ce != nil {
			if !withServer {
				return nil, ce
			}
			// --all: the server's side is still worth showing
			withLocal = false
			if displayoutput && !shared.QuietOutput && !JSONoutput {
				if !SuppressWarnings {
					fmt.Println(hftx.InfoSign(ce.Message + "; showing the server's branches only"))
				} else {
					fmt.Println(hftx.InfoSign(" We are not in the local repo so we can only shot the server's branches"))
				}
			}
		} else {
			for name, sha := range lr.branches {
				b := row(name)
				b.LocalSHA, b.Current = sha, name == lr.current
			}
		}
	}

	var c *forge.Client
	ctx := context.Background()
	if withServer {
		var ce *cerr.CustomError
		if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
			return nil, ce
		}
		list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Branch], error) {
			return c.Branches().List(ctx, OwnerOrg, reponame, o)
		})
		if e != nil {
			return nil, forge.Error(fmt.Sprintf("Unable to fetch the branches of %s/%s", OwnerOrg, reponame), e)
		}
		for _, sb := range list {
			b := row(sb.Name)
			b.ServerSHA, b.IsDefault, b.Protected, b.LastCommit = sb.SHA, sb.IsDefault, sb.Protected, sb.LastCommit
		}
	}

	// Where both sides differ, the server counts the commits between its branch and the local head
	// (only the differing branches cost a comparison)
	if withLocal && withServer {
		for _, b := range rows {
			if ce := compareSides(ctx, c, reponame, b); ce != nil {
				return nil, ce
			}
		}
	}

	branches := make([]Branch, 0, len(rows))
	for _, b := range rows {
		branches = append(branches, *b)
	}
	slices.SortFunc(branches, func(a, b Branch) int { return strings.Compare(a.Name, b.Name) })

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return branches, nil
	}
	if JSONoutput {
		return branches, shared.PrintJSON(branches, "branch list")
	}
	renderBranchTable(branches, withLocal, withServer)
	return branches, nil
}

// compareSides fills how the local branch relates to the server's, when both exist and differ
// The local head is unknown to the server (404) when it has unpushed commits
func compareSides(ctx context.Context, c *forge.Client, reponame string, b *Branch) *cerr.CustomError {
	if b.LocalSHA == "" || b.ServerSHA == "" || b.LocalSHA == b.ServerSHA {
		return nil
	}
	cmp, e := c.Branches().Compare(ctx, OwnerOrg, reponame, b.Name, b.LocalSHA)
	switch {
	case gol.IsNotFound(e):
		b.Unpushed = true
	case e != nil:
		return forge.Error(fmt.Sprintf("Unable to compare branch %s with the server's", b.Name), e)
	default:
		b.Ahead, b.Behind = cmp.Ahead, cmp.Behind
	}
	return nil
}

// localBranches reads the working copy of the current directory, and makes sure it is OwnerOrg/reponame's
func localBranches(reponame string) (*localRepo, *cerr.CustomError) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
	}
	lr, e := readLocalRepo(cwd)
	if errors.Is(e, errNoWorkingCopy) {
		return nil, &cerr.CustomError{Title: "No local working copy", Message: cwd + " is " + e.Error()}
	}
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
	}
	// Without -o (--local only), only the repo name part of the remotes' URLs is checked
	if reponame != "" && !lr.matchesRepo(OwnerOrg, reponame) {
		return nil, &cerr.CustomError{Title: "Not the same repository",
			Message: fmt.Sprintf("the working copy in %s is not a clone of %s (its remotes: %s)",
				cwd, strings.TrimPrefix(OwnerOrg+"/"+reponame, "/"), strings.Join(lr.remotes, ", "))}
	}
	return lr, nil
}
