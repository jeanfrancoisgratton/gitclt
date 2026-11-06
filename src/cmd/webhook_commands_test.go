// gitclt
// Tests the webhook commands' wiring: arguments per kind of webhook, and what reaches the server

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

const hook = `{"id":5,"type":"forgejo","url":"https://ci.example.com/hook","content_type":"json","events":["push"],"active":true}`

func TestWebhookCommands(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"GET /api/v1/admin/hooks":   testutil.Reply(`[` + hook + `]`),
		"POST /api/v1/admin/hooks":  testutil.Reply(hook),
		"GET /api/v1/admin/hooks/5": testutil.Reply(hook),
	}
	for _, base := range []string{"/api/v1/user/hooks", "/api/v1/repos/acme/web/hooks", "/api/v1/orgs/corp/hooks"} {
		routes["GET "+base] = testutil.Reply(`[` + hook + `]`)
		routes["POST "+base] = testutil.Reply(hook)
		routes["GET "+base+"/5"] = testutil.Reply(hook)
		routes["PATCH "+base+"/5"] = testutil.Reply(hook)
		routes["DELETE "+base+"/5"] = testutil.Status(http.StatusNoContent)
	}
	routes["POST /api/v1/repos/acme/web/hooks/5/tests"] = testutil.Status(http.StatusNoContent)
	testutil.Forgejo(t, routes)

	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"webhook", "user", "ls"}, nil, "ci.example.com"},
		{[]string{"hooks", "user", "get", "5"}, nil, "Content type:"},
		{[]string{"hook", "user", "add", "https://ci.example.com/hook"}, nil, "Created user webhook 5"},
		{[]string{"webhook", "user", "edit", "-a=false", "5"}, nil, "Updated user webhook 5"},
		{[]string{"webhook", "user", "edit", "5"}, nil, "Nothing to edit"},
		{[]string{"webhook", "user", "rm", "5"}, []string{"n"}, "Aborted"},
		{[]string{"webhook", "user", "rm", "-y", "5"}, nil, "Deleted user webhook 5"},
		{[]string{"webhook", "system", "ls", "-q"}, nil, "5"},
		{[]string{"webhook", "system", "create", "https://ci.example.com/hook"}, nil, "Created system webhook 5"},
		{[]string{"webhook", "default", "get", "5"}, nil, "is not a default webhook"},
		{[]string{"webhook", "repo", "ls", "web"}, nil, "ci.example.com"},
		{[]string{"webhook", "repo", "get", "web", "5"}, nil, "Content type:"},
		{[]string{"webhook", "repo", "create", "web", "https://ci.example.com/hook"}, nil, "Created webhook 5 of acme/web"},
		{[]string{"webhook", "repo", "edit", "web", "5", "-a=false"}, nil, "Updated webhook 5 of acme/web"},
		{[]string{"webhook", "repo", "rm", "-y", "web", "5"}, nil, "Deleted webhook 5 of acme/web"},
		{[]string{"webhook", "repository", "ping", "web", "5"}, nil, "Sent a test delivery"},
		{[]string{"webhook", "org", "ls", "corp"}, nil, "ci.example.com"},
		{[]string{"webhook", "organization", "get", "corp", "5"}, nil, "Content type:"},
		{[]string{"webhook", "org", "add", "corp", "https://ci.example.com/hook"}, nil, "Created webhook 5 of corp"},
		{[]string{"webhook", "org", "rm", "-y", "corp", "5"}, nil, "Deleted webhook 5 of corp"},
	} {
		if out, stderr := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

// From inside a clone, like the other repository commands; numbers are IDs, so several need no .
func TestWebhookCommands_FromAClone(t *testing.T) {
	var deleted []string
	srv := testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/corp/web/hooks":   testutil.Reply(`[` + hook + `]`),
		"GET /api/v1/repos/corp/web/hooks/5": testutil.Reply(hook),
		"GET /api/v1/repos/corp/web/hooks/6": testutil.Reply(hook),
		"GET /api/v1/repos/other/web/hooks":  testutil.Reply(`[]`),
		"DELETE /api/v1/repos/corp/web/hooks/{id}": func(w http.ResponseWriter, r *http.Request) {
			deleted = append(deleted, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		},
	})
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "[remote \"origin\"]\n\turl = " + srv.URL + "/corp/web.git\n"
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"webhook", "repo", "ls"}, "ci.example.com"},
		{[]string{"webhook", "repo", "ls", "."}, "ci.example.com"},
		{[]string{"webhook", "repo", "get", "5"}, "Content type:"},
		{[]string{"webhook", "repo", "ls", "-o", "other"}, "There is no webhook of other/web"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
	execute(t, []string{"webhook", "repo", "rm", "-y", "5", "6"})
	if !slices.Equal(deleted, []string{"5", "6"}) {
		t.Fatalf("deleted %v", deleted)
	}
}

func TestWebhookCommands_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"webhook", "user", "get"}, "accepts 1 arg(s)"},
		{[]string{"webhook", "user", "ls", "extra"}, "accepts 0 arg(s)"},
		{[]string{"webhook", "user", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"webhook", "system", "edit"}, "accepts 1 arg(s)"},
		// Refused before connecting: the API has no such call, whatever the connection
		{[]string{"webhook", "default", "ls"}, "cannot list default webhooks"},
		{[]string{"webhook", "repo", "get"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"webhook", "repo", "ls", "a", "b"}, "accepts between 0 and 1 arg(s)"},
		{[]string{"webhook", "org", "ls"}, "accepts 1 arg(s)"},
		{[]string{"webhook", "org", "get", "corp"}, "accepts 2 arg(s)"},
		{[]string{"webhook", "org", "rm", "corp"}, "requires at least 2 arg(s)"},
		{[]string{"webhook", "user", "add", "--secret", "--secret-file", "f", "u"}, "none of the others can be"},
		{[]string{"webhook", "user", "add", "--auth-header", "--auth-header-file", "f", "u"}, "none of the others can be"},
		{[]string{"webhook", "user", "edit", "-t", "slack", "5"}, "unknown shorthand flag"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestWebhookCommands_Help(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args           []string
		want, unwanted []string
	}{
		{[]string{"webhook", "--help"}, []string{"user", "system", "default", "repo", "org", "that URL is a credential"}, nil},
		{[]string{"webhook", "repo", "ls", "--help"}, []string{"From inside a clone", "--owner"}, []string{"An ID is never"}},
		{[]string{"webhook", "repo", "rm", "--help"}, []string{"From inside a clone", "An ID is never taken for <REPO>"}, nil},
		{[]string{"webhook", "user", "ls", "--help"}, nil, []string{"From inside a clone", "--owner"}},
		{[]string{"webhook", "system", "create", "--help"}, []string{"undocumented setting", "--secret-file", "--type"}, nil},
		{[]string{"webhook", "default", "create", "--help"}, []string{"cannot list default webhooks"}, nil},
		// Only what exists is offered
		{[]string{"webhook", "default", "--help"}, nil, []string{"  list ", "  ping "}},
		{[]string{"webhook", "user", "--help"}, []string{"  list "}, []string{"  ping "}},
		{[]string{"webhook", "repo", "--help"}, []string{"  list ", "  ping "}, nil},
		{[]string{"webhook", "user", "edit", "--help"}, []string{"replaces the whole list", "action_run_*"}, []string{"--type"}},
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
