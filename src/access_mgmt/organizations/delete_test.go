// gitclt
// Tests org delete: the confirmation prompt and --yes

package organizations

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// A fake Forgejo counting the delete requests on acme
func deleteServer(t *testing.T) *int {
	t.Helper()
	calls := new(int)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/orgs/acme": func(w http.ResponseWriter, r *http.Request) {
			*calls++
			w.WriteHeader(http.StatusNoContent)
		},
	})
	return calls
}

func TestDeleteOrg_Confirmation(t *testing.T) {
	for answer, want := range map[string]int{"": 0, "n": 0, "yess": 0, "y": 1, " Yes ": 1} {
		defaults(t)
		calls := deleteServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteOrg("acme") }, answer)
		if err != nil || *calls != want {
			t.Errorf("%q: calls=%d, %v", answer, *calls, err)
		}
		if msg := map[int]string{0: "Aborted", 1: "Deleted org acme"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%q: output = %q", answer, out)
		}
	}
}

func TestDeleteOrg_YesAndQuiet(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &shared.QuietOutput, true)
	calls := deleteServer(t)

	// No prompt (testutil.Run fails the test on an unanswered prompt), no output
	if out := testutil.Run(t, func() { _ = DeleteOrg("acme") }); *calls != 1 || out != "" {
		t.Fatalf("calls=%d, %q", *calls, out)
	}
}

func TestDeleteOrg_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	testutil.WriteRegistry(t, "")
	if err := DeleteOrg("acme"); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.WriteRegistry(t, githubRegistry)
	if err := DeleteOrg("acme"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if err := DeleteOrg("nope"); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("got %+v", err)
	}
}
