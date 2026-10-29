// gitclt
// Tests repo edit: only the flags that were passed are sent, to the right owner

package repositories

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Stands in for cmd.Flags().Changed: only the named flags were passed
func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

func editDefaults(t *testing.T) {
	t.Helper()
	defaults(t)
	testutil.Set(t, &shared.QuietOutput, false)
	for _, s := range []*string{&EditRepoName, &EditRepoDescription, &EditRepoWebsite, &EditRepoDefaultBranch} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&EditRepoPublic, &EditRepoArchived, &EditRepoTemplate} {
		testutil.Set(t, b, false)
	}
}

const editedRepo = `{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`

func TestEditRepo_OnlyPassedFlagsAreSent(t *testing.T) {
	editDefaults(t)
	testutil.Set(t, &EditRepoDefaultBranch, "develop")
	testutil.Set(t, &EditRepoDescription, "Gadgets") // set, but not passed: must not be sent
	var bodies []map[string]any
	// No -o: the fake connection's username (acme) is the owner
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/repos/acme/web": testutil.Record(t, &bodies, http.StatusOK, editedRepo)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() {
		err = EditRepo("web", passed("default-branch", "website", "public", "archived"))
	})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("%d requests, %v", len(bodies), err)
	}
	b := bodies[0]
	if got := slices.Sorted(maps.Keys(b)); !slices.Equal(got, []string{"archived", "default_branch", "private", "website"}) {
		t.Fatalf("sent %v", got)
	}
	// --website "" clears the field, --public=false makes it private, --archived=false unarchives: all must be sent
	if b["default_branch"] != "develop" || b["website"] != "" || b["private"] != true || b["archived"] != false {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Updated repository acme/web") {
		t.Fatalf("output = %q", out)
	}
}

func TestEditRepo_RenameWithOwner(t *testing.T) {
	editDefaults(t)
	testutil.Set(t, &OwnerOrg, "devops")
	testutil.Set(t, &EditRepoName, "site")
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/repos/devops/web": testutil.Record(t, &bodies, http.StatusOK,
		`{"id":1,"owner":{"login":"devops"},"name":"site","full_name":"devops/site"}`)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditRepo("web", passed("name")) })
	if err != nil || len(bodies) != 1 || bodies[0]["name"] != "site" {
		t.Fatalf("bodies %v, %v", bodies, err)
	}
	// The new name is shown
	if !strings.Contains(out, "Updated repository devops/site") {
		t.Fatalf("output = %q", out)
	}
}

func TestEditRepo_Errors(t *testing.T) {
	editDefaults(t)

	// Checked before connecting: no registry needed
	testutil.WriteRegistry(t, "")
	if err := EditRepo("web", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: got %+v", err)
	}
	if err := EditRepo("web", passed("description")); err == nil {
		t.Fatal("expected an error without a connection")
	}

	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/repos/acme/web": testutil.Status(http.StatusForbidden)})
	if err := EditRepo("web", passed("description")); err == nil || err.Code != http.StatusForbidden ||
		!strings.Contains(err.Title, "acme/web") {
		t.Fatalf("403: got %+v", err)
	}
}
