// gitclt
// Tests repo ls against a fake Forgejo: owner filter, unknown-owner fallback, output formats

package repositories

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

const myRepos = `[
	{"id":9,"owner":{"login":"me"},"name":"zeta","full_name":"me/zeta","html_url":"https://forge/me/zeta","empty":true},
	{"id":3,"owner":{"login":"me"},"name":"alpha","full_name":"me/alpha","private":true}]`

// Resets the package flags to their defaults for the duration of the test
func defaults(t *testing.T) {
	t.Helper()
	testutil.Set(t, &JSONoutput, false)
	testutil.Set(t, &OwnerOrg, "")
	testutil.Set(t, &SortBy, "")
	testutil.Set(t, &SortByAsc, "")
}

func names(repos []gol.Repo) string {
	var out []string
	for _, r := range repos {
		out = append(out, r.FullName)
	}
	return strings.Join(out, " ")
}

func TestListRepositories_Data(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Reply(myRepos)})

	// Not displayed: the list is returned, already sorted (default: ID, ascending)
	repos, err := ListRepositories(false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if got := names(repos); got != "me/alpha me/zeta" {
		t.Fatalf("got %s", got)
	}
}

func TestListRepositories_Table(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Reply(myRepos)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListRepositories(true) })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Number of repositories on test: 2", "ID", "alpha", "private", "https://forge/me/zeta", "yes"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestListRepositories_Empty(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Reply(`[]`)})
	out := testutil.Run(t, func() { _, _ = ListRepositories(true) })
	if strings.TrimSpace(out) != "Number of repositories on test: 0" {
		t.Fatalf("output = %q", out)
	}
}

func TestListRepositories_JSON(t *testing.T) {
	defaults(t)
	testutil.Set(t, &JSONoutput, true)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Reply(myRepos)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListRepositories(true) })
	if err != nil {
		t.Fatal(err.Error())
	}
	if !strings.Contains(out, `"fullName"`) || strings.Index(out, "me/alpha") > strings.Index(out, "me/zeta") {
		t.Fatalf("output = %s", out)
	}
}

// -o ORG lists that organization's repositories (the library asks whether the owner is an org first)
func TestListRepositories_OwnerOrg(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/orgs/acme":       testutil.Reply(`{"id":1,"username":"acme"}`),
		"GET /api/v1/orgs/acme/repos": testutil.Reply(`[{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}]`),
	})
	repos, err := ListRepositories(false)
	if err != nil || names(repos) != "acme/web" {
		t.Fatalf("got %s, %v", names(repos), err)
	}
}

// An unknown owner falls back to listing everything
func TestListRepositories_UnknownOwner(t *testing.T) {
	defaults(t)
	testutil.Set(t, &OwnerOrg, "ghost")
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Reply(myRepos)})
	repos, err := ListRepositories(false)
	if err != nil || names(repos) != "me/alpha me/zeta" {
		t.Fatalf("got %s, %v", names(repos), err)
	}
}

func TestListRepositories_Errors(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/repos": testutil.Status(http.StatusForbidden)})
	if _, err := ListRepositories(false); err == nil || err.Code != http.StatusForbidden {
		t.Fatalf("got %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if _, err := ListRepositories(false); err == nil || err.Title != "No connection specified" {
		t.Fatalf("got %+v", err)
	}
}
