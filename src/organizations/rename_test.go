// gitclt
// Tests org rename against a fake Forgejo

package organizations

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestRenameOrg(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/orgs/acme/rename": testutil.Record(t, &bodies, http.StatusNoContent, "")})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = RenameOrg("acme", "wile") })
	if err != nil || len(bodies) != 1 || bodies[0]["new_name"] != "wile" {
		t.Fatalf("sent %v, %v", bodies, err)
	}
	if !strings.Contains(out, "Renamed organization acme to wile") {
		t.Fatalf("output = %q", out)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = RenameOrg("acme", "wile") }); err != nil || out != "" {
		t.Fatalf("quiet: %v, %q", err, out)
	}
}

func TestRenameOrg_Errors(t *testing.T) {
	defaults(t)

	testutil.WriteRegistry(t, "")
	if err := RenameOrg("acme", "wile"); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := RenameOrg("acme", "wile"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if err := RenameOrg("nope", "wile"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to rename the organization" {
		t.Fatalf("got %+v", err)
	}
	// Refused by the library before any request
	if err := RenameOrg("acme", ""); err == nil {
		t.Fatal("expected an error on an empty new name")
	}
}
