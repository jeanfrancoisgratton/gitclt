// gitclt
// Tests the org commands' wiring against a fake Forgejo

package cmd

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestOrgCommands(t *testing.T) {
	org := `{"id":2,"name":"acme","full_name":"ACME Inc."}`
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/orgs":      testutil.Reply(`[` + org + `]`),
		"GET /api/v1/users/jfg/orgs": testutil.Reply(`[]`),
		"GET /api/v1/orgs":           testutil.Reply(`[` + org + `]`),
		"GET /api/v1/orgs/acme":      testutil.Reply(org),
		"DELETE /api/v1/orgs/acme":   testutil.Status(http.StatusNoContent),
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
	} {
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

// org get / org rm used to accept no argument, then crash on args[0]
func TestOrgCommands_Arguments(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, args := range [][]string{{"org", "get"}, {"org", "rm"}, {"org", "ls", "a", "b"}, {"org", "listall", "a"}} {
		if _, stderr := execute(t, args); !strings.Contains(stderr, "arg(s)") && !strings.Contains(stderr, "unknown command") {
			t.Errorf("%v: stderr = %q", args, stderr)
		}
	}
}
