// gitclt
// Tests how a clone's repository is told from its remotes, on the connection's server (local.Repo.RepoOn, and the -c hint)

package forge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

func workingCopy(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func remote(name, url string) string {
	return "[remote \"" + name + "\"]\n\turl = " + url + "\n\tfetch = +refs/heads/*:refs/remotes/" + name + "/*\n"
}

func TestInferRepo(t *testing.T) {
	for _, tc := range []struct {
		name, config, server string
		owner, repo, remote  string
	}{
		{"from anywhere in the clone", remote("origin", "git@forge.example:acme/web.git"), "https://forge.example", "acme", "web", "origin"},
		{"the SSH port is not the web port", remote("origin", "ssh://git@forge.example:9722/acme/web.git"), "https://forge.example:3000", "acme", "web", "origin"},
		{"the connection picks the forge, not origin",
			remote("origin", "git@forge.example:acme/web.git") + remote("gh", "git@github.com:me/web.git"), "https://github.com/", "me", "web", "gh"},
		{"Bitbucket's SSH host", remote("bb", "git@ssh.bitbucket.org:ws/web.git"), "https://bitbucket.org", "ws", "web", "bb"},
		{"GitHub's API host", remote("origin", "git@github.com:me/web.git"), "https://api.github.com", "me", "web", "origin"},
		{"host case", remote("origin", "git@FORGE.example:acme/web.git"), "https://forge.example", "acme", "web", "origin"},
		{"origin wins on the same server",
			remote("upstream", "git@forge.example:corp/web.git") + remote("origin", "git@forge.example:me/web.git"), "https://forge.example", "me", "web", "origin"},
		{"two remotes, same repository",
			remote("a", "git@forge.example:acme/web.git") + remote("b", "https://forge.example/acme/web.git"), "https://forge.example", "acme", "web", "a"},
		{"a local remote is skipped", remote("disk", "/srv/web.git") + remote("up", "git@forge.example:acme/web.git"), "https://forge.example", "acme", "web", "up"},
		{"no scheme in the connection", remote("origin", "git@forge.example:acme/web.git"), "forge.example", "acme", "web", "origin"},
	} {
		root := workingCopy(t, "[core]\n\tbare = false\n"+tc.config+"[branch \"main\"]\n\tremote = origin\n\turl = not-a-remote\n")
		got, err := inferRepo(filepath.Join(root, "src", "deep"), tc.server)
		if err != nil || got.Owner != tc.owner || got.Name != tc.repo || got.Remote != tc.remote {
			t.Errorf("%s: got %+v, %v", tc.name, got, err)
		}
	}
}

// Failing is right here: guessing would act on the wrong repository
func TestInferRepo_Refusals(t *testing.T) {
	if _, err := inferRepo(t.TempDir(), "https://forge.example"); !errors.Is(err, local.ErrNoWorkingCopy) {
		t.Errorf("no working copy: %v", err)
	}
	for name, tc := range map[string]struct{ config, want string }{
		"no remote":        {"[core]\n\tbare = false\n", "has no remote"},
		"another forge":    {remote("origin", "git@github.com:me/web.git"), "origin is on github.com"},
		"only local":       {remote("disk", "/srv/web.git"), "no remote of the working copy is on this server"},
		"different owners": {remote("a", "git@forge.example:acme/web.git") + remote("b", "git@forge.example:me/web.git"), "a: acme/web, b: me/web"},
		"lookalike host":   {remote("origin", "git@notforge.example:acme/web.git"), "no remote of the working copy is on this server"},
	} {
		_, err := inferRepo(workingCopy(t, tc.config), "https://forge.example")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// A linked worktree keeps its config in the main working copy
func TestInferRepo_Worktree(t *testing.T) {
	main := workingCopy(t, remote("origin", "git@forge.example:acme/web.git"))
	gitDir := filepath.Join(main, ".git", "worktrees", "wt")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := t.TempDir()
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inferRepo(wt, "https://forge.example")
	if err != nil || got.Owner != "acme" || got.Name != "web" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A clone with no remote on the connection's server points to -c
func TestInferRepo_Hint(t *testing.T) {
	_, err := inferRepo(workingCopy(t, remote("origin", "git@github.com:me/web.git")), "https://forge.example")
	if !errors.Is(err, local.ErrNoRemoteOnHost) || !strings.HasSuffix(err.Error(), "; choose the matching connection with -c") {
		t.Fatalf("got %v", err)
	}
}
