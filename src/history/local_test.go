// gitclt
// Tests history ls from a clone: the checked out branch by default, and the notice about local commits

package history

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/testutil"
)

const (
	shaServer = "1111111111111111111111111111111111111111"
	shaLocal  = "2222222222222222222222222222222222222222"
)

// cloneOf makes the current directory a clone of repo (owner/name), on branch current, with the given local branch heads
func cloneOf(t *testing.T, repo, current string, heads map[string]string) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".git/HEAD":   "ref: refs/heads/" + current + "\n",
		".git/config": "[remote \"origin\"]\n\turl = git@forge.example:" + repo + ".git\n",
	}
	for b, sha := range heads {
		files[".git/refs/heads/"+b] = sha + "\n"
	}
	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
}

// cloneServer serves acme/web with the given server branches, all at shaServer; counts answers the compare endpoint
// (base...head → commits of head missing from base), anything else is a 404, as for a local head the server never saw
func cloneServer(t *testing.T, refs *[]string, branches []string, counts map[string]int) {
	t.Helper()
	const base = "/api/v1/repos/acme/web"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base: testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","default_branch":"develop"}`),
		"GET " + base + "/branches/{b}": func(w http.ResponseWriter, r *http.Request) {
			for _, b := range branches {
				if b == r.PathValue("b") {
					testutil.Reply(fmt.Sprintf(`{"name":%q,"commit":{"id":%q}}`, b, shaServer))(w, r)
					return
				}
			}
			testutil.Status(http.StatusNotFound)(w, r)
		},
		"GET " + base + "/commits": func(w http.ResponseWriter, r *http.Request) {
			*refs = append(*refs, r.URL.Query().Get("sha"))
			testutil.Reply("["+commitJSON(shaServer, "Pushed")+"]")(w, r)
		},
		"GET " + base + "/compare/{basehead}": func(w http.ResponseWriter, r *http.Request) {
			n, ok := counts[r.PathValue("basehead")]
			if !ok {
				testutil.Status(http.StatusNotFound)(w, r)
				return
			}
			testutil.Reply(fmt.Sprintf(`{"total_commits":%d}`, n))(w, r)
		},
	})
}

func TestList_FromClone(t *testing.T) {
	for _, tc := range []struct {
		name      string
		current   string
		heads     map[string]string
		branches  []string // on the server
		counts    map[string]int
		ref       string // -r
		wantRef   string // asked of the server
		wantInOut string // "" means no notice
	}{
		{"checked out branch, in sync", "feature", map[string]string{"feature": shaServer}, []string{"develop", "feature"}, nil, "", "feature", ""},
		{"branch not on the server", "feature", map[string]string{"feature": shaLocal}, []string{"develop"}, nil, "", "", "not on the server yet"},
		{"unpushed commits", "develop", map[string]string{"develop": shaLocal}, []string{"develop"}, nil, "", "develop", "not pushed yet"},
		{"behind the server", "develop", map[string]string{"develop": shaLocal}, []string{"develop"},
			map[string]int{"develop..." + shaLocal: 0, shaLocal + "...develop": 2}, "", "develop", "2 commit(s) behind"},
		{"-r wins, and gets the notice", "feature", map[string]string{"feature": shaServer, "develop": shaLocal}, []string{"develop", "feature"},
			nil, "develop", "develop", "develop has commits that are not pushed yet"},
		{"-r on a tag: no notice", "feature", map[string]string{"feature": shaLocal}, []string{"develop", "feature"}, nil, "v1.0.0", "v1.0.0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaults(t)
			testutil.Set(t, &ListRef, tc.ref)
			cloneOf(t, "acme/web", tc.current, tc.heads)
			var refs []string
			cloneServer(t, &refs, tc.branches, tc.counts)

			out := testutil.StripANSI(testutil.Run(t, func() {
				if _, err := List("web", true); err != nil {
					t.Fatal(err.Error())
				}
			}))
			if len(refs) != 1 || refs[0] != tc.wantRef {
				t.Fatalf("asked for %q", refs)
			}
			if tc.wantInOut == "" && strings.Contains(out, "⚠") || tc.wantInOut != "" && !strings.Contains(out, tc.wantInOut) {
				t.Fatalf("output:\n%s", out)
			}
		})
	}
}

// In a clone of another repository: the default branch, and no notice, whatever the local branches are
func TestList_OtherClone(t *testing.T) {
	defaults(t)
	cloneOf(t, "acme/site", "feature", map[string]string{"feature": shaLocal, "develop": shaLocal})
	var refs []string
	cloneServer(t, &refs, []string{"develop", "feature"}, nil)

	out := testutil.Run(t, func() {
		if _, err := List("web", true); err != nil {
			t.Fatal(err.Error())
		}
	})
	if len(refs) != 1 || refs[0] != "" || strings.Contains(out, "⚠") {
		t.Fatalf("asked for %q, output:\n%s", refs, out)
	}
}
