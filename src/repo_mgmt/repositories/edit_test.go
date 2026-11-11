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
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Stands in for cmd.Flags().Changed: only the named flags were passed
func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

func editDefaults(t *testing.T) {
	t.Helper()
	defaults(t)
	testutil.Set(t, &shared.QuietOutput, false)
	for _, s := range []*string{&EditRepoName, &EditRepoDescription, &EditRepoWebsite, &EditRepoDefaultBranch, &EditDefaultMerge,
		&EditExternalTracker, &EditTrackerFormat, &EditTrackerStyle, &EditTrackerPattern, &EditExternalWiki,
		&EditActionsAllowed, &EditActionsToken} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&EditRepoPublic, &EditRepoArchived, &EditRepoTemplate, &EditActions, &EditWiki, &EditSquash} {
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

// Features and merge settings end up in the PATCH; the reply reports them, so there is nothing to warn about
func TestEditRepo_FeaturesAndSettings(t *testing.T) {
	editDefaults(t)
	testutil.Set(t, &EditActions, true)
	testutil.Set(t, &EditSquash, true)
	testutil.Set(t, &EditDefaultMerge, "squash")
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_pull_requests":true}`),
		"PATCH /api/v1/repos/acme/web": testutil.Record(t, &bodies, http.StatusOK,
			`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_actions":true,"has_wiki":false,
			"has_pull_requests":true,"allow_squash_merge":true,"default_merge_style":"squash"}`),
	})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditRepo("web", passed("actions", "wiki", "squash", "default-merge")) })
	if err != nil || len(bodies) != 1 {
		t.Fatalf("%d requests, %v", len(bodies), err)
	}
	b := bodies[0]
	// --wiki was passed unset: --wiki=false, sent as such
	if b["has_actions"] != true || b["has_wiki"] != false || b["allow_squash_merge"] != true || b["default_merge_style"] != "squash" {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Updated repository acme/web") {
		t.Fatalf("output = %q", out)
	}
}

// The server answers success but keeps Actions off: the rest went through, so a warning, not an error
func TestEditRepo_NotApplied(t *testing.T) {
	editDefaults(t)
	testutil.Set(t, &EditActions, true)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/repos/acme/web": testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_actions":false}`),
	})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditRepo("web", passed("actions")) })
	if err != nil || !strings.Contains(out, "acme/web was updated, but") || !strings.Contains(out, "actions on") {
		t.Fatalf("%v, output = %q", err, out)
	}
}

// Refused before any request: bad values, an incomplete external tracker, and (by the library) a GitHub-only setting on Forgejo
func TestEditRepo_Refused(t *testing.T) {
	editDefaults(t)
	testutil.Forgejo(t, nil) // any request would be a 404

	for _, tc := range []struct {
		set   func()
		flags []string
		want  string
	}{
		{func() { EditDefaultMerge = "manually-merged" }, []string{"default-merge"}, "--default-merge"},
		{func() { EditActionsAllowed = "some" }, []string{"actions-allowed"}, "--actions-allowed"},
		{func() { EditActionsToken = "admin" }, []string{"actions-token"}, "--actions-token"},
		{func() { EditTrackerFormat = "x" }, []string{"tracker-format"}, "needs --external-tracker"},
		{func() { EditExternalTracker, EditTrackerStyle = "https://t", "letters" }, []string{"external-tracker", "tracker-style"}, "--tracker-style"},
		{func() { EditExternalTracker, EditTrackerPattern = "https://t", "[0-9]+" }, []string{"external-tracker", "tracker-pattern"}, "needs --tracker-style regexp"},
		{func() {}, []string{"auto-merge"}, "not supported"},
		{func() {}, []string{"discussions"}, "not supported"},
	} {
		editDefaults(t)
		tc.set()
		err := EditRepo("web", passed(tc.flags...))
		if err == nil || !strings.Contains(err.Title+" "+err.Message, tc.want) {
			t.Errorf("%v: got %+v", tc.flags, err)
		}
	}
}

// --external-tracker "" goes back to the built-in tracker: an external tracker with a zero URL
func TestEditOptions_ExternalTracker(t *testing.T) {
	editDefaults(t)
	ero, err := editOptionsFromFlags(passed("external-tracker"))
	if err != nil || ero.IssueTracker.External == nil || *ero.IssueTracker.External != (gol.ExternalTracker{}) {
		t.Fatalf("got %+v, %v", ero.IssueTracker.External, err)
	}

	testutil.Set(t, &EditExternalTracker, "https://t.example.com")
	testutil.Set(t, &EditTrackerStyle, "regexp")
	testutil.Set(t, &EditTrackerPattern, "#([0-9]+)")
	ero, err = editOptionsFromFlags(passed("external-tracker", "tracker-style", "tracker-pattern"))
	want := gol.ExternalTracker{URL: "https://t.example.com", Style: gol.ExternalIssueRegexp, Pattern: "#([0-9]+)"}
	if err != nil || *ero.IssueTracker.External != want {
		t.Fatalf("got %+v, %v", ero.IssueTracker.External, err)
	}
}
