// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/workingcopy.go (was src/branch_mgmt/branches/local.go)
// Original timestamp : 2026.10.26 16:40:12

package shared

import (
	"strings"

	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

// ErrNoWorkingCopy is returned when a directory is not part of a git working copy
var ErrNoWorkingCopy = local.ErrNoWorkingCopy

// WorkingCopy is what the branch and history commands need from a local working copy
// The library's local package reads it straight from the .git directory: gitclt does not need the git client
type WorkingCopy struct {
	Current  string            // the checked out branch; empty on a detached HEAD
	Branches map[string]string // branch name → head commit
	Remotes  []string          // the remotes' URLs
}

// ReadWorkingCopy reads the working copy that dir is part of (dir itself, or one of its parents)
// A linked worktree has its own HEAD, but shares the branches and the remotes with the main working copy
func ReadWorkingCopy(dir string) (*WorkingCopy, error) {
	repo, err := local.Open(dir)
	if err != nil {
		return nil, err
	}
	defer repo.Close()

	r := &WorkingCopy{Branches: map[string]string{}}
	if r.Current, err = repo.Current(); err != nil {
		return nil, err
	}
	branches, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	for _, b := range branches {
		r.Branches[b.Name] = b.SHA
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	for _, remote := range remotes {
		r.Remotes = append(r.Remotes, remote.URL)
	}
	return r, nil
}

// MatchesRepo tells if one of the remotes' URLs points to owner/repo, whatever the host and protocol
// https://forge/owner/repo(.git), ssh://git@forge:2222/owner/repo, git@forge:owner/repo.git...
// An empty owner matches any owner; a remote that is a local path matches nothing
func (r *WorkingCopy) MatchesRepo(owner, repo string) bool {
	for _, u := range r.Remotes {
		if _, o, n, ok := local.ParseRemoteURL(u); ok && (owner == "" || strings.EqualFold(o, owner)) && strings.EqualFold(n, repo) {
			return true
		}
	}
	return false
}
