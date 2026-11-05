// gitclt
// Tests the commands from inside a clone: what reaches the server tells which repository was understood

package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

// Starts in a subdirectory of a clone of corp/web, whose origin is on the fake server, and records the requests
func insideClone(t *testing.T) *[]string {
	t.Helper()
	var seen []string
	record := func(reply string, status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			seen = append(seen, r.Method+" "+r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(reply))
		}
	}
	tag := `{"name":"v1","id":"abc","commit":{"sha":"abc"}}`
	srv := testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/corp/web":                    record(`{"name":"web","full_name":"corp/web","owner":{"login":"corp"}}`, 200),
		"GET /api/v1/repos/corp/web/tags":               record(`[`+tag+`]`, 200),
		"GET /api/v1/repos/corp/web/tags/v1":            record(tag, 200),
		"DELETE /api/v1/repos/corp/web/tags/v1":         record(``, 204),
		"DELETE /api/v1/repos/corp/web/tags/v2":         record(``, 204),
		"GET /api/v1/repos/corp/web/releases":           record(`[]`, 200),
		"GET /api/v1/repos/corp/web/branches":           record(`[]`, 200),
		"GET /api/v1/repos/corp/web/pulls":              record(`[]`, 200),
		"GET /api/v1/repos/corp/web/branch_protections": record(`[]`, 200),
		"GET /api/v1/repos/corp/web/commits":            record(`[]`, 200),
		"GET /api/v1/repos/corp/web/tags/v2":            record(`{"name":"v2","id":"def","commit":{"sha":"def"}}`, 200),
		"GET /api/v1/repos/other/web":                   record(`{"name":"web"}`, 200),
		"GET /api/v1/repos/acme/api":                    record(`{"name":"api"}`, 200),
		"GET /api/v1/repos/other/web/tags":              record(`[]`, 200),
		"GET /api/v1/repos/acme/api/tags":               record(`[]`, 200),
	})
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "[remote \"origin\"]\n\turl = " + srv.URL + "/corp/web.git\n"
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(root, "src", "deep"))
	return &seen
}

func TestInference_ReadsTheRepoFromTheClone(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"tag", "ls"}, "GET /api/v1/repos/corp/web/tags"},
		{[]string{"tag", "ls", "."}, "GET /api/v1/repos/corp/web/tags"},
		{[]string{"tag", "get", "v1"}, "GET /api/v1/repos/corp/web/tags/v1"},
		{[]string{"tag", "get", ".", "v1"}, "GET /api/v1/repos/corp/web/tags/v1"},
		{[]string{"tag", "get", "web", "v1"}, "GET /api/v1/repos/corp/web/tags/v1"},
		{[]string{"release", "ls"}, "GET /api/v1/repos/corp/web/releases"},
		{[]string{"branch", "ls", "-r"}, "GET /api/v1/repos/corp/web/branches"},
		{[]string{"branch", "prot", "ls"}, "GET /api/v1/repos/corp/web/branch_protections"},
		{[]string{"pr", "ls"}, "GET /api/v1/repos/corp/web/pulls"},
		{[]string{"history", "ls"}, "GET /api/v1/repos/corp/web/commits"},
		{[]string{"repo", "info"}, "GET /api/v1/repos/corp/web"},
		// -o only replaces the owner; another repository is still asked for by name, with the connection's owner
		{[]string{"tag", "ls", "-o", "other"}, "GET /api/v1/repos/other/web/tags"},
		{[]string{"tag", "ls", "api"}, "GET /api/v1/repos/acme/api/tags"},
	} {
		seen := insideClone(t)
		out, stderr := execute(t, tc.args)
		if !slices.Contains(*seen, tc.want) {
			t.Errorf("%v: requests %v; output %q %q", tc.args, *seen, out, stderr)
		}
	}
}

// Several targets: the first of them is still the repository, as it always was, so nothing is deleted on a guess
func TestInference_SeveralTargets(t *testing.T) {
	seen := insideClone(t)
	out, _ := execute(t, []string{"tag", "rm", "-y", "v1", "v2"})
	if !strings.Contains(out, "does not exist") || !strings.Contains(out, "pass . for the current one") {
		t.Errorf("output %q", out)
	}
	for _, r := range *seen {
		if strings.HasPrefix(r, "DELETE") {
			t.Fatalf("deleted on a guess: %v", *seen)
		}
	}

	// The . says it out loud
	seen = insideClone(t)
	execute(t, []string{"tag", "rm", "-y", ".", "v1", "v2"})
	if !slices.Contains(*seen, "DELETE /api/v1/repos/corp/web/tags/v1") || !slices.Contains(*seen, "DELETE /api/v1/repos/corp/web/tags/v2") {
		t.Fatalf("requests %v", *seen)
	}

	// A single target cannot be anything else; the question names the repository, as it was not typed
	seen = insideClone(t)
	out, _ = execute(t, []string{"tag", "rm", "v1"}, "n")
	if !strings.Contains(out, "corp/web") || slices.Contains(*seen, "DELETE /api/v1/repos/corp/web/tags/v1") {
		t.Fatalf("output %q, requests %v", out, *seen)
	}
}

