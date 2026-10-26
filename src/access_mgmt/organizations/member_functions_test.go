// gitclt
// Tests org members, org ismember and RemoveMemberFromOrg against a fake Forgejo

package organizations

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const acmeMembers = `[{"id":4,"login":"jfg","full_name":"J.F. Gratton","email":"jfg@example.com"},{"id":9,"login":"zed"}]`

// A fake Forgejo where jfg and zed are the members of acme, and no other org exists
func membersServer(t *testing.T) *int {
	t.Helper()
	removed := new(int)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/orgs/acme":             testutil.Reply(acmeOrg),
		"GET /api/v1/orgs/acme/members":     testutil.Reply(acmeMembers),
		"GET /api/v1/orgs/acme/members/jfg": testutil.Status(http.StatusNoContent),
		"DELETE /api/v1/orgs/acme/members/jfg": func(w http.ResponseWriter, r *http.Request) {
			*removed++
			w.WriteHeader(http.StatusNoContent)
		},
		// Anything else under /members/ is a 404, which is how Forgejo says "not a member"
		"GET /api/v1/orgs/broken/members/jfg": testutil.Status(http.StatusInternalServerError),
	})
	return removed
}

func TestListMembers(t *testing.T) {
	defaults(t)
	membersServer(t)

	members, err := ListMembers("acme", false)
	if err != nil || len(members) != 2 || members[1].Login != "zed" {
		t.Fatalf("got %+v, %v", members, err)
	}

	out := testutil.Run(t, func() { _, err = ListMembers("acme", true) })
	for _, want := range []string{"J.F. Gratton", "jfg@example.com", "zed"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListMembers("acme", true) }); err != nil || !strings.Contains(out, `"zed"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestIsMemberOfOrg(t *testing.T) {
	defaults(t)
	membersServer(t)

	for user, want := range map[string]bool{"jfg": true, "zed2": false} {
		var b bool
		var err *cerr.CustomError
		out := testutil.Run(t, func() { b, err = IsMemberOfOrg("acme", user) })
		if err != nil || b != want {
			t.Errorf("%s: got %t, %v", user, b, err)
		}
		if msg := map[bool]string{true: "jfg is part of acme", false: "zed2 is not part of acme"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%s: output = %q", user, out)
		}
	}

	// The library answers false for a missing org too: that one must be an error, not "not part of"
	if b, err := IsMemberOfOrg("nope", "jfg"); b || err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to fetch organization nope" {
		t.Errorf("missing org: got %t, %+v", b, err)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out := testutil.Run(t, func() { _, _ = IsMemberOfOrg("acme", "jfg") }); out != "" {
		t.Fatalf("quiet output = %q", out)
	}
}

func TestRemoveMemberFromOrg_Confirmation(t *testing.T) {
	for answer, want := range map[string]int{"": 0, "n": 0, "yess": 0, "y": 1, " Yes ": 1} {
		defaults(t)
		removed := membersServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = RemoveMemberFromOrg("acme", "jfg") }, answer)
		if err != nil || *removed != want {
			t.Errorf("%q: removed=%d, %v", answer, *removed, err)
		}
		if msg := map[int]string{0: "Aborted, jfg is still a member of acme", 1: "Removed jfg from acme"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%q: output = %q", answer, out)
		}
	}
}

func TestRemoveMemberFromOrg(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	removed := membersServer(t)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = RemoveMemberFromOrg("acme", "jfg") })
	if err != nil || *removed != 1 || !strings.Contains(out, "Removed jfg from acme") {
		t.Fatalf("removed=%d, %v, %q", *removed, err, out)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = RemoveMemberFromOrg("acme", "jfg") }); err != nil || out != "" {
		t.Fatalf("quiet: %v, %q", err, out)
	}
}

func TestMemberFunctions_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	for name, call := range map[string]func() *cerr.CustomError{
		"members":  func() *cerr.CustomError { _, e := ListMembers("acme", false); return e },
		"ismember": func() *cerr.CustomError { _, e := IsMemberOfOrg("acme", "jfg"); return e },
		"remove":   func() *cerr.CustomError { return RemoveMemberFromOrg("acme", "jfg") },
	} {
		testutil.WriteRegistry(t, "")
		if err := call(); err == nil {
			t.Errorf("%s: expected an error without a connection", name)
		}
		testutil.WriteRegistry(t, githubRegistry)
		if err := call(); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: github: got %+v", name, err)
		}
	}

	membersServer(t)
	if _, err := ListMembers("nope", false); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to fetch the member list of organization nope" {
		t.Errorf("members: got %+v", err)
	}
	// Only a 404 means "not a member"; any other error is reported, with its status
	if _, err := IsMemberOfOrg("broken", "jfg"); err == nil || err.Code != http.StatusInternalServerError {
		t.Errorf("ismember: got %+v", err)
	}
	if err := RemoveMemberFromOrg("acme", "zed"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to remove user zed from acme" {
		t.Errorf("remove: got %+v", err)
	}
}
