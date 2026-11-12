// gitclt
// Tests reading a local working copy's .git directory: branches (loose and packed), HEAD, remotes, worktrees

package shared

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"
)

const (
	wcMain    = "1111111111111111111111111111111111111111"
	wcDevelop = "2222222222222222222222222222222222222222"
	wcOld     = "3333333333333333333333333333333333333333"
	wcFeature = "4444444444444444444444444444444444444444"
)

func writeGitFiles(t *testing.T, root string, files map[string]string) {
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
func fakeClone(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeGitFiles(t, root, map[string]string{
		".git/HEAD": "ref: refs/heads/develop\n",
		".git/packed-refs": "# pack-refs with: peeled fully-peeled sorted\n" +
			wcMain + " refs/heads/main\n" +
			wcOld + " refs/heads/develop\n" +
			wcOld + " refs/tags/v1.0.0\n^" + wcMain + "\n" +
			wcOld + " refs/remotes/origin/main\n",
		".git/refs/heads/develop":   wcDevelop + "\n",
		".git/refs/heads/feature/x": wcFeature + "\n",
		".git/config": "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = git@forge.example:acme/web.git\n" +
			"\tfetch = +refs/heads/*:refs/remotes/origin/*\n[branch \"develop\"]\n\tremote = origin\n\turl = not-a-remote\n",
		"src/main.go": "package main\n",
	})
	return root
}

func TestReadWorkingCopy(t *testing.T) {
	root := fakeClone(t)

	// From a subdirectory: the working copy is found in a parent
	r, err := ReadWorkingCopy(filepath.Join(root, "src"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"main": wcMain, "develop": wcDevelop, "feature/x": wcFeature}
	if !maps.Equal(r.Branches, want) || r.Current != "develop" {
		t.Fatalf("branches=%v, current=%q", r.Branches, r.Current)
	}
	if len(r.Remotes) != 1 || r.Remotes[0] != "git@forge.example:acme/web.git" {
		t.Fatalf("remotes=%q", r.Remotes)
	}

	// Detached HEAD: no current branch
	writeGitFiles(t, root, map[string]string{".git/HEAD": wcMain + "\n"})
	if r, err = ReadWorkingCopy(root); err != nil || r.Current != "" {
		t.Fatalf("detached: %+v, %v", r, err)
	}
}

func TestReadWorkingCopy_Worktree(t *testing.T) {
	root := fakeClone(t)
	wt := t.TempDir()
	writeGitFiles(t, root, map[string]string{
		".git/worktrees/hotfix/HEAD":      "ref: refs/heads/feature/x\n",
		".git/worktrees/hotfix/commondir": "../..\n",
	})
	writeGitFiles(t, wt, map[string]string{".git": "gitdir: " + filepath.Join(root, ".git/worktrees/hotfix") + "\n"})

	r, err := ReadWorkingCopy(wt)
	if err != nil || r.Current != "feature/x" || r.Branches["develop"] != wcDevelop || len(r.Remotes) != 1 {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestReadWorkingCopy_NoWorkingCopy(t *testing.T) {
	if _, err := ReadWorkingCopy(t.TempDir()); !errors.Is(err, ErrNoWorkingCopy) {
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
		r := &WorkingCopy{Remotes: []string{"https://elsewhere/x/y", tc.url}}
		if got := r.MatchesRepo(tc.owner, tc.repo); got != tc.want {
			t.Errorf("%s vs %s/%s: got %t", tc.url, tc.owner, tc.repo, got)
		}
	}
}
