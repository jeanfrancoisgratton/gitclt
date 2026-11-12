// gitclt
// A fake local working copy, for the tests of the commands that read one (shared.ReadWorkingCopy is tested in shared)

package branches

import (
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
