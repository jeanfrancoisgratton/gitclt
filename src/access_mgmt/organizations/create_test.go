// gitclt
// Tests org create: the request body, and which endpoint --owner leads to

package organizations

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func keys(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
}

// A fake Forgejo where "me" is logged in, recording what is POSTed to either create endpoint
func createServer(t *testing.T, status int) (mine, admin *[]map[string]any) {
	t.Helper()
	mine, admin = &[]map[string]any{}, &[]map[string]any{}
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user":                  testutil.Reply(`{"id":1,"login":"me"}`),
		"POST /api/v1/orgs":                 testutil.Record(t, mine, status, acmeOrg),
		"POST /api/v1/admin/users/jfg/orgs": testutil.Record(t, admin, status, acmeOrg),
	})
	return mine, admin
}

func TestCreateOrg(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateOrgFullName, "ACME Inc.")
	testutil.Set(t, &CreateOrgWebsite, "https://acme.example.com")
	testutil.Set(t, &CreateOrgVisibility, "limited")
	testutil.Set(t, &CreateOrgRepoAdminChangeTeamAccess, true)
	mine, admin := createServer(t, http.StatusCreated)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateOrg("acme") })
	if err != nil || len(*mine) != 1 || len(*admin) != 0 {
		t.Fatalf("mine=%v admin=%v, %v", *mine, *admin, err)
	}
	b := (*mine)[0]
	// Fields left empty are not sent
	if got := keys(b); !slices.Equal(got, []string{"full_name", "repo_admin_change_team_access", "username", "visibility", "website"}) {
		t.Fatalf("sent %v", got)
	}
	if b["username"] != "acme" || b["visibility"] != "limited" || b["repo_admin_change_team_access"] != true {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Created organization acme") {
		t.Fatalf("output = %q", out)
	}
}

// --owner someone else goes through the admin endpoint; --owner yourself (any case) does not
func TestCreateOrg_Owner(t *testing.T) {
	for owner, wantAdmin := range map[string]bool{"jfg": true, "me": false, "ME": false} {
		defaults(t)
		testutil.Set(t, &CreateOrgOwner, owner)
		testutil.Set(t, &shared.QuietOutput, true)
		mine, admin := createServer(t, http.StatusCreated)

		if out := testutil.Run(t, func() {
			if err := CreateOrg("acme"); err != nil {
				t.Errorf("%s: %v", owner, err)
			}
		}); out != "" {
			t.Errorf("%s: quiet output = %q", owner, out)
		}
		if got := len(*admin) == 1 && len(*mine) == 0; got != wantAdmin {
			t.Errorf("%s: mine=%v admin=%v", owner, *mine, *admin)
		}
	}
}

func TestCreateOrg_Errors(t *testing.T) {
	defaults(t)

	testutil.WriteRegistry(t, "")
	if err := CreateOrg("acme"); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := CreateOrg("acme"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	createServer(t, http.StatusUnprocessableEntity)
	if err := CreateOrg("acme"); err == nil || err.Code != http.StatusUnprocessableEntity ||
		err.Title != "Unable to create the organization" {
		t.Fatalf("got %+v", err)
	}
}
