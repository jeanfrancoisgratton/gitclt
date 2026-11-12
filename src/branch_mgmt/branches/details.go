// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/details.go
// Original timestamp : 2026.11.12 10:00:00

package branches

import (
	"errors"
	"fmt"
	"gitclt/forge"
	"os"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/branchinfo"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

// openClone opens the working copy of the current directory, and makes sure it is owner/reponame's; the caller closes it
// Without a repo name (--local alone), any working copy does
func openClone(owner, reponame string) (*local.Repo, *cerr.CustomError) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
	}
	repo, e := local.Open(cwd)
	if errors.Is(e, local.ErrNoWorkingCopy) {
		return nil, &cerr.CustomError{Title: "No local working copy", Message: cwd + " is " + e.Error()}
	}
	if e != nil {
		return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
	}
	if reponame == "" {
		return repo, nil
	}
	ok, e := repo.MatchesRepo(owner, reponame)
	if e == nil && !ok {
		remotes, _ := repo.Remotes()
		urls := make([]string, len(remotes))
		for i, r := range remotes {
			urls[i] = r.URL
		}
		e = fmt.Errorf("the working copy in %s is not a clone of %s (its remotes: %s)",
			cwd, strings.TrimPrefix(owner+"/"+reponame, "/"), strings.Join(urls, ", "))
		_ = repo.Close()
		return nil, &cerr.CustomError{Title: "Not the same repository", Message: e.Error()}
	}
	if e != nil {
		_ = repo.Close()
		return nil, &cerr.CustomError{Title: "Unable to read the local branches", Message: e.Error()}
	}
	return repo, nil
}

// cloneRemote is the clone's remote on the connection's server, which pairs the local branches with the server's
// through what they track; "" (origin) without a server, or when none of the remotes is on it
func cloneRemote(c *forge.Client, repo *local.Repo) string {
	if c == nil || repo == nil {
		return ""
	}
	if rr, e := repo.RepoOn(c.Connection.URL); e == nil {
		return rr.Remote
	}
	return ""
}

// localDefault is the ref the clone compares with: the local default branch, or else what the last fetch saw of the
// remote's; "" when the clone has neither
func localDefault(repo *local.Repo, def, remote string) string {
	if def == "" {
		return ""
	}
	if remote == "" {
		remote = "origin"
	}
	if bs, e := repo.Branches(); e == nil && slices.ContainsFunc(bs, func(b local.Branch) bool { return b.Name == def }) {
		return def
	}
	if rb, e := repo.RemoteBranches(); e == nil {
		if _, ok := rb[remote+"/"+def]; ok {
			return remote + "/" + def
		}
	}
	return ""
}

// mergeBase is the newest commit that the branch's head and the default branch share, read in the clone; "" when
// the clone cannot tell (a missing object, unrelated histories), as the ahead/behind count then says why
func mergeBase(repo *local.Repo, defRef, sha string) string {
	if defRef == "" || sha == "" {
		return ""
	}
	if mb, e := repo.MergeBase(defRef, sha); e == nil && mb != nil {
		return mb.SHA
	}
	return ""
}

// fromInfo turns a branch of branchinfo.Collect into a row: the local side's count against the default branch wins,
// as the server only knows what was pushed
func fromInfo(ib branchinfo.Branch) Branch {
	b := Branch{Name: ib.Name, IsDefault: ib.Default, Parent: ib.Origin, CI: ib.Status}
	if ib.Local != nil {
		b.LocalSHA, b.Current, b.Upstream = ib.Local.SHA, ib.Local.Current, ib.Local.Upstream
	}
	if ib.Server != nil {
		b.ServerSHA, b.Protected, b.LastCommit = ib.Server.SHA, ib.Server.Protected, ib.Server.LastCommit
		if ib.Server.Name != ib.Name {
			b.ServerName = ib.Server.Name
		}
	}
	if b.LastCommit == nil && ib.LocalCommit != nil {
		b.LastCommit = ib.LocalCommit.Summary()
	}
	switch {
	case ib.LocalAhead != nil:
		b.VsDefault, b.Merged = ib.LocalAhead, ib.LocalMerged
	case ib.ServerAhead != nil:
		b.VsDefault, b.Merged = ib.ServerAhead, ib.ServerMerged
	}
	for _, pr := range ib.PullRequests {
		b.PullRequests = append(b.PullRequests, pullRequest(pr))
	}
	return b
}

// serverName is the branch's name on the server: the one it tracks on the clone's remote, else its own
func serverName(b Branch, remote string) string {
	if remote == "" {
		remote = "origin"
	}
	if b.ServerName != "" {
		return b.ServerName
	}
	if b.Upstream != nil && b.Upstream.Remote == remote {
		return b.Upstream.Branch
	}
	return b.Name
}

// missing collects the parts that could not be read, as branchinfo does: a refusal of the server (401, 403, 404),
// what the forge cannot do (ErrNotSupported), or an object the clone lacks (a shallow or partial clone)
type missing []gol.PartError

// note keeps err as a missing part and returns nil, or returns err when it is anything else; nil is nil
func (m *missing) note(part string, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, gol.ErrNotSupported) && !errors.Is(err, local.ErrObjectNotFound) &&
		!gol.IsUnauthorized(err) && !gol.IsForbidden(err) && !gol.IsNotFound(err) {
		return err
	}
	*m = append(*m, gol.PartError{Part: part, Status: gol.StatusCode(err), Message: err.Error()})
	return nil
}

// missingNotes words the missing parts, one line per kind of part and reason: the per-branch parts of branchinfo
// ("status of develop", "status of main"...) would otherwise take a line each
func missingNotes(parts []gol.PartError) []string {
	type group struct {
		kind, message string
		branches      []string
	}
	var groups []*group
	for _, p := range parts {
		kind, branch, _ := strings.Cut(p.Part, " of ")
		message := p.Message
		// Each names the commit it lacks: one reason for them all
		if strings.Contains(message, local.ErrObjectNotFound.Error()) {
			message = "the clone lacks the commits (a shallow or partial clone)"
		}
		i := slices.IndexFunc(groups, func(g *group) bool { return g.kind == kind && g.message == message })
		if i < 0 {
			groups = append(groups, &group{kind: kind, message: message})
			i = len(groups) - 1
		}
		if branch != "" {
			groups[i].branches = append(groups[i].branches, branch)
		}
	}
	notes := make([]string, len(groups))
	for i, g := range groups {
		what := g.kind
		if len(g.branches) > 0 {
			what += " (" + strings.Join(g.branches, ", ") + ")"
		}
		notes[i] = what + ": " + g.message
	}
	return notes
}
