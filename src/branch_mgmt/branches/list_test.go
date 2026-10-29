// gitclt
// Tests branch ls: local, server and merged views, against a fake working copy and a fake Forgejo

package branches

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
)

// Resets the package flags to their defaults for the duration of the test
func defaults(t *testing.T) {
	t.Helper()
	for _, b := range []*bool{&JSONoutput, &LocalOnly, &RemoteOnly, &AllBranches, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	testutil.Set(t, &OwnerOrg, "acme")
}

// The server has main (default, protected) and develop, at the same commits as the working copy's main and stale develop,
// and a release branch the working copy does not have
// Its compare endpoint answers with counts[base...head] (the number of commits of head missing from base), 404 otherwise
func branchServer(t *testing.T, counts map[string]int) (compares *int) {
	t.Helper()
	compares = new(int)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(`{"name":"web","owner":{"login":"acme"},"default_branch":"main"}`),
		"GET /api/v1/repos/acme/web/branches": testutil.Reply(`[
			{"name":"main","commit":{"id":"` + shaMain + `","message":"Release","author":{"name":"Alice"},"timestamp":"2026-10-20T14:00:00Z"},"protected":true},
			{"name":"develop","commit":{"id":"` + shaOld + `"}},
			{"name":"release/1.0","commit":{"id":"` + shaOld + `"}}]`),
		"GET /api/v1/repos/acme/web/compare/{basehead}": func(w http.ResponseWriter, r *http.Request) {
			*compares++
			n, ok := counts[r.PathValue("basehead")]
			if !ok {
				testutil.Status(http.StatusNotFound)(w, r)
				return
			}
			testutil.Reply(fmt.Sprintf(`{"total_commits":%d}`, n))(w, r)
		},
	})
	return compares
}

// The working copy's develop (shaDevelop) is 2 commits ahead of the server's, and 1 behind
var developCounts = map[string]int{"develop..." + shaDevelop: 2, shaDevelop + "...develop": 1}

func names(list []Branch) []string {
	var n []string
	for _, b := range list {
		n = append(n, b.Name)
	}
	return n
}

