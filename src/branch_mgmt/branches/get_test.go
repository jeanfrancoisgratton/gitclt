// gitclt
// Tests branch get: one branch, both sides merged, against a fake working copy and a fake Forgejo

package branches

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

// The server has main (default, protected) and develop, like branchServer's; any other branch is a 404
func getServer(t *testing.T, counts map[string]int) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(`{"name":"web","owner":{"login":"acme"},"default_branch":"main"}`),
		"GET /api/v1/repos/acme/web/branches/main": testutil.Reply(`{"name":"main","protected":true,` +
			`"commit":{"id":"` + shaMain + `","message":"Release\n\nThe details","author":{"name":"Alice"},"timestamp":"2026-10-20T14:00:00Z"}}`),
		"GET /api/v1/repos/acme/web/branches/develop": testutil.Reply(`{"name":"develop","commit":{"id":"` + shaOld + `"}}`),
		"GET /api/v1/repos/acme/web/compare/{basehead}": func(w http.ResponseWriter, r *http.Request) {
			n, ok := counts[r.PathValue("basehead")]
			if !ok {
				testutil.Status(http.StatusNotFound)(w, r)
				return
			}
			testutil.Reply(fmt.Sprintf(`{"total_commits":%d}`, n))(w, r)
		},
	})
}

func TestGetBranch_BothSides(t *testing.T) {
	defaults(t)
	getServer(t, developCounts)
	t.Chdir(fakeWorkingCopy(t))

	b, ce := GetBranch("web", "develop", false)
	if ce != nil {
		t.Fatal(ce.Error())
	}
	if !b.Current || b.LocalSHA != shaDevelop || b.ServerSHA != shaOld || b.Ahead != 2 || b.Behind != 1 {
		t.Fatalf("develop: %+v", b)
	}

	out := testutil.Run(t, func() { _, ce = GetBranch("web", "develop", true) })
	for _, want := range []string{"2222222 (checked out)", "3333333", "ahead 2, behind 1", "Default:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// The server's last commit: its title only
	out = testutil.Run(t, func() { _, ce = GetBranch("web", "main", true) })
	for _, want := range []string{", Alice", "Commit title:   Release\n", "Protected:      yes", "in sync"} {
		if !strings.Contains(out, want) {
			t.Errorf("main: missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "The details") || strings.Contains(out, "(checked out)") {
		t.Errorf("main:\n%s", out)
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, ce = GetBranch("web", "develop", true) })
	if ce != nil || !strings.Contains(out, `"localSha"`) || !strings.Contains(out, `"ahead": 2`) {
		t.Fatalf("JSON: %v\n%s", ce, out)
	}
}

// A branch on one side only is still shown
func TestGetBranch_OneSide(t *testing.T) {
	defaults(t)
	getServer(t, nil)
	t.Chdir(fakeWorkingCopy(t))

	out := testutil.Run(t, func() {
		if _, ce := GetBranch("web", "feature/x", true); ce != nil {
			t.Error(ce.Error())
		}
	})
	if !strings.Contains(out, "4444444") || !strings.Contains(out, "not on the server") || !strings.Contains(out, "local only") || strings.Contains(out, "Default:") {
		t.Fatalf("local only:\n%s", out)
	}

	// Outside of a working copy: the server's side, and why the local one is missing
	t.Chdir(t.TempDir())
	out = testutil.Run(t, func() {
		if _, ce := GetBranch("web", "main", true); ce != nil {
			t.Error(ce.Error())
		}
	})
	if !strings.Contains(out, "n/a (") || !strings.Contains(out, "is not inside a git working copy") || strings.Contains(out, "Status:") {
		t.Fatalf("no working copy:\n%s", out)
	}
}

func TestGetBranch_Errors(t *testing.T) {
	defaults(t)
	getServer(t, nil)
	t.Chdir(fakeWorkingCopy(t))

	if _, ce := GetBranch("web", "nope", false); ce == nil || ce.Title != "No such branch" {
		t.Fatalf("nowhere: %+v", ce)
	}
	testutil.Set(t, &OwnerOrg, "")
	if _, ce := GetBranch("web", "main", false); ce == nil || ce.Title != "Missing owner" {
		t.Fatalf("no owner: %+v", ce)
	}

	// A server failure other than a 404 is an error, even when the branch is local
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web/branches/develop": testutil.Status(http.StatusInternalServerError),
	})
	if _, ce := GetBranch("web", "develop", false); ce == nil || ce.Code != http.StatusInternalServerError {
		t.Fatalf("500: %+v", ce)
	}
}
