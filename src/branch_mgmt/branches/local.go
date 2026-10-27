// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/local.go
// Original timestamp : 2026.10.26 16:40:12

package branches

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// localRepo is what branch ls needs from a local working copy
// It is read straight from the .git directory: gitclt does not need the git client to be installed
type localRepo struct {
	current  string            // the checked out branch; empty on a detached HEAD
	branches map[string]string // branch name → head commit
	remotes  []string          // the remotes' URLs
}

var errNoWorkingCopy = errors.New("not inside a git working copy")

// readLocalRepo reads the working copy that dir is part of (dir itself, or one of its parents)
func readLocalRepo(dir string) (*localRepo, error) {
	gitDir, err := findGitDir(dir)
	if err != nil {
		return nil, err
	}
	// A linked worktree has its own HEAD, but shares the branches and the config with the main working copy
	commonDir := gitDir
	if b, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		commonDir = resolve(gitDir, strings.TrimSpace(string(b)))
	}

	r := &localRepo{branches: map[string]string{}}
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return nil, err
	}
	if ref, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/"); ok {
		r.current = ref
	}
	if err = r.readPackedRefs(filepath.Join(commonDir, "packed-refs")); err != nil {
		return nil, err
	}
	if err = r.readLooseRefs(filepath.Join(commonDir, "refs", "heads")); err != nil {
		return nil, err
	}
	if r.remotes, err = readRemoteURLs(filepath.Join(commonDir, "config")); err != nil {
		return nil, err
	}
	return r, nil
}

// findGitDir looks for .git in dir and its parents; .git is a directory, or a file pointing to one (worktrees, submodules)
func findGitDir(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, ".git")
		if fi, err := os.Stat(p); err == nil {
			if fi.IsDir() {
				return p, nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			if target, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir: "); ok {
				return resolve(dir, target), nil
			}
			return "", errors.New(p + " is neither a directory nor a gitdir link")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoWorkingCopy
		}
		dir = parent
	}
}

func resolve(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

// packed-refs holds the branches git has packed (git gc, or a fresh clone): "<sha> refs/heads/<name>" lines
func (r *localRepo) readPackedRefs(path string) error {
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
			r.branches[name] = sha
		}
	}
	return s.Err()
}

// Loose refs are one file per branch (feature/x is refs/heads/feature/x); they are newer than packed ones
func (r *localRepo) readLooseRefs(root string) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name, _ := filepath.Rel(root, path)
		r.branches[filepath.ToSlash(name)] = strings.TrimSpace(string(b))
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// readRemoteURLs returns the url values of the [remote "..."] sections of a git config file
func readRemoteURLs(path string) ([]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var urls []string
	inRemote := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "[") {
			inRemote = strings.HasPrefix(line, "[remote ")
			continue
		}
		if key, val, ok := strings.Cut(line, "="); ok && inRemote && strings.TrimSpace(key) == "url" {
			urls = append(urls, strings.TrimSpace(val))
		}
	}
	return urls, s.Err()
}

// matchesRepo tells if one of the remotes' URLs points to owner/repo, whatever the host and protocol
// https://forge/owner/repo(.git), ssh://git@forge:2222/owner/repo, git@forge:owner/repo.git...
// An empty owner matches any owner
func (r *localRepo) matchesRepo(owner, repo string) bool {
	for _, u := range r.remotes {
		u = strings.TrimSuffix(strings.TrimSuffix(u, "/"), ".git")
		parts := strings.FieldsFunc(u, func(c rune) bool { return c == '/' || c == ':' })
		if n := len(parts); n >= 2 && (owner == "" || strings.EqualFold(parts[n-2], owner)) && strings.EqualFold(parts[n-1], repo) {
			return true
		}
	}
	return false
}
