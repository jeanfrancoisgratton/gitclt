// gitclt
// Tests the team commands' wiring against a fake Forgejo

package cmd

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestTeamCommands(t *testing.T) {
	team := `{"id":7,"name":"devs","description":"Developers","permission":"write"}`
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/orgs/acme/teams":               testutil.Reply(`[` + team + `]`),
		"GET /api/v1/orgs/acme/teams/search":        testutil.Reply(`{"ok":true,"data":[` + team + `]}`),
		"GET /api/v1/teams/7":                       testutil.Reply(team),
		"POST /api/v1/orgs/acme/teams":              testutil.Status(http.StatusCreated),
		"DELETE /api/v1/teams/7":                    testutil.Status(http.StatusNoContent),
		"GET /api/v1/teams/7/members":               testutil.Reply(`[` + jfg + `]`),
		"GET /api/v1/teams/7/members/jfg":           testutil.Status(http.StatusNoContent),
		"PUT /api/v1/teams/7/members/jfg":           testutil.Status(http.StatusNoContent),
		"DELETE /api/v1/teams/7/members/jfg":        testutil.Status(http.StatusNoContent),
		"GET /api/v1/teams/7/repos":                 testutil.Reply(`[{"id":11,"full_name":"acme/widgets"}]`),
		"PUT /api/v1/teams/7/repos/acme/widgets":    testutil.Status(http.StatusNoContent),
		"DELETE /api/v1/teams/7/repos/acme/widgets": testutil.Status(http.StatusNoContent),
	})
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"team", "ls", "acme"}, nil, "Developers"},
		{[]string{"team", "ls", "--json", "acme"}, nil, `"devs"`},
		{[]string{"team", "ls", "nope"}, nil, "Unable to fetch the team list of organization nope"},
		{[]string{"team", "get", "acme", "devs"}, nil, "Permission:"},
		{[]string{"team", "get", "7"}, nil, "Developers"},
		{[]string{"team", "get", "devs"}, nil, `"devs" is not a team ID`},
		{[]string{"team", "get", "acme", "ghosts"}, nil, "Unable to fetch team ghosts of acme"},
		{[]string{"team", "create", "acme", "devs"}, nil, "Created team devs in acme"},
		{[]string{"team", "create", "-p", "write", "-u", "code=read", "acme", "devs"}, nil, "use either --permission or --units"},
		{[]string{"team", "edit", "acme", "devs"}, nil, "Nothing to edit"},
		{[]string{"team", "rm", "acme", "devs"}, []string{"n"}, "Aborted"},
		{[]string{"team", "rm", "--yes", "acme", "devs"}, nil, "Deleted team devs of acme"},
		{[]string{"team", "members", "acme", "devs"}, nil, "J.F. Gratton"},
		{[]string{"team", "ismember", "acme", "devs", "jfg"}, nil, "jfg is part of acme/devs"},
		{[]string{"team", "ismember", "acme", "devs", "zed"}, nil, "zed is not part of acme/devs"},
		{[]string{"team", "addmember", "acme", "devs", "jfg"}, nil, "Added jfg to acme/devs"},
		{[]string{"team", "removemember", "acme", "devs", "jfg"}, []string{"n"}, "Aborted"},
		{[]string{"team", "rmmember", "-y", "acme", "devs", "jfg"}, nil, "Removed jfg from acme/devs"},
		{[]string{"team", "repos", "acme", "devs"}, nil, "acme/widgets"},
		{[]string{"team", "addrepo", "acme", "devs", "widgets"}, nil, "Team devs now has access to acme/widgets"},
		{[]string{"team", "rmrepo", "-y", "acme", "devs", "widgets"}, nil, "Team devs no longer has access to acme/widgets"},
	} {
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

// --all-repos=false must be sent (as false); flags not passed must not be sent at all
func TestTeamEdit_ChangedFlags(t *testing.T) {
	var sent [][]string
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/orgs/acme/teams/search": testutil.Reply(`{"ok":true,"data":[{"id":7,"name":"devs"}]}`),
		"GET /api/v1/teams/7":                testutil.Reply(`{"id":7,"name":"devs"}`),
		"PATCH /api/v1/teams/7":              keysRecorder(t, &sent, http.StatusOK),
	})

	out, _ := execute(t, []string{"team", "edit", "--all-repos=false", "-d", "Coders", "acme", "devs"})
	if !strings.Contains(out, "Updated team devs of acme") || len(sent) != 1 ||
		!slices.Equal(sent[0], []string{"description", "includes_all_repositories", "name"}) {
		t.Fatalf("sent %v, output %q", sent, out)
	}
}

func TestTeamCommands_Arguments(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, args := range [][]string{{"team", "ls"}, {"team", "ls", "a", "b"}, {"team", "get"}, {"team", "get", "a", "b", "c"},
		{"team", "create", "a"}, {"team", "edit", "a"}, {"team", "rm", "a"}, {"team", "members", "a"},
		{"team", "ismember", "a", "b"}, {"team", "addmember", "a", "b"}, {"team", "removemember", "a", "b"},
		{"team", "repos", "a"}, {"team", "addrepo", "a", "b"}, {"team", "removerepo", "a", "b"}} {
		if _, stderr := execute(t, args); !strings.Contains(stderr, "arg(s)") && !strings.Contains(stderr, "unknown command") {
			t.Errorf("%v: stderr = %q", args, stderr)
		}
	}
	if _, stderr := execute(t, []string{"team", "create", "-p", "owner", "acme", "devs"}); !strings.Contains(stderr, "must be one of read, write or admin") {
		t.Errorf("permission: stderr = %q", stderr)
	}
}
