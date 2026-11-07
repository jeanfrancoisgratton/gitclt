// gitclt
// Tests the issue commands' wiring: optional repository, nested commands, and what the flags allow

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

func issueServer(t *testing.T, owner string, deleted *[]string) string {
	t.Helper()
	base := "/api/v1/repos/" + owner + "/web"
	issue := `{"number":2,"title":"First","state":"open","user":{"login":"acme"},"labels":[{"name":"bug"}],"comments":0}`
	srv := testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base:              testutil.Reply(`{"name":"web"}`),
		"GET " + base + "/issues":  testutil.Reply(`[` + issue + `]`),
		"POST " + base + "/issues": testutil.Reply(issue),
		"GET " + base + "/issues/{n}": func(w http.ResponseWriter, r *http.Request) {
			if r.PathValue("n") == "2" || r.PathValue("n") == "3" {
				testutil.Reply(issue)(w, r)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
		"PATCH " + base + "/issues/2": testutil.Reply(issue),
		"DELETE " + base + "/issues/{n}": func(w http.ResponseWriter, r *http.Request) {
			*deleted = append(*deleted, r.PathValue("n"))
			w.WriteHeader(http.StatusNoContent)
		},
		"GET " + base + "/issues/2/comments":  testutil.Reply(`[{"id":16,"user":{"login":"acme"},"body":"Hello"}]`),
		"POST " + base + "/issues/2/comments": testutil.Reply(`{"id":17,"body":"x"}`),
		"GET " + base + "/labels":             testutil.Reply(`[{"id":1,"name":"bug","color":"d73a4a"},{"id":2,"name":"docs","color":"0075ca"}]`),
		"POST " + base + "/labels":            testutil.Reply(`{"id":3,"name":"new","color":"d73a4a"}`),
		"DELETE " + base + "/labels/{id}":     testutil.Status(http.StatusNoContent),
		"GET " + base + "/milestones":         testutil.Reply(`[{"id":1,"title":"v1.5","state":"open"}]`),
		"POST " + base + "/milestones":        testutil.Reply(`{"id":2,"title":"new","state":"open"}`),
		"DELETE " + base + "/milestones/{id}": testutil.Status(http.StatusNoContent),
	})
	return srv.URL
}

func TestIssueCommands(t *testing.T) {
	var deleted []string
	issueServer(t, "acme", &deleted)
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"issue", "ls", "web"}, nil, "First"},
		{[]string{"issues", "list", "-q", "web"}, nil, "2"},
		{[]string{"issue", "get", "web", "2"}, nil, "Issue:"},
		{[]string{"issue", "create", "-t", "First", "web"}, nil, "Created issue #2"},
		{[]string{"issue", "edit", "web", "2", "-t", "x"}, nil, "Updated issue #2"},
		{[]string{"issue", "edit", "web", "2"}, nil, "Nothing to edit"},
		{[]string{"issue", "close", "web", "2"}, nil, "Closed issue #2"},
		{[]string{"issue", "rm", "web", "2"}, []string{"n"}, "Aborted"},
		{[]string{"issue", "rm", "-y", "web", "2"}, nil, "Deleted issue #2"},
		{[]string{"issue", "comment", "ls", "web", "2"}, nil, "Hello"},
		{[]string{"issue", "comments", "add", "-b", "hi", "web", "2"}, nil, "Added comment 17"},
		{[]string{"issue", "label", "ls", "web"}, nil, "#d73a4a"},
		{[]string{"issue", "labels", "web"}, nil, "#d73a4a"},
		{[]string{"issue", "label", "-j", "web"}, nil, `"color"`},
		{[]string{"issue", "labels", "-q", "web"}, nil, "docs"},
		{[]string{"issue", "labels", "add", "-C", "d73a4a", "web", "new"}, nil, "Created label new"},
		{[]string{"issue", "label", "rm", "-y", "web", "bug"}, nil, "Deleted label bug"},
		{[]string{"issue", "milestone", "ls", "web"}, nil, "v1.5"},
		{[]string{"issue", "ms", "add", "web", "new"}, nil, "Created milestone new"},
		{[]string{"issue", "milestones", "rm", "-y", "web", "v1.5"}, nil, "Deleted milestone v1.5"},
	} {
		if out, stderr := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

// From inside a clone; numbers are never a repository, so several need no .
func TestIssueCommands_FromAClone(t *testing.T) {
	var deleted []string
	url := issueServer(t, "corp", &deleted)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[remote \"origin\"]\n\turl = "+url+"/corp/web.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"issue", "ls"}, "First"},
		{[]string{"issue", "get", "#2"}, "Issue:"},
		{[]string{"issue", "create", "-t", "First"}, "corp/web"},
		{[]string{"issue", "label", "ls"}, "docs"},
		{[]string{"issue", "labels"}, "docs"},
		{[]string{"issue", "ms", "ls"}, "v1.5"},
		// Several names: the first is the repository, and nothing is deleted on a guess
		{[]string{"issue", "label", "rm", "-y", "bug", "docs"}, "pass . for the current one"},
		{[]string{"issue", "label", "rm", "-y", ".", "bug", "docs"}, "Deleted label docs"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
	execute(t, []string{"issue", "rm", "-y", "2", "3"})
	if !slices.Equal(deleted, []string{"2", "3"}) {
		t.Fatalf("deleted %v", deleted)
	}
}

func TestIssueCommands_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"issue", "ls", "a", "b"}, "accepts at most 1 arg(s)"},
		{[]string{"issue", "get"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"issue", "create"}, `required flag(s) "title" not set`},
		{[]string{"issue", "create", "-t", "x", "-b", "y", "-F", "z"}, "none of the others can be"},
		{[]string{"issue", "edit", "-l", "a", "--add-label", "b", "2"}, "none of the others can be"},
		{[]string{"issue", "edit", "-l", "a", "--remove-label", "b", "2"}, "none of the others can be"},
		{[]string{"issue", "close"}, "requires at least 1 arg(s)"},
		{[]string{"issue", "comment", "add"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"issue", "comment", "add", "-b", "y", "-F", "z", "2"}, "none of the others can be"},
		{[]string{"issue", "label", "add", "x"}, `required flag(s) "color" not set`},
		{[]string{"issue", "label", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"issue", "ms", "get"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"issue", "ms", "rm"}, "requires at least 1 arg(s)"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestIssueCommands_Help(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args           []string
		want, unwanted []string
	}{
		{[]string{"issue", "--help"}, []string{"Bitbucket Cloud retired", "refuse a pull request's number", "From inside a clone"}, nil},
		{[]string{"issue", "ls", "--help"}, []string{"From inside a clone", "--owner"}, []string{"A number is never"}},
		{[]string{"issue", "close", "--help"}, []string{"A number is never taken for <REPO>", "numeric name"}, nil},
		{[]string{"issue", "label", "--help"}, []string{"share a name"}, nil},
		{[]string{"issue", "label", "rm", "--help"}, []string{"use . for the current clone's"}, nil},
		{[]string{"issue", "ms", "--help"}, []string{"share a title"}, nil},
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
