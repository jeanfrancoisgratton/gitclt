// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/list.go
// Original timestamp : 2026.10.26 16:40:12

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/branchinfo"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListBranches lists the branches of owner/reponame: those of the local working copy (the current directory),
// those on the server, or both (the default), paired by the branch each local one tracks, else by name
// Without a repo name, it is the one the current directory is a clone of (see forge.Client.Target)
// --local alone needs neither the owner nor the repo name; when they are given, the working copy must be that repo's
// With both, a current directory that is not a working copy of the repo only drops the local side, with a notice
// The details come from the library's branchinfo: the clone's parts cost nothing; on the server, the comparisons with
// the default branch and the CI states are requests per branch, the pull requests one list
func ListBranches(reponame string, displayoutput bool) ([]Branch, *cerr.CustomError) {
	withLocal, withServer := !RemoteOnly, !LocalOnly
	if AllBranches || (LocalOnly && RemoteOnly) {
		withLocal, withServer = true, true
	}

	// The owner is only known once connected (see forge.Client.Owner); --local alone only checks an explicit -o
	var c *forge.Client
	owner := OwnerOrg
	ctx := context.Background()
	if withServer {
		var ce *cerr.CustomError
		if c, owner, reponame, ce = connect(ctx, reponame); ce != nil {
			return nil, ce
		}
	}

	// A notice goes above the table, and stays on top with its header when paged
	notice := ""
	var clone *local.Repo
	if withLocal {
		repo, ce := openClone(owner, reponame)
		if ce != nil {
			if !withServer {
				return nil, ce
			}
			// --all: the server's side is still worth showing
			withLocal = false
			if displayoutput && !shared.QuietOutput && !JSONoutput {
				if !SuppressWarnings {
					notice = hftx.InfoSign(ce.Message+"; showing the server's branches only") + "\n"
				} else {
					notice = hftx.InfoSign(" We are not in the local repo so we can only show the server's branches") + "\n"
				}
			}
		} else {
			clone = repo
			defer clone.Close()
		}
	}

	opts := branchinfo.Options{Local: clone, Owner: owner, Repo: reponame, Remote: cloneRemote(c, clone)}
	if withLocal {
		opts.AheadBehind, opts.LastCommit, opts.Parent = branchinfo.Local, branchinfo.Local, branchinfo.Local
	}
	if withServer {
		opts.Client = c.Client
		opts.AheadBehind |= branchinfo.Server
		opts.LastCommit |= branchinfo.Server
		opts.PullRequests, opts.Statuses = true, true
	}
	res, e := branchinfo.Collect(ctx, opts)
	if e != nil {
		if !withServer {
			return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
		}
		return nil, forge.Error(fmt.Sprintf("Unable to fetch the branches of %s/%s", owner, reponame), e)
	}

	defRef := ""
	if withLocal {
		defRef = localDefault(clone, res.Default, opts.Remote)
	}
	branches := make([]Branch, 0, len(res.Branches))
	for _, ib := range res.Branches {
		b := fromInfo(ib)
		if withLocal && !b.IsDefault {
			b.MergeBase = mergeBase(clone, defRef, b.LocalSHA)
		}
		// Where both sides differ, the server counts the commits between its branch and the local head
		// (only the differing branches cost a comparison)
		if withLocal && withServer {
			if ce := compareSides(ctx, c, owner, reponame, &b); ce != nil {
				return nil, ce
			}
		}
		branches = append(branches, b)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return branches, nil
	}
	notes := missingNotes(res.Missing)
	if JSONoutput {
		// The notes would break the JSON: they go to the standard error
		for _, n := range notes {
			fmt.Fprintln(os.Stderr, hftx.InfoSign("Not shown: "+n))
		}
		return branches, shared.PrintJSON(branches, "branch list")
	}
	banner := shared.TableHeader
	if notice != "" {
		banner++
	}
	out := notice + renderBranchTable(branches, res.Default, withLocal, withServer)
	if len(notes) > 0 && !SuppressWarnings {
		out += "\n"
		for _, n := range notes {
			out += "\n" + hftx.InfoSign("Not shown: "+n)
		}
	}
	shared.Page(out, banner)
	return branches, nil
}

// connect opens the client on the selected connection, and decides the repository and its owner (see forge.Client.Target)
func connect(ctx context.Context, repo string) (*forge.Client, string, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", "", ce
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return nil, "", "", ce
	}
	return c, owner, repo, nil
}

// compareSides fills how the local branch relates to the server's, when both exist and differ
// The local head is unknown to the server (404) when it has unpushed commits
func compareSides(ctx context.Context, c *forge.Client, owner, reponame string, b *Branch) *cerr.CustomError {
	if b.LocalSHA == "" || b.ServerSHA == "" || b.LocalSHA == b.ServerSHA {
		return nil
	}
	name := b.Name
	if b.ServerName != "" {
		name = b.ServerName
	}
	cmp, e := c.Branches().Compare(ctx, owner, reponame, name, b.LocalSHA)
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
