// gitclt
// Tests repo info against a fake Forgejo

package repositories

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

const webRepo = `{"id":7,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","description":"The site",
	"default_branch":"develop","ssh_url":"git@forge:acme/web.git","archived":true}`

func TestRepositoryInfo(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/repos/acme/web": testutil.Reply(webRepo)})

	r, err := RepositoryInfo("web", false)
	if err != nil || r.FullName != "acme/web" || !r.Archived {
		t.Fatalf("got %+v, %v", r, err)
	}

	out := testutil.Run(t, func() { _, err = RepositoryInfo("web", true) })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Full name:", "acme/web", "The site", "develop", "git@forge:acme/web.git"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, err = RepositoryInfo("web", true) })
	if err != nil || !strings.Contains(out, `"defaultBranch"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestRepositoryInfo_NotFound(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Forgejo(t, nil)

	var err *cerr.CustomError
	if _, err = RepositoryInfo("nope", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("got %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if _, err = RepositoryInfo("web", false); err == nil {
		t.Fatal("expected an error without a connection")
	}
}

// Features and the settings groups the forge reports; a nil field is n/a, never "no"
func TestRepositoryInfo_Settings(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/repos/acme/web": testutil.Reply(`{"id":7,"owner":{"login":"acme"},
		"name":"web","full_name":"acme/web","has_actions":true,"has_issues":true,"has_wiki":false,"allow_squash_merge":true,
		"internal_tracker":{"enable_time_tracker":true},"external_wiki":{"external_wiki_url":"https://wiki.example.com"}}`)})

	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { _, err = RepositoryInfo("web", true) }))
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Features:", "Actions:", "Packages:", "n/a", "Merge:", "Squash:", "Issue tracker:",
		"Time tracking:", "none (built-in)", "Wiki:", "https://wiki.example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Forgejo has no Actions settings: the group is left out
	if strings.Contains(out, "Token permission") {
		t.Errorf("unexpected Actions settings in:\n%s", out)
	}
}

// Bitbucket reports no feature at all
func TestPrintSettings_NothingReported(t *testing.T) {
	var b strings.Builder
	printSettings(&b, &gol.Repo{})
	out := testutil.StripANSI(b.String())
	if !strings.Contains(out, "not reported by this forge") || strings.Contains(out, "Merge:") {
		t.Fatalf("output = %q", out)
	}
}
