// gitclt
// Tests the repo commands' wiring: required and mutually exclusive flags, and a run against a fake Forgejo

package cmd

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestRepoCommands_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"repo", "info", "-o", "acme"}, "accepts 1 arg(s)"},
		{[]string{"repo", "ls", "-s", "c", "-S", "c"}, "none of the others can be"},
		{[]string{"repo", "create", "-o", "me", "--defaultbranch", "main", "web"}, "unknown flag: --defaultbranch"},
	} {
		if _, stderr := execute(t, tc.args); !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: stderr = %q", tc.args, stderr)
		}
	}
}

// Without --yes, nothing is deleted unless the answer is yes
func TestRepoCommands_DeleteAsks(t *testing.T) {
	deletes := 0
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/me/web": testutil.Reply(`{"id":1,"owner":{"login":"me"},"name":"web","full_name":"me/web"}`),
		"DELETE /api/v1/repos/me/web": func(w http.ResponseWriter, r *http.Request) {
			deletes++
			w.WriteHeader(http.StatusNoContent)
		},
	})
	if out, _ := execute(t, []string{"repo", "rm", "-o", "me", "web"}, "n"); !strings.Contains(out, "Aborted") || deletes != 0 {
		t.Errorf("no: output = %q, %d deletes", out, deletes)
	}
	if out, _ := execute(t, []string{"repo", "rm", "-o", "me", "web"}, "y"); !strings.Contains(out, "Deleted repository me/web") || deletes != 1 {
		t.Errorf("yes: output = %q, %d deletes", out, deletes)
	}
	if out, _ := execute(t, []string{"repo", "rm", "-y", "-o", "me", "web"}); !strings.Contains(out, "Deleted repository me/web") || deletes != 2 {
		t.Errorf("-y: output = %q, %d deletes", out, deletes)
	}
}

func TestRepoCommands(t *testing.T) {
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/repos": testutil.Reply(`[{"id":1,"owner":{"login":"me"},"name":"web","full_name":"me/web"}]`),
		"GET /api/v1/repos/me/web": testutil.Reply(`{"id":1,"owner":{"login":"me"},"name":"web","full_name":"me/web",
			"default_branch":"develop"}`),
		"PATCH /api/v1/repos/me/web": testutil.Reply(`{"id":1,"owner":{"login":"me"},"name":"web","full_name":"me/web"}`),
	})
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"repo", "ls"}, "Number of repositories on test: 1"},
		{[]string{"repo", "ls", "--json"}, `"fullName": "me/web"`},
		{[]string{"repo", "info", "-o", "me", "web"}, "Default branch:"},
		{[]string{"repo", "info", "-o", "me", "nope"}, "Unable to fetch repo"},
		{[]string{"repo", "ls", "-q"}, "me/web\n"},
		{[]string{"repo", "rm", "-y", "-o", "me", "web"}, "Unable to delete the repository me/web"},
		{[]string{"repo", "rm", "-o", "me", "nope"}, "No such repository"},
		{[]string{"repo", "edit", "web"}, "Nothing to edit"},
		{[]string{"repo", "edit", "-o", "me", "--public=false", "--default-branch", "main", "web"}, "Updated repository me/web"},
	} {
		if out, _ := execute(t, tc.args); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}
