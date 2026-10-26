// gitclt
// Tests team ls against a fake Forgejo; also holds what the other teams tests share

package teams

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const devsTeam = `{"id":7,"name":"devs","description":"Developers","organization":{"username":"acme"},"permission":"write",
	"units_map":{"repo.code":"write","repo.issues":"read"},"includes_all_repositories":true,"can_create_org_repo":false}`

const ownersTeam = `{"id":1,"name":"Owners","organization":{"username":"acme"},"permission":"owner"}`

const twoTeams = `[` + ownersTeam + `,` + devsTeam + `]`

const githubRegistry = `{"auths":{"gh":{"url":"https://github.com","token":"x","servertype":"github"}},"default_connection":"gh"}`

// Resets the package flags to their defaults for the duration of the test
func defaults(t *testing.T) {
	t.Helper()
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &CreateTeamIncludesAllRepos, &CreateTeamCanCreateOrgRepo,
		&EditTeamIncludesAllRepos, &EditTeamCanCreateOrgRepo, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	for _, s := range []*string{&CreateTeamDescription, &CreateTeamUnits, &EditTeamName, &EditTeamDescription, &EditTeamUnits} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &CreateTeamPermission, "")
	testutil.Set(t, &EditTeamPermission, "")
}

// The routes every team lookup needs: acme has the Owners and devs teams, no other org exists
// GetByName goes through the search endpoint, and filters on the name itself
func teamRoutes(extra map[string]http.HandlerFunc) map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		"GET /api/v1/orgs/acme/teams":        testutil.Reply(twoTeams),
		"GET /api/v1/orgs/acme/teams/search": testutil.Reply(`{"ok":true,"data":` + twoTeams + `}`),
		"GET /api/v1/teams/1":                testutil.Reply(ownersTeam),
		"GET /api/v1/teams/7":                testutil.Reply(devsTeam),
	}
	for k, v := range extra {
		routes[k] = v
	}
	return routes
}

func TestListTeams(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, teamRoutes(nil))

	list, err := ListTeams("acme", false)
	if err != nil || len(list) != 2 || list[1].Name != "devs" || list[1].Org != "acme" {
		t.Fatalf("got %+v, %v", list, err)
	}

	out := testutil.Run(t, func() { _, err = ListTeams("acme", true) })
	for _, want := range []string{"Owners", "owner", "devs", "Developers", "write"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListTeams("acme", true) }); err != nil || !strings.Contains(out, `"devs"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestListTeams_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	testutil.WriteRegistry(t, "")
	if _, err = ListTeams("acme", false); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if _, err = ListTeams("acme", false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if _, err = ListTeams("nope", false); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to fetch the team list of organization nope" {
		t.Fatalf("got %+v", err)
	}
}
