// gitclt
// Tests team delete: the confirmation prompt, --yes, and the owner team

package teams

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// A fake Forgejo counting the delete requests on either team of acme
func deleteServer(t *testing.T) *int {
	t.Helper()
	calls := new(int)
	count := func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.WriteHeader(http.StatusNoContent)
	}
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"DELETE /api/v1/teams/7": count,
		"DELETE /api/v1/teams/1": count,
	}))
	return calls
}

func TestDeleteTeam_Confirmation(t *testing.T) {
	for answer, want := range map[string]int{"": 0, "n": 0, "yess": 0, "y": 1, " Yes ": 1} {
		defaults(t)
		calls := deleteServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteTeam("acme", "devs") }, answer)
		if err != nil || *calls != want {
			t.Errorf("%q: calls=%d, %v", answer, *calls, err)
		}
		if msg := map[int]string{0: "Aborted, team devs was not deleted", 1: "Deleted team devs of acme"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%q: output = %q", answer, out)
		}
	}
}

func TestDeleteTeam_YesAndQuiet(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &shared.QuietOutput, true)
	calls := deleteServer(t)

	// No prompt (testutil.Run fails the test on an unanswered prompt), no output
	if out := testutil.Run(t, func() { _ = DeleteTeam("acme", "devs") }); *calls != 1 || out != "" {
		t.Fatalf("calls=%d, %q", *calls, out)
	}
}

func TestDeleteTeam_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	testutil.WriteRegistry(t, "")
	if err := DeleteTeam("acme", "devs"); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := DeleteTeam("acme", "devs"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	calls := deleteServer(t)
	if err := DeleteTeam("acme", "ghosts"); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("missing team: got %+v", err)
	}
	// The owner team is refused by the library, without a request
	if err := DeleteTeam("acme", "Owners"); err == nil || err.Title != "Unable to delete the team" || *calls != 0 {
		t.Fatalf("owner team: calls=%d, got %+v", *calls, err)
	}
}
