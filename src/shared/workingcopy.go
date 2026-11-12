// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/workingcopy.go (was src/branch_mgmt/branches/local.go)
// Original timestamp : 2026.10.26 16:40:12

package shared

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WorkingCopy is what the branch and history commands need from a local working copy
// It is read straight from the .git directory: gitclt does not need the git client to be installed
type WorkingCopy struct {
	Current  string            // the checked out branch; empty on a detached HEAD
	Branches map[string]string // branch name → head commit
	Remotes  []string          // the remotes' URLs
}

// ReadWorkingCopy reads the working copy that dir is part of (dir itself, or one of its parents)
func ReadWorkingCopy(dir string) (*WorkingCopy, error) {
	gitDir, err := FindGitDir(dir)
	if err != nil {
		return nil, err
	}
	// A linked worktree has its own HEAD, but shares the branches and the config with the main working copy
	commonDir := CommonGitDir(gitDir)

	r := &WorkingCopy{Branches: map[string]string{}}
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return nil, err
	}
	if ref, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/"); ok {
		r.Current = ref
	}
	if err = r.readPackedRefs(filepath.Join(commonDir, "packed-refs")); err != nil {
		return nil, err
	}
	if err = r.readLooseRefs(filepath.Join(commonDir, "refs", "heads")); err != nil {
		return nil, err
	}
	remotes, err := ReadGitRemotes(filepath.Join(commonDir, "config"))
	if err != nil {
		return nil, err
	}
	for _, remote := range remotes {
		r.Remotes = append(r.Remotes, remote.URL)
	}
	return r, nil
}

// packed-refs holds the branches git has packed (git gc, or a fresh clone): "<sha> refs/heads/<name>" lines
func (r *WorkingCopy) readPackedRefs(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		sha, ref, ok := strings.Cut(s.Text(), " ")
		if !ok {
			continue // comments, and "^<sha>" peeled tag lines, have no space
		}
		if name, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			r.Branches[name] = sha
		}
	}
	return s.Err()
}

// Loose refs are one file per branch (feature/x is refs/heads/feature/x); they are newer than packed ones
func (r *WorkingCopy) readLooseRefs(root string) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name, _ := filepath.Rel(root, path)
		r.Branches[filepath.ToSlash(name)] = strings.TrimSpace(string(b))
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// MatchesRepo tells if one of the remotes' URLs points to owner/repo, whatever the host and protocol
// https://forge/owner/repo(.git), ssh://git@forge:2222/owner/repo, git@forge:owner/repo.git...
// An empty owner matches any owner
func (r *WorkingCopy) MatchesRepo(owner, repo string) bool {
	for _, u := range r.Remotes {
		u = strings.TrimSuffix(strings.TrimSuffix(u, "/"), ".git")
		parts := strings.FieldsFunc(u, func(c rune) bool { return c == '/' || c == ':' })
		if n := len(parts); n >= 2 && (owner == "" || strings.EqualFold(parts[n-2], owner)) && strings.EqualFold(parts[n-1], repo) {
			return true
		}
	}
	return false
}
