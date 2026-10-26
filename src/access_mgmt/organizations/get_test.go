// gitclt
// Tests org get against a fake Forgejo

package organizations

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestOrgGet(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/orgs/acme": testutil.Reply(acmeOrg)})

	o, err := OrgGet("acme", false)
	if err != nil || o.FullName != "ACME Inc." || !o.RepoAdminChangeTeamAccess {
		t.Fatalf("got %+v, %v", o, err)
	}

	out := testutil.Run(t, func() { _, err = OrgGet("acme", true) })
	for _, want := range []string{"Full name:", "ACME Inc.", "https://acme.example.com", "info@acme.example.com", "limited",
		"Repo owners can change team access:", "true"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = OrgGet("acme", true) }); err != nil || !strings.Contains(out, `"fullName"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestOrgGet_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	testutil.WriteRegistry(t, "")
	if _, err = OrgGet("acme", false); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if _, err = OrgGet("acme", false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if _, err = OrgGet("nope", false); err == nil || err.Code != http.StatusNotFound || err.Title != "Unable to fetch organization nope" {
		t.Fatalf("got %+v", err)
	}
}