func TestListBranches_All(t *testing.T) {
	defaults(t)
	compares := branchServer(t, developCounts)
	t.Chdir(fakeWorkingCopy(t))

	list, err := ListBranches("web", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !slices.Equal(names(list), []string{"develop", "feature/x", "main", "release/1.0"}) {
		t.Fatalf("got %v", names(list))
	}
	// Only develop differs on both sides: one comparison, two requests on Forgejo
	if d := list[0]; !d.Current || d.LocalSHA != shaDevelop || d.ServerSHA != shaOld || d.Ahead != 2 || d.Behind != 1 || *compares != 2 {
		t.Errorf("develop: %+v", d)
	}
	if m := list[2]; !m.IsDefault || !m.Protected || m.LocalSHA != m.ServerSHA || m.LastCommit == nil || m.LastCommit.Author != "Alice" {
		t.Errorf("main: %+v", m)
	}

	out := testutil.Run(t, func() { _, err = ListBranches("web", true) })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Status", "* develop", "2222222", "ahead 2, behind 1", "Last commit", ", Alice", "local only", "in sync", "server only"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, err = ListBranches("web", true) })
	if err != nil || !strings.Contains(out, `"localSha"`) || !strings.Contains(out, `"current": true`) || !strings.Contains(out, `"behind": 1`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

// -a overrides -l and -r; -l with -r is the same as -a
func TestListBranches_AllOverrides(t *testing.T) {
	for _, flags := range [][]*bool{{&AllBranches, &LocalOnly}, {&AllBranches, &RemoteOnly}, {&LocalOnly, &RemoteOnly}} {
		defaults(t)
		for _, f := range flags {
			testutil.Set(t, f, true)
		}
		branchServer(t, developCounts)
		t.Chdir(fakeWorkingCopy(t))
		list, err := ListBranches("web", false)
		if err != nil || len(list) != 4 || list[1].Name != "feature/x" || list[3].Name != "release/1.0" {
			t.Fatalf("all=%t local=%t remote=%t: %v, %+v", AllBranches, LocalOnly, RemoteOnly, names(list), err)
		}
	}
}

func TestListBranches_Local(t *testing.T) {
	defaults(t)
	testutil.Set(t, &LocalOnly, true)
	testutil.Set(t, &OwnerOrg, "")
	root := fakeWorkingCopy(t)
	t.Chdir(root)

	// No server involved (no fake Forgejo either), and no repo needed
	var err error
	out := testutil.Run(t, func() {
		if _, ce := ListBranches("", true); ce != nil {
			err = ce
		}
	})
	if err != nil || !strings.Contains(out, "feature/x") || strings.Contains(out, "Server") {
		t.Fatalf("%v\n%s", err, out)
	}

	// A repo name is checked against the remotes; without -o, only the name part
	if _, ce := ListBranches("web", false); ce != nil {
		t.Fatalf("web: %v", ce)
	}
	testutil.Set(t, &OwnerOrg, "other")
	if _, ce := ListBranches("web", false); ce == nil || ce.Title != "Not the same repository" {
		t.Fatalf("other/web: %+v", ce)
	}

	// Outside of a working copy, --local has nothing to show
	t.Chdir(t.TempDir())
	if _, ce := ListBranches("", false); ce == nil || ce.Title != "No local working copy" {
		t.Fatalf("no working copy: %+v", ce)
	}
}

func TestListBranches_Remote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &RemoteOnly, true)
	branchServer(t, developCounts)
	t.Chdir(t.TempDir()) // the local side is never read

	out := testutil.Run(t, func() { _, _ = ListBranches("web", true) })
	if strings.Contains(out, "Local") || strings.Contains(out, "Status") || !strings.Contains(out, "release/1.0") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestListBranches_AllWithoutWorkingCopy(t *testing.T) {
	defaults(t)
	branchServer(t, developCounts)

	// Not a working copy, or another repo's: the server's branches only, with a notice
	t.Chdir(t.TempDir())
	out := testutil.Run(t, func() { _, _ = ListBranches("web", true) })
	if !strings.Contains(out, "is not inside a git working copy; showing the server's branches only") || strings.Contains(out, "Status") {
		t.Fatalf("got:\n%s", out)
	}

	testutil.Set(t, &OwnerOrg, "other")
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/other/web":          testutil.Reply(`{"name":"web","default_branch":"main"}`),
		"GET /api/v1/repos/other/web/branches": testutil.Reply(`[{"name":"main","commit":{"id":"` + shaMain + `"}}]`),
	})
	t.Chdir(fakeWorkingCopy(t))
	out = testutil.Run(t, func() { _, _ = ListBranches("web", true) })
	if !strings.Contains(out, "is not a clone of other/web") || strings.Contains(out, "feature/x") {
		t.Fatalf("other repo:\n%s", out)
	}

	// -q silences the notice, not the list
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, _ = ListBranches("web", true) }); strings.Contains(out, "clone") || !strings.Contains(out, "main") {
		t.Fatalf("quiet:\n%s", out)
	}
}

func TestListBranches_Errors(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, "")
	if _, err := ListBranches("", false); err == nil || err.Title != "Missing repository" {
		t.Fatalf("no repo: %+v", err)
	}

	testutil.Set(t, &RemoteOnly, true)
	testutil.Forgejo(t, nil)
	if _, err := ListBranches("nope", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("404: %+v", err)
	}
}

// The status of a branch that differs, from the local branch's point of view
func TestListBranches_Status(t *testing.T) {
	for _, tc := range []struct {
		counts map[string]int
		want   string
	}{
		{map[string]int{"develop..." + shaDevelop: 2, shaDevelop + "...develop": 0}, "ahead 2"},
		{map[string]int{"develop..." + shaDevelop: 0, shaDevelop + "...develop": 3}, "behind 3"},
		{nil, "unpushed commits"}, // the server does not know the local head
	} {
		defaults(t)
		branchServer(t, tc.counts)
		t.Chdir(fakeWorkingCopy(t))
		out := testutil.Run(t, func() { _, _ = ListBranches("web", true) })
		if !strings.Contains(out, tc.want) {
			t.Errorf("%v: missing %q in:\n%s", tc.counts, tc.want, out)
		}
	}

	// Any other failure is an error, not a status
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                    testutil.Reply(`{"name":"web","default_branch":"main"}`),
		"GET /api/v1/repos/acme/web/branches":           testutil.Reply(`[{"name":"develop","commit":{"id":"` + shaOld + `"}}]`),
		"GET /api/v1/repos/acme/web/compare/{basehead}": testutil.Status(http.StatusInternalServerError),
	})
	t.Chdir(fakeWorkingCopy(t))
	if _, err := ListBranches("web", false); err == nil || err.Code != http.StatusInternalServerError {
		t.Fatalf("500: %+v", err)
	}
}
