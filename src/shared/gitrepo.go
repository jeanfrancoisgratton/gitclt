// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/gitrepo.go
// Original timestamp : 2026.11.05 12:00:00

package shared

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Everything here is read straight from the .git directory: gitclt does not need the git client to be installed

// ErrNoWorkingCopy is returned when a directory is not part of a git working copy
var ErrNoWorkingCopy = errors.New("not inside a git working copy")

// GitRemote is one url line of a [remote "..."] section of a git config file
type GitRemote struct {
	Name string
	URL  string
}

// LocalRepo is the repository a working copy is a clone of, as seen from one server
type LocalRepo struct {
	Owner  string
	Name   string
	Remote string // the remote it was read from
}

// FindGitDir looks for .git in dir and its parents; .git is a directory, or a file pointing to one (worktrees, submodules)
func FindGitDir(dir string) (string, error) {
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
				return resolvePath(dir, target), nil
			}
			return "", errors.New(p + " is neither a directory nor a gitdir link")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoWorkingCopy
		}
		dir = parent
	}
}

// CommonGitDir returns the directory holding the branches and the config: a linked worktree shares them with the main working copy
func CommonGitDir(gitDir string) string {
	if b, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		return resolvePath(gitDir, strings.TrimSpace(string(b)))
	}
	return gitDir
}

func resolvePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

// ReadGitRemotes returns the url lines of the [remote "..."] sections of a git config file
func ReadGitRemotes(path string) ([]GitRemote, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var remotes []GitRemote
	name, inRemote := "", false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "[") {
			inRemote = strings.HasPrefix(line, "[remote ")
			name = strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "[remote "), "]"), `"`)
			continue
		}
		if key, val, ok := strings.Cut(line, "="); ok && inRemote && strings.TrimSpace(key) == "url" {
			remotes = append(remotes, GitRemote{Name: name, URL: strings.TrimSpace(val)})
		}
	}
	return remotes, s.Err()
}

// ParseRemoteURL splits a remote's URL: https://forge/owner/repo(.git), ssh://git@forge:2222/owner/repo, git@forge:owner/repo.git...
// ok is false for what points to no server (a local path, file://)
func ParseRemoteURL(remote string) (host, owner, repo string, ok bool) {
	var path string
	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil || u.Hostname() == "" {
			return "", "", "", false
		}
		host, path = u.Hostname(), u.Path
	} else {
		// scp-like syntax: git itself takes a colon after the first slash as part of a local path
		h, p, found := strings.Cut(remote, ":")
		if !found || strings.Contains(h, "/") {
			return "", "", "", false
		}
		_, host, _ = cutLast(h, "@")
		path = p
	}
	parts := strings.Split(strings.TrimSuffix(strings.Trim(path, "/"), ".git"), "/")
	n := len(parts)
	if n < 2 || parts[n-2] == "" || parts[n-1] == "" {
		return "", "", "", false
	}
	return host, parts[n-2], parts[n-1], true
}

// cutLast is strings.Cut on the last separator; without one, everything is in after
func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return "", s, false
}

// sameHost compares a forge's hosts: the SSH and API hosts are often separate from the web one (ex: ssh.bitbucket.org)
func sameHost(a, b string) bool {
	norm := func(h string) string {
		h = strings.ToLower(h)
		for _, prefix := range []string{"altssh.", "ssh.", "api.", "www."} {
			if trimmed, ok := strings.CutPrefix(h, prefix); ok {
				return trimmed
			}
		}
		return h
	}
	return norm(a) == norm(b)
}

// InferRepo finds the repository that dir (or one of its parents) is a clone of, on the server at serverURL
// Only the remotes on that server count: another forge's repository of the same name is not the one being asked about
// Among several, origin wins; failing that, they must all point to the same repository
func InferRepo(dir, serverURL string) (*LocalRepo, error) {
	gitDir, err := FindGitDir(dir)
	if err != nil {
		return nil, err
	}
	remotes, err := ReadGitRemotes(filepath.Join(CommonGitDir(gitDir), "config"))
	if err != nil {
		return nil, err
	}
	if len(remotes) == 0 {
		return nil, errors.New("this git working copy has no remote")
	}
	if !strings.Contains(serverURL, "://") {
		serverURL = "https://" + serverURL
	}
	server, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	var found []LocalRepo
	var seen []string
	for _, r := range remotes {
		host, owner, name, ok := ParseRemoteURL(r.URL)
		if !ok {
			continue
		}
		seen = append(seen, r.Name+" is on "+host)
		if sameHost(host, server.Hostname()) {
			if r.Name == "origin" {
				return &LocalRepo{Owner: owner, Name: name, Remote: r.Name}, nil
			}
			found = append(found, LocalRepo{Owner: owner, Name: name, Remote: r.Name})
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("none of this working copy's remotes is on %s (%s); choose the matching connection with -c",
			server.Hostname(), strings.Join(seen, ", "))
	}
	for _, r := range found[1:] {
		if !strings.EqualFold(r.Owner, found[0].Owner) || !strings.EqualFold(r.Name, found[0].Name) {
			names := make([]string, len(found))
			for i, f := range found {
				names[i] = f.Remote + ": " + f.Owner + "/" + f.Name
			}
			return nil, fmt.Errorf("this working copy's remotes on %s point to different repositories (%s), and none is called origin",
				server.Hostname(), strings.Join(names, ", "))
		}
	}
	return &found[0], nil
}
