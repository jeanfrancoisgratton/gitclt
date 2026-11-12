// gitclt
// Tests branch rm, against a fake Forgejo: server branches only, and -r required

package branches

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

// deleteServer answers the server's branch deletions with status, and records the deleted names
func deleteServer(t *testing.T, status int) (deleted *[]string) {
	t.Helper()
	deleted = new([]string)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/repos/acme/web/branches/{name}": func(w http.ResponseWriter, r *http.Request) {
			*deleted = append(*deleted, r.PathValue("name"))
			testutil.Status(status)(w, r)
		},
	})
	return deleted
}

// Without -r, nothing is deleted anywhere: the command used to delete local branches by default
func TestDeleteBranch_NeedsRemote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	deleted := deleteServer(t, http.StatusNoContent)

	ce := DeleteBranch("web", []string{"old"})
	if ce == nil || ce.Title != "Server branches only" || !strings.Contains(ce.Message, "git branch -d") || len(*deleted) != 0 {
		t.Fatalf("got %+v, deleted %v", ce, *deleted)
	}
}

func TestDeleteBranch_Remote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &RemoteOnly, true)
	deleted := deleteServer(t, http.StatusNoContent)
	t.Chdir(t.TempDir()) // the local side is never read

	out := testutil.Run(t, func() {
		if ce := DeleteBranch("web", []string{"old", "older"}); ce != nil {
			t.Error(ce.Error())
		}
	})
	if !slices.Equal(*deleted, []string{"old", "older"}) || !strings.Contains(out, "Deleted server branch older") || strings.Contains(out, "local") {
		t.Fatalf("deleted %v:\n%s", *deleted, out)
	}

	// Without -o, the connection's username is the owner
	testutil.Set(t, &OwnerOrg, "")
	if ce := DeleteBranch("web", []string{"old"}); ce != nil || len(*deleted) != 3 {
		t.Fatalf("no -o: %+v, deleted %v", ce, *deleted)
	}
}

// The first failure stops the run
func TestDeleteBranch_RemoteFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &RemoteOnly, true)
	deleted := deleteServer(t, http.StatusForbidden)

	ce := DeleteBranch("web", []string{"old", "older"})
	if ce == nil || ce.Code != http.StatusForbidden || !slices.Equal(*deleted, []string{"old"}) {
		t.Fatalf("got %+v, deleted %v", ce, *deleted)
	}
}
