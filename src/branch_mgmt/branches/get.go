// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/get.go
// Original timestamp : 2026.10.27 21:50:20

package branches

import (
	"cmp"
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

// GetBranch describes one branch of owner/repo, as seen on the server and, when the current directory is a working copy
// of the repo, locally as well (the default; -l or -r keep one side); a branch that exists on only one side is still
// shown. What could not be read is listed, not an error
// The clone's parts cost nothing; on the server, it is a handful of requests for this branch alone (branchinfo, which
// branch ls uses, reads every branch's)
func GetBranch(repo, branch string, displayoutput bool) (*Branch, *cerr.CustomError) {
	withLocal, withServer := !RemoteOnly, !LocalOnly
	if LocalOnly && RemoteOnly {
		withLocal, withServer = true, true
	}

	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()
	owner := OwnerOrg
	def := ""
	if withServer {
		if c, owner, repo, ce = connect(ctx, repo); ce != nil {
			return nil, ce
		}
		// The default branch is what the branch is compared with; a 404 further down is then about the branch
		r, e := c.Repos().Get(ctx, owner, repo)
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		if e != nil {
			return nil, forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
		}
		def = r.DefaultBranch
	}
	b := &Branch{Name: branch}
	var m missing

	// Not being in a working copy of the repo only drops the local side, unless it is the only one asked for
	noLocal := ""
	remote := ""
	if withLocal {
		clone, lce := openClone(owner, repo)
		if lce != nil {
			if !withServer {
				return nil, lce
			}
			noLocal = lce.Message
		} else {
			defer clone.Close()
			remote = cloneRemote(c, clone)
			if def == "" {
				def, _ = clone.DefaultBranch(cmp.Or(remote, "origin"))
			}
			if e := readLocal(clone, b, def, remote, &m); e != nil {
				return nil, &cerr.CustomError{Title: "Unable to read the local branch " + branch, Message: e.Error()}
			}
		}
	}
	b.IsDefault = def != "" && branch == def

	if withServer {
		if b.LocalSHA != "" {
			if name := serverName(*b, remote); name != branch {
				b.ServerName = name
			}
		}
		if ce = readServer(ctx, c, owner, repo, def, b, &m); ce != nil {
			return nil, ce
		}
	}
	if b.LocalSHA == "" && b.ServerSHA == "" {
		where := "neither on the server nor in the local working copy"
		switch {
		case !withServer:
			where = "not in the local working copy"
		case !withLocal || noLocal != "":
			where = "not on the server"
		}
		return nil, &cerr.CustomError{Title: "No such branch", Code: http.StatusNotFound, Message: branch + " is " + where}
	}
	if withLocal && withServer && noLocal == "" {
		if ce = compareSides(ctx, c, owner, repo, b); ce != nil {
			return nil, ce
		}
	}
	b.Missing = m

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return b, nil
	}
	if JSONoutput {
		return b, shared.PrintJSON(b, "branch info")
	}
	// The branch's line stays on top
	shared.Page(renderBranch(*b, def, noLocal, withLocal, withServer), 1)
	return b, nil
}

// readLocal fills what the clone tells of the branch, if it has it: its head, what it tracks, where it was created
// from (the reflogs, else a guess), its last commit, and how it relates to the default branch
func readLocal(clone *local.Repo, b *Branch, def, remote string, m *missing) error {
	branches, e := clone.Branches()
	if e != nil {
		return e
	}
	var lb *local.Branch
	for i := range branches {
		if branches[i].Name == b.Name {
			lb = &branches[i]
		}
	}
	if lb == nil {
		return nil
	}
	b.LocalSHA, b.Current, b.Upstream = lb.SHA, lb.Current, lb.Upstream

	commit, e := clone.Commit(lb.SHA)
	if e = m.note("local last commit", e); e != nil {
		return e
	}
	if commit != nil {
		b.LastCommit = commit.Summary()
	}

	origin, e := clone.CreatedFrom(b.Name)
	if e != nil {
		return e
	}
	if origin == nil {
		origin, e = clone.GuessParent(b.Name, nil)
		if e = m.note("local parent", e); e != nil {
			return e
		}
	}
	b.Parent = origin

	if b.Name == def {
		return nil
	}
	defRef := localDefault(clone, def, remote)
	if defRef == "" {
		if def != "" {
			*m = append(*m, gol.PartError{Part: "local ahead/behind", Message: fmt.Sprintf("no default branch %q in the clone", def)})
		}
		return nil
	}
	cmpr, e := clone.Compare(defRef, lb.SHA)
	if e = m.note("local ahead/behind", e); e != nil {
		return e
	}
	if cmpr != nil {
		cmpr.Head = b.Name
		b.VsDefault, b.Merged = cmpr, gol.Ptr(cmpr.Ahead == 0)
	}
	b.MergeBase = mergeBase(clone, defRef, lb.SHA)
	return nil
}

// readServer fills what the server tells of the branch, if it has it: its head and last commit, protection, CI state,
// open pull requests, and, for a branch the clone does not have, how it relates to the default branch
func readServer(ctx context.Context, c *forge.Client, owner, repo, def string, b *Branch, m *missing) *cerr.CustomError {
	name := cmp.Or(b.ServerName, b.Name)
	sb, e := c.Branches().Get(ctx, owner, repo, name)
	switch {
	case gol.IsNotFound(e):
		b.ServerName = ""
		return nil
	case e != nil:
		return forge.Error("Unable to get the branch "+name, e)
	}
	b.ServerSHA, b.Protected = sb.SHA, sb.Protected
	b.IsDefault = b.IsDefault || sb.IsDefault
	if sb.LastCommit != nil {
		b.LastCommit = sb.LastCommit
	}

	// The clone's count wins, as the server only knows what was pushed
	if b.VsDefault == nil && !b.IsDefault && def != "" {
		cmpr, e := c.Branches().Compare(ctx, owner, repo, def, name)
		if e = m.note("ahead/behind", e); e != nil {
			return forge.Error(fmt.Sprintf("Unable to compare %s with %s", name, def), e)
		}
		if cmpr != nil {
			b.VsDefault, b.Merged = cmpr, gol.Ptr(cmpr.Ahead == 0)
		}
	}

	st, e := c.Commits().Status(ctx, owner, repo, name)
	if e = m.note("CI state", e); e != nil {
		return forge.Error("Unable to get the CI state of "+name, e)
	}
	b.CI = st

	// All the open ones, as branchinfo reads them: filtering by branch needs Forgejo 16; those from forks are left out
	prs, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.PullRequest], error) {
		return c.PullRequests().List(ctx, owner, repo, gol.ListPROptions{ListOptions: o})
	})
	if e = m.note("pull requests", e); e != nil {
		return forge.Error("Unable to list the pull requests from "+name, e)
	}
	full := owner + "/" + repo
	for _, pr := range prs {
		if pr.Head.Ref == name && (pr.Head.Repo == "" || strings.EqualFold(pr.Head.Repo, full)) {
			b.PullRequests = append(b.PullRequests, pullRequest(pr))
		}
	}
	return nil
}
