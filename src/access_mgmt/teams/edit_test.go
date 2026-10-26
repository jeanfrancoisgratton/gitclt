// gitclt
// Tests team edit: only the flags that were passed are sent, and the owner team is protected

package teams

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

func TestEditTeam_OnlyPassedFlagsAreSent(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditTeamDescription, "Gadgets") // set, but not passed: must not be sent
	testutil.Set(t, &EditTeamIncludesAllRepos, false)
	var bodies []map[string]any
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"PATCH /api/v1/teams/7": testutil.Record(t, &bodies, http.StatusOK, devsTeam),
	}))

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditTeam("acme", "devs", passed("all-repos")) })
	if err != nil || len(bodies) != 1 {
		t.Fatalf("%d requests, %v", len(bodies), err)
	}
	b := bodies[0]
	// The library always sends the current name, as Forgejo requires it
	if got := keys(b); !slices.Equal(got, []string{"includes_all_repositories", "name"}) {
		t.Fatalf("sent %v", got)
	}
	// --all-repos=false revokes: it must be sent, not dropped
	if b["name"] != "devs" || b["includes_all_repositories"] != false {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Updated team devs of acme") {
		t.Fatalf("output = %q", out)
	}
}

func TestEditTeam_Access(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditTeamName, "coders")
	testutil.Set(t, &EditTeamUnits, "wiki=none")
	var bodies []map[string]any
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"PATCH /api/v1/teams/7": testutil.Record(t, &bodies, http.StatusOK, devsTeam),
	}))

	if err := EditTeam("acme", "devs", passed("name", "units")); err != nil || len(bodies) != 1 {
		t.Fatalf("%d requests, %v", len(bodies), err)
	}
	units, _ := bodies[0]["units_map"].(map[string]any)
	if bodies[0]["name"] != "coders" || len(units) != 1 || units["repo.wiki"] != "none" {
		t.Fatalf("sent %v", bodies[0])
	}
}

func TestEditTeam_Errors(t *testing.T) {
	defaults(t)

	// Checked before connecting: no registry needed
	testutil.WriteRegistry(t, "")
	if err := EditTeam("acme", "devs", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: got %+v", err)
	}
	if err := EditTeam("acme", "devs", passed("units")); err == nil || err.Title != "Invalid team access" {
		t.Fatalf("empty units: got %+v", err)
	}
	if err := EditTeam("acme", "devs", passed("description")); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := EditTeam("acme", "devs", passed("description")); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}

	var bodies []map[string]any
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"PATCH /api/v1/teams/1": testutil.Record(t, &bodies, http.StatusOK, ownersTeam),
	}))
	if err := EditTeam("acme", "ghosts", passed("description")); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("missing team: got %+v", err)
	}
	// Renaming the owner team is refused by the library, without a request
	testutil.Set(t, &EditTeamName, "Bosses")
	if err := EditTeam("acme", "owners", passed("name")); err == nil || err.Title != "Unable to edit the team" || len(bodies) != 0 {
		t.Fatalf("owner team: sent %v, got %+v", bodies, err)
	}
}
