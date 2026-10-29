// gitclt
// Tests branch compare, against a fake Forgejo

package branches

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

// Forgejo's compare endpoint answers with counts[base...head] (the number of commits of head missing from base), 404 otherwise
func compareServer(t *testing.T, counts map[string]int) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
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

func TestCompareBranches(t *testing.T) {
	for _, tc := range []struct {
		ahead, behind int
		status, ff    string
	}{
		{3, 0, "ahead 3", "yes"},
		{1, 2, "ahead 1, behind 2", "no"},
		{0, 4, "behind 4", "no"},
		{0, 0, "in sync", "no"},
	} {
		defaults(t)
		compareServer(t, map[string]int{"main...develop": tc.ahead, "develop...main": tc.behind})

		cmp, ce := CompareBranches("web", "main", "develop", false)
		if ce != nil || cmp.Ahead != tc.ahead || cmp.Behind != tc.behind || cmp.Base != "main" || cmp.Head != "develop" {
			t.Fatalf("%+v: got %+v, %v", tc, cmp, ce)
		}
		out := testutil.Run(t, func() { _, ce = CompareBranches("web", "main", "develop", true) })
		if !strings.Contains(out, "Status:        "+tc.status+"\n") || !strings.Contains(out, "Fast-forward:  "+tc.ff) {
			t.Errorf("%+v:\n%s", tc, out)
		}
	}
}

func TestCompareBranches_JSON(t *testing.T) {
	defaults(t)
	testutil.Set(t, &JSONoutput, true)
	compareServer(t, map[string]int{"main...develop": 3, "develop...main": 1})

	out := testutil.Run(t, func() {
		if _, ce := CompareBranches("web", "main", "develop", true); ce != nil {
			t.Error(ce.Error())
		}
	})
	if !strings.Contains(out, `"ahead": 3`) || !strings.Contains(out, `"behind": 1`) || !strings.Contains(out, `"base": "main"`) {
		t.Fatalf("got:\n%s", out)
	}
}

func TestCompareBranches_Errors(t *testing.T) {
	defaults(t)
	compareServer(t, nil)
	if _, ce := CompareBranches("web", "main", "nope", false); ce == nil || ce.Title != "Unknown branch" || ce.Code != http.StatusNotFound {
		t.Fatalf("404: %+v", ce)
	}

	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web/compare/{basehead}": testutil.Status(http.StatusInternalServerError),
	})
	if _, ce := CompareBranches("web", "main", "develop", false); ce == nil || ce.Code != http.StatusInternalServerError {
		t.Fatalf("500: %+v", ce)
	}

	// Without -o, the connection's username is the owner: same server, same answer
	testutil.Set(t, &OwnerOrg, "")
	if _, ce := CompareBranches("web", "main", "develop", false); ce == nil || ce.Code != http.StatusInternalServerError {
		t.Fatalf("no -o: %+v", ce)
	}
}
