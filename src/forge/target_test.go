// gitclt
// Tests how the repository and its owner are decided, from the current directory and the flags

package forge

import (
	"context"
	"gitclt/connections"
	"os"
	"path/filepath"
	"testing"
)

func clone(t *testing.T, config string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
}

func TestTarget(t *testing.T) {
	clone(t, "[remote \"origin\"]\n\turl = git@forge.example:corp/web.git\n")
	// None of these needs the server: a nil gitoperationslib client would panic if it were called
	c := &Client{Connection: connections.ActiveConnection{URL: "https://forge.example", Owner: "org", Username: "me"}}
	for _, tc := range []struct {
		name, flag, repo, owner, wantRepo string
	}{
		{"all from the clone", "", "", "corp", "web"},
		{"-o wins over the clone", "other", "", "other", "web"},
		{"the clone's own name: its owner", "", "web", "corp", "web"},
		{"the clone's own name, in another case", "", "WEB", "corp", "WEB"},
		{"the clone's own name, with -o", "other", "web", "other", "web"},
		{"another repository: as always", "", "api", "org", "api"},
		{"another repository, with -o", "other", "api", "other", "api"},
	} {
		owner, repo, err := c.Target(context.Background(), tc.flag, tc.repo)
		if err != nil || owner != tc.owner || repo != tc.wantRepo {
			t.Errorf("%s: got %q/%q, %+v", tc.name, owner, repo, err)
		}
	}
}

func TestTarget_CannotTell(t *testing.T) {
	c := &Client{Connection: connections.ActiveConnection{URL: "https://forge.example", Owner: "org"}}

	// Another forge's clone
	clone(t, "[remote \"origin\"]\n\turl = git@github.com:corp/web.git\n")
	if _, _, err := c.Target(context.Background(), "", ""); err == nil || err.Title != "Missing repository" {
		t.Fatalf("another forge: %+v", err)
	}
	// With a name, the remote does not matter: nothing changes from before
	if owner, repo, err := c.Target(context.Background(), "", "web"); err != nil || owner != "org" || repo != "web" {
		t.Fatalf("named: %q/%q, %+v", owner, repo, err)
	}

	// Not a clone at all
	t.Chdir(t.TempDir())
	if _, _, err := c.Target(context.Background(), "", ""); err == nil || err.Title != "Missing repository" {
		t.Fatalf("no clone: %+v", err)
	}
	if owner, repo, err := c.Target(context.Background(), "", "web"); err != nil || owner != "org" || repo != "web" {
		t.Fatalf("named: %q/%q, %+v", owner, repo, err)
	}
}
