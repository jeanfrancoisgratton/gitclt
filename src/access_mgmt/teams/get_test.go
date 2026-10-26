// gitclt
// Tests team get, by name and by ID, against a fake Forgejo

package teams

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestTeamGet(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, teamRoutes(nil))

	// The name is matched case-insensitively
	tm, err := TeamGet("acme", "DEVS", false)
	if err != nil || tm.ID != 7 || !tm.IncludesAllRepos || tm.Units["repo.issues"] != "read" {
		t.Fatalf("got %+v, %v", tm, err)
	}

	out := testutil.Run(t, func() { _, err = TeamGet("acme", "devs", true) })
	for _, want := range []string{"Organization:", "acme", "Developers", "Access to all repos:", "true", "code", "issues"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Units are sorted, and shown without their repo. prefix
	if strings.Index(out, "code") > strings.Index(out, "issues") || strings.Contains(out, "repo.code") {
		t.Errorf("units:\n%s", out)
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = TeamGet("acme", "devs", true) }); err != nil || !strings.Contains(out, `"Developers"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestTeamGetByID(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, teamRoutes(nil))

	tm, err := TeamGetByID(1, false)
	if err != nil || tm.Name != "Owners" || tm.Permission != "owner" {
		t.Fatalf("got %+v, %v", tm, err)
	}
	if out := testutil.Run(t, func() { _, err = TeamGetByID(7, true) }); err != nil || !strings.Contains(out, "Developers") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestTeamGet_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	for name, get := range map[string]func() *cerr.CustomError{
		"name": func() *cerr.CustomError { _, e := TeamGet("acme", "devs", false); return e },
		"id":   func() *cerr.CustomError { _, e := TeamGetByID(7, false); return e },
	} {
		testutil.WriteRegistry(t, "")
		if err = get(); err == nil {
			t.Errorf("%s: expected an error without a connection", name)
		}
		testutil.WriteRegistry(t, githubRegistry)
		if err = get(); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: github: got %+v", name, err)
		}
	}

	testutil.Forgejo(t, teamRoutes(nil))
	// A missing team in an existing org, and a missing org, are both 404s
	for _, org := range []string{"acme", "nope"} {
		if _, err = TeamGet(org, "ghosts", false); err == nil || err.Code != http.StatusNotFound ||
			err.Title != "Unable to fetch team ghosts of "+org {
			t.Errorf("%s: got %+v", org, err)
		}
	}
	if _, err = TeamGetByID(99, false); err == nil || err.Code != http.StatusNotFound || err.Title != "Unable to fetch team #99" {
		t.Errorf("id: got %+v", err)
	}
}
