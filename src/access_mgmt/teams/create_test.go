// gitclt
// Tests team create: the access sent, from --permission, --units or neither

package teams

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

// A fake Forgejo recording what is POSTed to acme's teams
func createServer(t *testing.T, status int) *[]map[string]any {
	t.Helper()
	bodies := &[]map[string]any{}
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/orgs/acme/teams": testutil.Record(t, bodies, status, devsTeam)})
	return bodies
}

func TestCreateTeam(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateTeamDescription, "Developers")
	testutil.Set(t, &CreateTeamIncludesAllRepos, true)
	testutil.Set(t, &CreateTeamUnits, "code=write, repo.issues=READ")
	bodies := createServer(t, http.StatusCreated)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateTeam("acme", "devs") })
	if err != nil || len(*bodies) != 1 {
		t.Fatalf("%d requests, %v", len(*bodies), err)
	}
	b := (*bodies)[0]
	units, _ := b["units_map"].(map[string]any)
	if b["name"] != "devs" || b["description"] != "Developers" || b["includes_all_repositories"] != true ||
		len(units) != 2 || units["repo.code"] != "write" || units["repo.issues"] != "read" {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Created team devs in acme") {
		t.Fatalf("output = %q", out)
	}
}

// No --permission nor --units means read on every unit; --permission admin is sent as is
func TestCreateTeam_Permission(t *testing.T) {
	for perm, want := range map[PermissionFlag]string{"": "read", "write": "write", "admin": ""} {
		defaults(t)
		testutil.Set(t, &shared.QuietOutput, true)
		testutil.Set(t, &CreateTeamPermission, perm)
		bodies := createServer(t, http.StatusCreated)

		if err := CreateTeam("acme", "devs"); err != nil || len(*bodies) != 1 {
			t.Fatalf("%q: %d requests, %v", perm, len(*bodies), err)
		}
		b := (*bodies)[0]
		units, _ := b["units_map"].(map[string]any)
		if want == "" {
			if b["permission"] != "admin" || len(units) != 0 {
				t.Errorf("admin: sent %v", b)
			}
			continue
		}
		if len(units) != 10 || units["repo.wiki"] != want {
			t.Errorf("%q: sent %v", perm, b)
		}
	}
}

func TestCreateTeam_Errors(t *testing.T) {
	defaults(t)

	// Checked before connecting: no registry needed
	testutil.WriteRegistry(t, "")
	testutil.Set(t, &CreateTeamPermission, "write")
	testutil.Set(t, &CreateTeamUnits, "code=read")
	if err := CreateTeam("acme", "devs"); err == nil || err.Title != "Invalid team access" {
		t.Fatalf("both: got %+v", err)
	}
	testutil.Set(t, &CreateTeamPermission, "")
	testutil.Set(t, &CreateTeamUnits, "code=admin")
	if err := CreateTeam("acme", "devs"); err == nil || err.Title != "Invalid unit" {
		t.Fatalf("unit: got %+v", err)
	}

	testutil.Set(t, &CreateTeamUnits, "")
	if err := CreateTeam("acme", "devs"); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := CreateTeam("acme", "devs"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	createServer(t, http.StatusUnprocessableEntity)
	if err := CreateTeam("acme", "devs"); err == nil || err.Code != http.StatusUnprocessableEntity ||
		err.Title != "Unable to create the team" {
		t.Fatalf("got %+v", err)
	}
}
