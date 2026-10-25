// gitclt
// Tests repo info against a fake Forgejo

package repositories

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
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
