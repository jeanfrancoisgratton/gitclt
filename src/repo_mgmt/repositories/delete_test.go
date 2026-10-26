// gitclt
// Tests repo delete against a fake Forgejo

package repositories

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestDeleteRepo(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Set(t, &shared.QuietOutput, false)
	deleted := false
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/repos/acme/web": func(w http.ResponseWriter, r *http.Request) {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		},
	})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteRepo("web") })
	if err != nil || !deleted || !strings.Contains(out, "Deleted repository acme/web") {
		t.Fatalf("deleted=%v, %v, %q", deleted, err, out)
	}

	if err = DeleteRepo("nope"); err == nil || err.Title != "Unable to delete the repository acme/nope" {
		t.Fatalf("got %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if err = DeleteRepo("web"); err == nil {
		t.Fatal("expected an error without a connection")
	}
}
