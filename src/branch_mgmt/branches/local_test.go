// gitclt
// Tests reading a local working copy's .git directory: branches (loose and packed), HEAD, remotes, worktrees

package branches

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"
)

const (
	shaMain    = "1111111111111111111111111111111111111111"
	shaDevelop = "2222222222222222222222222222222222222222"
	shaOld     = "3333333333333333333333333333333333333333"
	shaFeature = "4444444444444444444444444444444444444444"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// A working copy of acme/web, on develop: main is packed, develop is both packed (stale) and loose
func fakeWorkingCopy(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		".git/HEAD": "ref: refs/heads/develop\n",
		".git/packed-refs": "# pack-refs with: peeled fully-peeled sorted\n" +
			shaMain + " refs/heads/main\n" +
			shaOld + " refs/heads/develop\n" +
			shaOld + " refs/tags/v1.0.0\n^" + shaMain + "\n" +
			shaOld + " refs/remotes/origin/main\n",
		".git/refs/heads/develop":   shaDevelop + "\n",
		".git/refs/heads/feature/x": shaFeature + "\n",
		".git/config": "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = git@forge.example:acme/web.git\n" +
			"\tfetch = +refs/heads/*:refs/remotes/origin/*\n[branch \"develop\"]\n\tremote = origin\n\turl = not-a-remote\n",
		"src/main.go": "package main\n",
	})
	return root
}

func TestReadLocalRepo(t *testing.T) {
	root := fakeWorkingCopy(t)

	// From a subdirectory: the working copy is found in a parent
	r, err := readLocalRepo(filepath.Join(root, "src"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"main": shaMain, "develop": shaDevelop, "feature/x": shaFeature}
	if !maps.Equal(r.branches, want) || r.current != "develop" {
		t.Fatalf("branches=%v, current=%q", r.branches, r.current)
	}
	if len(r.remotes) != 1 || r.remotes[0] != "git@forge.example:acme/web.git" {
		t.Fatalf("remotes=%q", r.remotes)
	}

	// Detached HEAD: no current branch
	writeFiles(t, root, map[string]string{".git/HEAD": shaMain + "\n"})
	if r, err = readLocalRepo(root); err != nil || r.current != "" {
		t.Fatalf("detached: %+v, %v", r, err)
	}
}

func TestReadLocalRepo_Worktree(t *testing.T) {
	root := fakeWorkingCopy(t)
	wt := t.TempDir()
	writeFiles(t, root, map[string]string{
		".git/worktrees/hotfix/HEAD":      "ref: refs/heads/feature/x\n",
		".git/worktrees/hotfix/commondir": "../..\n",
	})
	writeFiles(t, wt, map[string]string{".git": "gitdir: " + filepath.Join(root, ".git/worktrees/hotfix") + "\n"})

	r, err := readLocalRepo(wt)
	if err != nil || r.current != "feature/x" || r.branches["develop"] != shaDevelop || len(r.remotes) != 1 {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestReadLocalRepo_NoWorkingCopy(t *testing.T) {
	if _, err := readLocalRepo(t.TempDir()); !errors.Is(err, errNoWorkingCopy) {
		t.Fatalf("got %v", err)
	}
}

func TestMatchesRepo(t *testing.T) {
	for _, tc := range []struct {
		url, owner, repo string
		want             bool
	}{
		{"https://forge.example/acme/web.git", "acme", "web", true},
		{"https://forge.example/acme/web/", "ACME", "Web", true},
		{"git@forge.example:acme/web.git", "acme", "web", true},
		{"ssh://git@forge.example:2222/acme/web", "acme", "web", true},
		{"https://forge.example/acme/web.git", "", "web", true},
		{"https://forge.example/acme/web.git", "other", "web", false},
		{"https://forge.example/acme/website.git", "acme", "web", false},
		{"web", "", "web", false},
	} {
		r := &localRepo{remotes: []string{"https://elsewhere/x/y", tc.url}}
		if got := r.matchesRepo(tc.owner, tc.repo); got != tc.want {
			t.Errorf("%s vs %s/%s: got %t", tc.url, tc.owner, tc.repo, got)
		}
	}
}
