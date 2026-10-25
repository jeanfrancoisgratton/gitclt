// gitclt
// Tests user delete: the confirmation prompt, --yes and --purge

package user

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// A fake Forgejo counting the delete requests on jfg, and recording their purge parameter
func deleteServer(t *testing.T) (calls *int, purge *string) {
	t.Helper()
	calls, purge = new(int), new(string)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/admin/users/jfg": func(w http.ResponseWriter, r *http.Request) {
			*calls++
			*purge = r.URL.Query().Get("purge")
			w.WriteHeader(http.StatusNoContent)
		},
	})
	return calls, purge
}

func TestDeleteUser_Confirmation(t *testing.T) {
	for _, answer := range []string{"", "n", "no", "nope"} {
		defaults(t)
		calls, _ := deleteServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteUser("jfg") }, answer)
		if err != nil || *calls != 0 || !strings.Contains(out, "Aborted") {
			t.Errorf("%q: calls=%d, %v, %q", answer, *calls, err, out)
		}
	}

	for _, answer := range []string{"y", " YES "} {
		defaults(t)
		calls, purge := deleteServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteUser("jfg") }, answer)
		if err != nil || *calls != 1 || *purge != "" || !strings.Contains(out, "Deleted user jfg") {
			t.Errorf("%q: calls=%d, purge=%q, %v, %q", answer, *calls, *purge, err, out)
		}
	}
}

func TestDeleteUser_PurgeAndYes(t *testing.T) {
	defaults(t)
	testutil.Set(t, &PurgeUser, true)
	calls, purge := deleteServer(t)

	// The prompt warns about the purge
	out := testutil.Run(t, func() { _ = DeleteUser("jfg") }, "y")
	if *calls != 1 || *purge != "true" || !strings.Contains(out, "purge all of their content") {
		t.Fatalf("calls=%d, purge=%q, %q", *calls, *purge, out)
	}

	// --yes: no prompt (testutil.Run fails the test on an unanswered prompt); -q: no output
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &PurgeUser, false)
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _ = DeleteUser("jfg") }); *calls != 2 || out != "" {
		t.Fatalf("calls=%d, %q", *calls, out)
	}
}

func TestDeleteUser_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	testutil.WriteRegistry(t, "")
	if err := DeleteUser("jfg"); err == nil || err.Title != "No connection specified" {
		t.Fatalf("no connection: got %+v", err)
	}
	testutil.Forgejo(t, nil)
	if err := DeleteUser("nobody"); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("404: got %+v", err)
	}
}
