// gitclt
// Tests team members, ismember, addmember and removemember against a fake Forgejo

package teams

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const devsMembers = `[{"id":4,"login":"jfg","full_name":"J.F. Gratton","email":"jfg@example.com"},{"id":9,"login":"zed"}]`

// A fake Forgejo where jfg and zed are the members of acme/devs; it counts the add and remove requests
func membersServer(t *testing.T) (added, removed *int) {
	t.Helper()
	added, removed = new(int), new(int)
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"GET /api/v1/teams/7/members":     testutil.Reply(devsMembers),
		"GET /api/v1/teams/7/members/jfg": testutil.Status(http.StatusNoContent),
		"PUT /api/v1/teams/7/members/bob": func(w http.ResponseWriter, r *http.Request) {
			*added++
			w.WriteHeader(http.StatusNoContent)
		},
		"DELETE /api/v1/teams/7/members/jfg": func(w http.ResponseWriter, r *http.Request) {
			*removed++
			w.WriteHeader(http.StatusNoContent)
		},
		// Anything else under /members/ is a 404, which is how Forgejo says "not a member"
		"GET /api/v1/teams/7/members/broken": testutil.Status(http.StatusInternalServerError),
	}))
	return added, removed
}

func TestListMembers(t *testing.T) {
	defaults(t)
	membersServer(t)

	members, err := ListMembers("acme", "devs", false)
	if err != nil || len(members) != 2 || members[1].Login != "zed" {
		t.Fatalf("got %+v, %v", members, err)
	}

	out := testutil.Run(t, func() { _, err = ListMembers("acme", "devs", true) })
	for _, want := range []string{"J.F. Gratton", "jfg@example.com", "zed"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListMembers("acme", "devs", true) }); err != nil || !strings.Contains(out, `"zed"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestIsMemberOfTeam(t *testing.T) {
	defaults(t)
	membersServer(t)

	for user, want := range map[string]bool{"jfg": true, "zed2": false} {
		var b bool
		var err *cerr.CustomError
		out := testutil.Run(t, func() { b, err = IsMemberOfTeam("acme", "devs", user) })
		if err != nil || b != want {
			t.Errorf("%s: got %t, %v", user, b, err)
		}
		if msg := map[bool]string{true: "jfg is part of acme/devs", false: "zed2 is not part of acme/devs"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%s: output = %q", user, out)
		}
	}

	// The team is resolved first: a missing team is an error, not "not part of"
	if b, err := IsMemberOfTeam("acme", "ghosts", "jfg"); b || err == nil || err.Code != http.StatusNotFound {
		t.Errorf("missing team: got %t, %+v", b, err)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out := testutil.Run(t, func() { _, _ = IsMemberOfTeam("acme", "devs", "jfg") }); out != "" {
		t.Fatalf("quiet output = %q", out)
	}
}

func TestAddMemberToTeam(t *testing.T) {
	defaults(t)
	added, _ := membersServer(t)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = AddMemberToTeam("acme", "devs", "bob") })
	if err != nil || *added != 1 || !strings.Contains(out, "Added bob to acme/devs") {
		t.Fatalf("added=%d, %v, %q", *added, err, out)
	}
}

func TestRemoveMemberFromTeam_Confirmation(t *testing.T) {
	for answer, want := range map[string]int{"": 0, "n": 0, "yess": 0, "y": 1, " Yes ": 1} {
		defaults(t)
		_, removed := membersServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = RemoveMemberFromTeam("acme", "devs", "jfg") }, answer)
		if err != nil || *removed != want {
			t.Errorf("%q: removed=%d, %v", answer, *removed, err)
		}
		if msg := map[int]string{0: "Aborted, jfg is still a member of acme/devs", 1: "Removed jfg from acme/devs"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%q: output = %q", answer, out)
		}
	}
}

func TestRemoveMemberFromTeam_YesAndQuiet(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &shared.QuietOutput, true)
	_, removed := membersServer(t)

	if out := testutil.Run(t, func() { _ = RemoveMemberFromTeam("acme", "devs", "jfg") }); *removed != 1 || out != "" {
		t.Fatalf("removed=%d, %q", *removed, out)
	}
}

func TestMemberFunctions_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	for name, call := range map[string]func() *cerr.CustomError{
		"members":   func() *cerr.CustomError { _, e := ListMembers("acme", "devs", false); return e },
		"ismember":  func() *cerr.CustomError { _, e := IsMemberOfTeam("acme", "devs", "jfg"); return e },
		"addmember": func() *cerr.CustomError { return AddMemberToTeam("acme", "devs", "bob") },
		"remove":    func() *cerr.CustomError { return RemoveMemberFromTeam("acme", "devs", "jfg") },
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
	if _, err := ListMembers("nope", "devs", false); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to fetch team devs of nope" {
		t.Errorf("members: got %+v", err)
	}
	// Only a 404 means "not a member"; any other error is reported, with its status
	if _, err := IsMemberOfTeam("acme", "devs", "broken"); err == nil || err.Code != http.StatusInternalServerError {
		t.Errorf("ismember: got %+v", err)
	}
	if err := AddMemberToTeam("acme", "devs", "nobody"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to add user nobody to acme/devs" {
		t.Errorf("add: got %+v", err)
	}
	if err := RemoveMemberFromTeam("acme", "devs", "zed"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to remove user zed from acme/devs" {
		t.Errorf("remove: got %+v", err)
	}
}
