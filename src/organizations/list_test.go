// gitclt
// Tests org ls / org listall against a fake Forgejo; also holds what the other organizations tests share

package organizations

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const acmeOrg = `{"id":2,"name":"acme","full_name":"ACME Inc.","description":"Widgets","website":"https://acme.example.com",
	"location":"Montreal","email":"info@acme.example.com","visibility":"limited","repo_admin_change_team_access":true}`

const twoOrgs = `[` + acmeOrg + `,{"id":3,"name":"umbrella","full_name":"Umbrella Corp","visibility":"private"}]`

// Resets the package flags to their defaults for the duration of the test
func defaults(t *testing.T) {
	t.Helper()
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &CreateOrgRepoAdminChangeTeamAccess,
		&EditOrgRepoAdminChangeTeamAccess, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	for _, s := range []*string{&CreateOrgOwner, &CreateOrgFullName, &CreateOrgDescription, &CreateOrgWebsite,
		&CreateOrgEmail, &CreateOrgLocation, &EditOrgFullName, &EditOrgDescription, &EditOrgWebsite,
		&EditOrgEmail, &EditOrgLocation} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &CreateOrgVisibility, "")
	testutil.Set(t, &EditOrgVisibility, "")
}

const githubRegistry = `{"auths":{"gh":{"url":"https://github.com","token":"x","servertype":"github"}},"default_connection":"gh"}`

func TestListOrgs(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/orgs":      testutil.Reply(`[` + acmeOrg + `]`),
		"GET /api/v1/users/jfg/orgs": testutil.Reply(twoOrgs),
	})

	// No username: the authenticated user's orgs
	orgs, err := ListOrgs("", false)
	if err != nil || len(orgs) != 1 || orgs[0].Name != "acme" {
		t.Fatalf("mine: got %+v, %v", orgs, err)
	}
	orgs, err = ListOrgs("jfg", false)
	if err != nil || len(orgs) != 2 || orgs[1].Name != "umbrella" {
		t.Fatalf("jfg: got %+v, %v", orgs, err)
	}

	out := testutil.Run(t, func() { _, err = ListOrgs("jfg", true) })
	for _, want := range []string{"ACME Inc.", "Widgets", "Umbrella Corp", "private"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListOrgs("jfg", true) }); err != nil || !strings.Contains(out, `"umbrella"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestListAllOrgs(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/orgs": testutil.Reply(twoOrgs)})

	orgs, err := ListAllOrgs(false)
	if err != nil || len(orgs) != 2 {
		t.Fatalf("got %+v, %v", orgs, err)
	}

	out := testutil.Run(t, func() { _, err = ListAllOrgs(true) })
	if err != nil || !strings.Contains(out, "ACME Inc.") || !strings.Contains(out, "Umbrella Corp") {
		t.Fatalf("table: %v\n%s", err, out)
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListAllOrgs(true) }); err != nil || !strings.Contains(out, `"acme"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestListOrgs_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	for name, list := range map[string]func() *cerr.CustomError{
		"ls":      func() *cerr.CustomError { _, e := ListOrgs("nobody", false); return e },
		"listall": func() *cerr.CustomError { _, e := ListAllOrgs(false); return e },
	} {
		testutil.WriteRegistry(t, "")
		if err = list(); err == nil {
			t.Errorf("%s: expected an error without a connection", name)
		}
		testutil.WriteRegistry(t, githubRegistry)
		if err = list(); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: github: got %+v", name, err)
		}
		testutil.Forgejo(t, nil)
		if err = list(); err == nil || err.Code != http.StatusNotFound {
			t.Errorf("%s: got %+v", name, err)
		}
	}
}
