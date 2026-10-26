// gitclt
// Tests org edit: only the flags that were passed are sent

package organizations

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Stands in for cmd.Flags().Changed: only the named flags were passed
func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

func TestEditOrg_OnlyPassedFlagsAreSent(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditOrgLocation, "Montreal")
	testutil.Set(t, &EditOrgDescription, "Gadgets") // set, but not passed: must not be sent
	testutil.Set(t, &EditOrgVisibility, "private")
	testutil.Set(t, &EditOrgRepoAdminChangeTeamAccess, false)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/orgs/acme": testutil.Record(t, &bodies, http.StatusOK, acmeOrg)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() {
		err = EditOrg("acme", passed("location", "website", "visibility", "repo-admin-team-access"))
	})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("%d requests, %v", len(bodies), err)
	}
	b := bodies[0]
	if got := keys(b); !slices.Equal(got, []string{"location", "repo_admin_change_team_access", "visibility", "website"}) {
		t.Fatalf("sent %v", got)
	}
	// --website "" clears the field, --repo-admin-team-access=false revokes it: both must be sent, not dropped
	if b["location"] != "Montreal" || b["website"] != "" || b["visibility"] != "private" || b["repo_admin_change_team_access"] != false {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Updated organization acme") {
		t.Fatalf("output = %q", out)
	}
}

func TestEditOrg_Errors(t *testing.T) {
	defaults(t)

	// Checked before connecting: no registry needed
	testutil.WriteRegistry(t, "")
	if err := EditOrg("acme", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: got %+v", err)
	}
	if err := EditOrg("acme", passed("email")); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := EditOrg("acme", passed("email")); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if err := EditOrg("nope", passed("email")); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to edit the organization" {
		t.Fatalf("got %+v", err)
	}
}
