// gitclt
// Tests branch rename, against a fake Forgejo: server branches only, and -r required

package branches

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

// renameServer records the bodies of the server's renames; Forgejo answers a rename with the renamed branch, fetched again
func renameServer(t *testing.T, status int) (bodies *[]map[string]any) {
	t.Helper()
	bodies = new([]map[string]any)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/repos/acme/web/branches/dev":   testutil.Record(t, bodies, status, `{}`),
		"GET /api/v1/repos/acme/web/branches/develop": testutil.Reply(`{"name":"develop","commit":{"id":"` + shaMain + `"}}`),
		"GET /api/v1/repos/acme/web":                  testutil.Reply(`{"name":"web","default_branch":"main"}`),
	})
	return bodies
}

// Without -r, nothing is renamed anywhere: the command used to rename the local branch by default
func TestRenameBranch_NeedsRemote(t *testing.T) {
	defaults(t)
	bodies := renameServer(t, http.StatusOK)

	ce := RenameBranch("web", "dev", "develop")
	if ce == nil || ce.Title != "Server branches only" || !strings.Contains(ce.Message, "git branch -m") || len(*bodies) != 0 {
		t.Fatalf("got %+v, bodies %v", ce, *bodies)
	}
}

func TestRenameBranch_Remote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &RemoteOnly, true)
	bodies := renameServer(t, http.StatusOK)
	t.Chdir(t.TempDir()) // the local side is never read

	out := testutil.Run(t, func() {
		if ce := RenameBranch("web", "dev", "develop"); ce != nil {
			t.Error(ce.Error())
		}
	})
	if len(*bodies) != 1 || (*bodies)[0]["name"] != "develop" || !strings.Contains(out, "Renamed server branch dev to develop") {
		t.Fatalf("bodies %v\n%s", *bodies, out)
	}

	// Without -o, the connection's username is the owner
	testutil.Set(t, &OwnerOrg, "")
	if ce := RenameBranch("web", "dev", "develop"); ce != nil || len(*bodies) != 2 {
		t.Fatalf("no -o: %+v, bodies %v", ce, *bodies)
	}
}

func TestRenameBranch_RemoteFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &RemoteOnly, true)
	renameServer(t, http.StatusConflict)

	if ce := RenameBranch("web", "dev", "develop"); ce == nil || ce.Code != http.StatusConflict {
		t.Fatalf("got %+v", ce)
	}
}
