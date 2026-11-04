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

const deleteTarget = `{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`

// A fake Forgejo holding acme/web; the DELETE request sets *deleted
func deleteServer(t *testing.T, deleted *bool) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(deleteTarget),
		"DELETE /api/v1/repos/acme/web": func(w http.ResponseWriter, r *http.Request) {
			*deleted = true
			w.WriteHeader(http.StatusNoContent)
		},
	})
}

func deleteDefaults(t *testing.T) {
	t.Helper()
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Set(t, &AssumeYes, false)
	testutil.Set(t, &shared.QuietOutput, false)
}

func TestDeleteRepo_Confirmation(t *testing.T) {
	for _, tc := range []struct {
		answer  string
		deleted bool
	}{
		{"y", true},
		{"YES", true},
		{"n", false},
		{"", false},
		{"web", false},
	} {
		deleteDefaults(t)
		deleted := false
		deleteServer(t, &deleted)

		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteRepo("web") }, tc.answer)
		if err != nil || deleted != tc.deleted || !strings.Contains(out, "Delete repository acme/web on test") {
			t.Errorf("answer %q: deleted=%v, %v, %q", tc.answer, deleted, err, out)
		}
		if want := "Aborted, repository acme/web was not deleted"; !tc.deleted && !strings.Contains(out, want) {
			t.Errorf("answer %q: missing %q in %q", tc.answer, want, out)
		}
		if want := "Deleted repository acme/web"; tc.deleted && !strings.Contains(out, want) {
			t.Errorf("answer %q: missing %q in %q", tc.answer, want, out)
		}
	}
}

// --yes asks nothing
func TestDeleteRepo_AssumeYes(t *testing.T) {
	deleteDefaults(t)
	testutil.Set(t, &AssumeYes, true)
	deleted := false
	deleteServer(t, &deleted)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteRepo("web") })
	if err != nil || !deleted || strings.Contains(out, "This cannot be undone") {
		t.Fatalf("deleted=%v, %v, %q", deleted, err, out)
	}
}

func TestDeleteRepo_Errors(t *testing.T) {
	deleteDefaults(t)
	deleted := false
	deleteServer(t, &deleted)

	// Refused before asking
	if err := DeleteRepo("nope"); err == nil || err.Title != "No such repository" || err.Code != http.StatusNotFound {
		t.Fatalf("got %+v", err)
	}
	if deleted {
		t.Fatal("nothing should have been deleted")
	}

	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":    testutil.Status(http.StatusForbidden),
		"DELETE /api/v1/repos/acme/web": testutil.Status(http.StatusNoContent),
	})
	if err := DeleteRepo("web"); err == nil || err.Code != http.StatusForbidden {
		t.Fatalf("get refused: %+v", err)
	}

	testutil.Set(t, &AssumeYes, true)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":    testutil.Reply(deleteTarget),
		"DELETE /api/v1/repos/acme/web": testutil.Status(http.StatusForbidden),
	})
	if err := DeleteRepo("web"); err == nil || err.Title != "Unable to delete the repository acme/web" {
		t.Fatalf("delete refused: %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if err := DeleteRepo("web"); err == nil {
		t.Fatal("expected an error without a connection")
	}
}