func TestSplitRepo(t *testing.T) {
	for _, tc := range []struct {
		args []string
		n    int
		repo string
		rest []string
	}{
		{[]string{}, 0, "", []string{}},
		{[]string{"web"}, 0, "web", []string{}},
		{[]string{"."}, 0, "", []string{}},
		{[]string{"v1"}, 1, "", []string{"v1"}},
		{[]string{"web", "v1"}, 1, "web", []string{"v1"}},
		{[]string{".", "v1"}, 1, "", []string{"v1"}},
		{[]string{"v1", "v2"}, 1, "v1", []string{"v2"}},
	} {
		repo, rest := splitRepo(tc.args, tc.n)
		if repo != tc.repo || !slices.Equal(rest, tc.rest) {
			t.Errorf("%v, %d: got %q, %v", tc.args, tc.n, repo, rest)
		}
	}
	for _, tc := range []struct {
		args []string
		n    int
		repo string
		rest []string
	}{
		{[]string{"12", "13"}, 1, "", []string{"12", "13"}},
		{[]string{"#12", "13"}, 1, "", []string{"#12", "13"}},
		{[]string{"web", "12", "13"}, 1, "web", []string{"12", "13"}},
		{[]string{".", "12", "13"}, 1, "", []string{"12", "13"}},
		{[]string{"12", "alice", "bob"}, 2, "", []string{"12", "alice", "bob"}},
		{[]string{"web", "12", "alice"}, 2, "web", []string{"12", "alice"}},
	} {
		repo, rest := splitRepoNumber(tc.args, tc.n)
		if repo != tc.repo || !slices.Equal(rest, tc.rest) {
			t.Errorf("number: %v, %d: got %q, %v", tc.args, tc.n, repo, rest)
		}
	}
}

// The subcommands' own --help tells it: cobra does not show the parent's
func TestInference_Help(t *testing.T) {
	testutil.WriteRegistry(t, "")
	const clone, owner, several, confirm, number = "From inside a clone", "takes the clone's owner", "use . for the current clone's",
		"confirmation question names the repository", "A number is never taken for <REPO>"
	for _, tc := range []struct {
		args           []string
		want, unwanted []string
	}{
		{[]string{"tag", "--help"}, []string{clone, owner, several}, []string{confirm}},
		{[]string{"tag", "ls", "--help"}, []string{clone, owner, several}, []string{confirm}},
		{[]string{"tag", "rm", "--help"}, []string{clone, owner, several, confirm}, nil},
		{[]string{"branch", "ls", "--help"}, []string{clone, owner, "--local alone needs none"}, []string{confirm}},
		{[]string{"branch", "rm", "--help"}, []string{clone, owner, several, confirm}, nil},
		{[]string{"branch", "prot", "ls", "--help"}, []string{clone, owner}, nil},
		{[]string{"release", "rm", "--help"}, []string{clone, confirm}, nil},
		{[]string{"release", "rma", "--help"}, []string{clone, confirm}, nil},
		{[]string{"release", "up", "--help"}, []string{clone, owner}, []string{confirm}},
		{[]string{"history", "ls", "--help"}, []string{clone, owner}, nil},
		{[]string{"pr", "ls", "--help"}, []string{clone, owner}, []string{number}},
		{[]string{"pr", "close", "--help"}, []string{clone, owner, number, "numeric name"}, nil},
		{[]string{"pr", "reopen", "--help"}, []string{number}, nil},
		{[]string{"pr", "reviewers", "--help"}, []string{number}, nil},
		{[]string{"repo", "info", "--help"}, []string{clone, owner}, []string{several}},
		{[]string{"repo", "edit", "--help"}, []string{clone, owner}, []string{several}},
		{[]string{"repo", "delete", "--help"}, nil, []string{clone}},
		{[]string{"repo", "create", "--help"}, nil, []string{clone}},
	} {
		out, stderr := execute(t, tc.args)
		for _, want := range tc.want {
			if !strings.Contains(out+stderr, want) {
				t.Errorf("%v: missing %q in:\n%s%s", tc.args, want, out, stderr)
			}
		}
		for _, unwanted := range tc.unwanted {
			if strings.Contains(out+stderr, unwanted) {
				t.Errorf("%v: unexpected %q in:\n%s%s", tc.args, unwanted, out, stderr)
			}
		}
	}
}
