// gitclt
// Tests the org commands' wiring against a fake Forgejo

package cmd

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestOrgCommands(t *testing.T) {
	org := `{"id":2,"name":"acme","full_name":"ACME Inc."}`
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/orgs":                testutil.Reply(`[` + org + `]`),
		"GET /api/v1/users/jfg/orgs":           testutil.Reply(`[]`),
		"GET /api/v1/orgs":                     testutil.Reply(`[` + org + `]`),
		"GET /api/v1/orgs/acme":                testutil.Reply(org),
		"DELETE /api/v1/orgs/acme":             testutil.Status(http.StatusNoContent),
		"POST /api/v1/orgs":                    testutil.Status(http.StatusCreated),
		"POST /api/v1/orgs/acme/rename":        testutil.Status(http.StatusNoContent),
		"GET /api/v1/orgs/acme/members":        testutil.Reply(`[` + jfg + `]`),
		"GET /api/v1/orgs/acme/members/jfg":    testutil.Status(http.StatusNoContent),
		"DELETE /api/v1/orgs/acme/members/jfg": testutil.Status(http.StatusNoContent),
	})
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"org", "ls"}, nil, "ACME Inc."},
		{[]string{"org", "ls", "--json", "jfg"}, nil, "[]"},
		{[]string{"org", "listall"}, nil, "ACME Inc."},
		{[]string{"org", "get", "acme"}, nil, "Full name:"},
		{[]string{"org", "get", "nope"}, nil, "Unable to fetch"},
		{[]string{"org", "rm", "acme"}, []string{"n"}, "Aborted"},
		{[]string{"org", "rm", "--yes", "acme"}, nil, "Deleted org acme"},
		{[]string{"org", "create", "acme"}, nil, "Created organization acme"},
		{[]string{"org", "edit", "acme"}, nil, "Nothing to edit"},
		{[]string{"org", "mv", "acme", "wile"}, nil, "Renamed organization acme to wile"},
		{[]string{"org", "members", "acme"}, nil, "J.F. Gratton"},
		{[]string{"org", "members", "nope"}, nil, "Unable to fetch the member list of organization nope"},
		{[]string{"org", "ismember", "acme", "jfg"}, nil, "jfg is part of acme"},
		{[]string{"org", "ismember", "acme", "zed"}, nil, "zed is not part of acme"},
		{[]string{"org", "ismember", "nope", "jfg"}, nil, "Unable to fetch organization nope"},
		{[]string{"org", "removemember", "acme", "jfg"}, []string{"n"}, "Aborted"},
		{[]string{"org", "rmmember", "-y", "acme", "jfg"}, nil, "Removed jfg from acme"},
	} {
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

// --repo-admin-team-access=false must be sent (as false); flags not passed must not be sent at all
func TestOrgEdit_ChangedFlags(t *testing.T) {
	var sent [][]string
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/orgs/acme": keysRecorder(t, &sent, http.StatusOK)})

	out, _ := execute(t, []string{"org", "edit", "--repo-admin-team-access=false", "-l", "Montreal", "acme"})
	if !strings.Contains(out, "Updated organization acme") || len(sent) != 1 ||
		!slices.Equal(sent[0], []string{"location", "repo_admin_change_team_access"}) {
		t.Fatalf("sent %v, output %q", sent, out)
	}
}

// org get / org rm used to accept no argument, then crash on args[0]
func TestOrgCommands_Arguments(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, args := range [][]string{{"org", "get"}, {"org", "rm"}, {"org", "ls", "a", "b"}, {"org", "listall", "a"},
		{"org", "create"}, {"org", "edit"}, {"org", "mv", "a"}, {"org", "members"}, {"org", "ismember", "a"},
		{"org", "removemember", "a"}} {
		if _, stderr := execute(t, args); !strings.Contains(stderr, "arg(s)") && !strings.Contains(stderr, "unknown command") {
			t.Errorf("%v: stderr = %q", args, stderr)
		}
	}
	if _, stderr := execute(t, []string{"org", "create", "-v", "secret", "acme"}); !strings.Contains(stderr, "must be one of public, limited or private") {
		t.Errorf("visibility: stderr = %q", stderr)
	}
}
