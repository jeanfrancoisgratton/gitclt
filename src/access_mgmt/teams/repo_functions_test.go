// gitclt
// Tests team repos, addrepo and removerepo against a fake Forgejo

package teams

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const devsRepos = `[{"id":11,"name":"widgets","full_name":"acme/widgets","owner":{"login":"acme"},"description":"The widgets",
	"private":true,"html_url":"https://forge.example.com/acme/widgets"}]`

// A fake Forgejo where acme/devs has access to widgets; it counts the add and remove requests
func reposServer(t *testing.T) (added, removed *int) {
	t.Helper()
	added, removed = new(int), new(int)
	testutil.Forgejo(t, teamRoutes(map[string]http.HandlerFunc{
		"GET /api/v1/teams/7/repos": testutil.Reply(devsRepos),
		"PUT /api/v1/teams/7/repos/acme/gadgets": func(w http.ResponseWriter, r *http.Request) {
			*added++
			w.WriteHeader(http.StatusNoContent)
		},
		"DELETE /api/v1/teams/7/repos/acme/widgets": func(w http.ResponseWriter, r *http.Request) {
			*removed++
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	return added, removed
}

func TestListRepos(t *testing.T) {
	defaults(t)
	reposServer(t)

	repos, err := ListRepos("acme", "devs", false)
	if err != nil || len(repos) != 1 || repos[0].FullName != "acme/widgets" {
		t.Fatalf("got %+v, %v", repos, err)
	}

	out := testutil.Run(t, func() { _, err = ListRepos("acme", "devs", true) })
	for _, want := range []string{"acme/widgets", "The widgets", "private"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListRepos("acme", "devs", true) }); err != nil || !strings.Contains(out, `"acme/widgets"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestAddRepoToTeam(t *testing.T) {
	defaults(t)
	added, _ := reposServer(t)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = AddRepoToTeam("acme", "devs", "gadgets") })
	if err != nil || *added != 1 || !strings.Contains(out, "Team devs now has access to acme/gadgets") {
		t.Fatalf("added=%d, %v, %q", *added, err, out)
	}
}

func TestRemoveRepoFromTeam_Confirmation(t *testing.T) {
	for answer, want := range map[string]int{"": 0, "n": 0, "y": 1, " Yes ": 1} {
		defaults(t)
		_, removed := reposServer(t)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = RemoveRepoFromTeam("acme", "devs", "widgets") }, answer)
		if err != nil || *removed != want {
			t.Errorf("%q: removed=%d, %v", answer, *removed, err)
		}
		if msg := map[int]string{0: "Aborted, team devs still has access to acme/widgets",
			1: "Team devs no longer has access to acme/widgets"}[want]; !strings.Contains(out, msg) {
			t.Errorf("%q: output = %q", answer, out)
		}
	}
}

func TestRepoFunctions_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	for name, call := range map[string]func() *cerr.CustomError{
		"repos":      func() *cerr.CustomError { _, e := ListRepos("acme", "devs", false); return e },
		"addrepo":    func() *cerr.CustomError { return AddRepoToTeam("acme", "devs", "gadgets") },
		"removerepo": func() *cerr.CustomError { return RemoveRepoFromTeam("acme", "devs", "widgets") },
	} {
		testutil.WriteRegistry(t, "")
		if err := call(); err == nil {
			t.Errorf("%s: expected an error without a connection", name)
		}
		testutil.WriteRegistry(t, githubRegistry)
		if err := call(); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: github: got %+v", name, err)
		}
	}

	reposServer(t)
	testutil.Set(t, &shared.QuietOutput, true)
	if _, err := ListRepos("acme", "ghosts", false); err == nil || err.Code != http.StatusNotFound {
		t.Errorf("repos: got %+v", err)
	}
	if err := AddRepoToTeam("acme", "devs", "nope"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to add repository acme/nope to team devs" {
		t.Errorf("add: got %+v", err)
	}
	if err := RemoveRepoFromTeam("acme", "devs", "nope"); err == nil || err.Code != http.StatusNotFound ||
		err.Title != "Unable to remove repository acme/nope from team devs" {
		t.Errorf("remove: got %+v", err)
	}
}
