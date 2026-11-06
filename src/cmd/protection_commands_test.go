// gitclt
// Tests the branch protection commands' wiring: their arguments and flags, and -o reaching the right package

package cmd

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestProtectionCommands(t *testing.T) {
	// Everything here fails before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"branch", "protection", "ls", "a", "b"}, "accepts at most 1 arg(s)"},
		{[]string{"branch", "prot", "get"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"branch", "prot", "add"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"branch", "prot", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"branch", "prot", "edit", "web", "main"}, "Nothing to edit"},
		{[]string{"branch", "prot", "edit", "--no-push=false", "web", "main"}, "Nothing to edit"},
		{[]string{"branch", "prot", "add", "--approvals", "-1", "web", "main"}, "cannot be negative"},
		{[]string{"branch", "prot", "add", "--no-push", "--push-users", "alice", "web", "main"}, "none of the others can be"},
		{[]string{"branch", "prot", "edit", "--any-push", "--no-push", "web", "main"}, "none of the others can be"},
		{[]string{"branch", "prot", "edit", "--any-merge", "--merge-teams", "ops", "web", "main"}, "none of the others can be"},
		// There is nothing to remove on a new rule
		{[]string{"branch", "prot", "add", "--any-push", "web", "main"}, "unknown flag: --any-push"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestProtectionCommands_Owner(t *testing.T) {
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/other/web":                    testutil.Reply(`{"name":"web","owner":{"login":"other"}}`),
		"GET /api/v1/repos/other/web/branch_protections": testutil.Reply(`[{"rule_name":"main"}]`),
	})
	out, stderr := execute(t, []string{"branch", "prot", "ls", "-q", "-o", "other", "web"})
	if out != "main\n" {
		t.Errorf("output = %q, stderr = %q", out, stderr)
	}
}

func TestProtectionCommands_EditPassesTheChangedFlags(t *testing.T) {
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                           testutil.Reply(`{"name":"web","owner":{"login":"acme"}}`),
		"GET /api/v1/repos/acme/web/branch_protections/main":   testutil.Reply(`{"rule_name":"main","required_approvals":2,"require_signed_commits":true,"enable_push":true}`),
		"PATCH /api/v1/repos/acme/web/branch_protections/main": testutil.Record(t, &bodies, http.StatusOK, `{"rule_name":"main"}`),
	})
	out, stderr := execute(t, []string{"branch", "prot", "edit", "--signed=false", "web", "main"})
	if len(bodies) != 1 || bodies[0]["require_signed_commits"] != false || bodies[0]["required_approvals"] != float64(2) {
		t.Errorf("bodies = %v, output = %q, stderr = %q", bodies, out, stderr)
	}
}
