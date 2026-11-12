// gitclt
// Tests the branch details of branch ls and get: parent, upstream, CI state, pull requests, and the parts left out

package branches

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
)

// fakeWorkingCopy, where feature/x was created from develop (its reflog says so) and tracks origin/feature/x, which
// the last fetch did not see
func detailedWorkingCopy(t *testing.T) string {
	t.Helper()
	root := fakeWorkingCopy(t)
	writeFiles(t, root, map[string]string{
		".git/logs/refs/heads/feature/x": "0000000000000000000000000000000000000000 " + shaFeature +
			" J.F. <jfg@example.com> 1762900000 -0500\tbranch: Created from develop\n",
		".git/config": "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = git@forge.example:acme/web.git\n" +
			"\tfetch = +refs/heads/*:refs/remotes/origin/*\n[branch \"feature/x\"]\n\tremote = origin\n\tmerge = refs/heads/feature/x\n",
	})
	return root
}

// branchServer's, plus a CI state on main (another branch has none: a 404) and an open pull request from develop
func detailsServer(t *testing.T) {
	t.Helper()
	branchServer(t, developCounts, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web/commits/main/status": testutil.Reply(`{"state":"success","sha":"` + shaMain + `","total_count":1,` +
			`"statuses":[{"context":"ci/woodpecker","status":"success","description":"Build passed","target_url":"https://ci.example/1"}]}`),
		"GET /api/v1/repos/acme/web/pulls": testutil.Reply(`[{"number":7,"title":"Dev work","state":"open","user":{"login":"bob"},` +
			`"head":{"ref":"develop","repo":{"full_name":"acme/web"}},"base":{"ref":"main"}}]`),
		// branch get reads one branch
		"GET /api/v1/repos/acme/web/branches/main":    testutil.Reply(`{"name":"main","protected":true,"commit":{"id":"` + shaMain + `"}}`),
		"GET /api/v1/repos/acme/web/branches/develop": testutil.Reply(`{"name":"develop","commit":{"id":"` + shaOld + `"}}`),
	})
}

func find(list []Branch, name string) Branch {
	for _, b := range list {
		if b.Name == name {
			return b
		}
	}
	return Branch{}
}

func TestListBranches_Details(t *testing.T) {
	defaults(t)
	detailsServer(t)
	t.Chdir(detailedWorkingCopy(t))

	list, err := ListBranches("web", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if f := find(list, "feature/x"); f.Parent == nil || !f.Parent.Exact || f.Parent.Branch != "develop" ||
		f.Upstream == nil || f.Upstream.Remote != "origin" || f.Upstream.Branch != "feature/x" || f.Upstream.SHA != "" {
		t.Errorf("feature/x: parent %+v, upstream %+v", f.Parent, f.Upstream)
	}
	if m := find(list, "main"); m.CI == nil || m.CI.State != "success" || len(m.CI.Statuses) != 1 {
		t.Errorf("main: CI %+v", m.CI)
	}
	if d := find(list, "develop"); len(d.PullRequests) != 1 || d.PullRequests[0].Number != 7 || d.PullRequests[0].Base != "main" {
		t.Errorf("develop: pull requests %+v", d.PullRequests)
	}

	out := testutil.Run(t, func() { _, err = ListBranches("web", true) })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Parent", "Upstream", "Vs main", "Merged", "Merge base", "CI state", "PRs",
		"origin/feature/x (gone)", "success", "#7", "Not shown: status (develop, release/1.0)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Default", "Protected"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, out)
		}
	}

	// -w hides the notes
	testutil.Set(t, &SuppressWarnings, true)
	if out = testutil.Run(t, func() { _, err = ListBranches("web", true) }); strings.Contains(out, "Not shown") {
		t.Errorf("-w:\n%s", out)
	}
}

// -l reads the clone alone, without the server's columns; -r the server alone, without the clone's
func TestListBranches_DetailsOneSide(t *testing.T) {
	defaults(t)
	detailsServer(t)
	t.Chdir(detailedWorkingCopy(t))

	testutil.Set(t, &LocalOnly, true)
	out := testutil.Run(t, func() { _, _ = ListBranches("", true) })
	if !strings.Contains(out, "Parent") || strings.Contains(out, "CI") || strings.Contains(out, "PRs") {
		t.Errorf("-l:\n%s", out)
	}
	testutil.Set(t, &LocalOnly, false)
	testutil.Set(t, &RemoteOnly, true)
	out = testutil.Run(t, func() { _, _ = ListBranches("web", true) })
	if strings.Contains(out, "Parent") || strings.Contains(out, "Upstream") || !strings.Contains(out, "PRs") {
		t.Errorf("-r:\n%s", out)
	}
}

func TestGetBranch_Details(t *testing.T) {
	defaults(t)
	detailsServer(t)
	t.Chdir(detailedWorkingCopy(t))

	// feature/x is only local: where it comes from, what it tracks
	b, ce := GetBranch("web", "feature/x", false)
	if ce != nil {
		t.Fatal(ce.Error())
	}
	if b.ServerSHA != "" || b.Parent == nil || b.Parent.Branch != "develop" || b.Upstream == nil {
		t.Errorf("feature/x: %+v", b)
	}
	out := testutil.Run(t, func() { _, ce = GetBranch("web", "feature/x", true) })
	for _, want := range []string{"Created from:", "develop, at 4444444", "Upstream:", "origin/feature/x (gone)", "not on the server"} {
		if !strings.Contains(out, want) {
			t.Errorf("feature/x: missing %q in:\n%s", want, out)
		}
	}

	// main: its CI state, every status
	out = testutil.Run(t, func() { _, ce = GetBranch("web", "main", true) })
	for _, want := range []string{"CI state:", "success", "ci/woodpecker: success, Build passed (https://ci.example/1)", "Pull requests:", "none open"} {
		if !strings.Contains(out, want) {
			t.Errorf("main: missing %q in:\n%s", want, out)
		}
	}

	// develop: its pull request; no CI state there (404), which is listed, not an error
	b, ce = GetBranch("web", "develop", false)
	if ce != nil || len(b.PullRequests) != 1 || b.CI != nil {
		t.Fatalf("develop: %+v, %v", b, ce)
	}
	out = testutil.Run(t, func() { _, ce = GetBranch("web", "develop", true) })
	for _, want := range []string{"#7 Dev work → main, by bob", "Could not be read:", "CI state: "} {
		if !strings.Contains(out, want) {
			t.Errorf("develop: missing %q in:\n%s", want, out)
		}
	}

	// -l needs no server: a branch only there is not found
	testutil.Set(t, &LocalOnly, true)
	if _, ce = GetBranch("", "release/1.0", false); ce == nil || ce.Message != "release/1.0 is not in the local working copy" {
		t.Errorf("-l: %+v", ce)
	}
	testutil.Set(t, &LocalOnly, false)
	testutil.Set(t, &RemoteOnly, true)
	if out = testutil.Run(t, func() { _, ce = GetBranch("web", "feature/x", true) }); ce == nil || ce.Message != "feature/x is not on the server" {
		t.Errorf("-r: %+v\n%s", ce, out)
	}
}
