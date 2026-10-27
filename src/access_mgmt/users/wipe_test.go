// gitclt
// Tests user wipe: the checks done before anything changes, the plan, the confirmation, and partial failures

package users

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

type fakeTeam struct {
	name, perm string
	members    []string
}

type fakeOrg struct {
	teams []fakeTeam
	repos []string
}

// A fake Forgejo where jfg (the connection's user) wipes bob
// orgs are bob's organizations; every DELETE is recorded in *calls, and the one on failOn answers 500
type wipeForge struct {
	admin     bool
	orgs      map[string]fakeOrg
	userRepos []string
	failOn    string
	calls     []string
}

func reply(t *testing.T, v any) http.HandlerFunc {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return testutil.Reply(string(b))
}

func repoList(owner string, names []string) []map[string]any {
	l := []map[string]any{}
	for _, n := range names {
		l = append(l, map[string]any{"name": n, "full_name": owner + "/" + n, "owner": map[string]string{"login": owner}})
	}
	return l
}

func (f *wipeForge) start(t *testing.T) {
	t.Helper()
	del := func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, r.URL.Path+"?"+r.URL.RawQuery)
		if r.URL.Path == f.failOn {
			testutil.Status(http.StatusInternalServerError)(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	routes := map[string]http.HandlerFunc{
		"GET /api/v1/user":                      testutil.Reply(fmt.Sprintf(`{"id":4,"login":"jfg","is_admin":%t}`, f.admin)),
		"GET /api/v1/users/bob":                 testutil.Reply(`{"id":7,"login":"bob"}`),
		"GET /api/v1/users/bob/repos":           reply(t, repoList("bob", f.userRepos)),
		"DELETE /api/v1/admin/users/bob":        del,
		"GET /api/v1/orgs/{org}":                testutil.Status(http.StatusNotFound), // bob is a user, not an org
		"DELETE /api/v1/orgs/{org}":             del,
		"DELETE /api/v1/orgs/{org}/members/bob": del,
	}
	orgs := []map[string]any{}
	id := int64(100)
	for _, name := range slices.Sorted(maps.Keys(f.orgs)) {
		org := f.orgs[name]
		orgs = append(orgs, map[string]any{"id": len(orgs) + 1, "username": name})
		routes["GET /api/v1/orgs/"+name] = testutil.Reply(`{"username":"` + name + `"}`)
		routes["GET /api/v1/orgs/"+name+"/repos"] = reply(t, repoList(name, org.repos))
		teams := []map[string]any{}
		for _, team := range org.teams {
			id++
			teams = append(teams, map[string]any{"id": id, "name": team.name, "permission": team.perm})
			users := []map[string]any{}
			for _, m := range team.members {
				users = append(users, map[string]any{"login": m})
			}
			routes[fmt.Sprintf("GET /api/v1/teams/%d/members", id)] = reply(t, users)
		}
		routes["GET /api/v1/orgs/"+name+"/teams"] = reply(t, teams)
	}
	routes["GET /api/v1/users/bob/orgs"] = reply(t, orgs)
	testutil.Forgejo(t, routes)
}

// acme: bob is a plain member of devs; sandbox: bob is alone in it; shared: bob co-owns it with alice
func happyForge() *wipeForge {
	return &wipeForge{admin: true, orgs: map[string]fakeOrg{
		"acme": {teams: []fakeTeam{
			{"Owners", "owner", []string{"alice"}},
			{"devs", "write", []string{"alice", "bob"}},
		}, repos: []string{"site"}},
		"sandbox": {teams: []fakeTeam{{"Owners", "owner", []string{"bob"}}}},
		"shared":  {teams: []fakeTeam{{"Owners", "owner", []string{"Alice", "bob"}}}},
	}}
}

func TestWipeUser(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	f := happyForge()
	f.start(t)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = WipeUser("bob") })
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	want := []string{
		"/api/v1/orgs/acme/members/bob?",
		"/api/v1/orgs/sandbox?",
		"/api/v1/orgs/shared/members/bob?",
		"/api/v1/admin/users/bob?",
	}
	if !slices.Equal(f.calls, want) {
		t.Fatalf("calls = %q", f.calls)
	}
	for _, s := range []string{"remove them from acme (teams: devs)", "delete organization sandbox: bob is its only member",
		"remove them from shared (teams: Owners)", "Deleted org sandbox", "Deleted user bob"} {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in %q", s, out)
		}
	}
}

func TestWipeUser_Blocked(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	f := &wipeForge{admin: true, userRepos: []string{"dotfiles", "notes"}, orgs: map[string]fakeOrg{
		"lastowner": {teams: []fakeTeam{
			{"Owners", "owner", []string{"bob"}},
			{"devs", "write", []string{"carol", "dave"}},
		}},
		"solo": {teams: []fakeTeam{{"Owners", "owner", []string{"bob"}}}, repos: []string{"legacy"}},
	}}
	f.start(t)

	err := WipeUser("bob")
	if err == nil || err.Title != "Cannot wipe user bob" || len(f.calls) != 0 {
		t.Fatalf("calls=%q, %+v", f.calls, err)
	}
	for _, s := range []string{"nothing was changed",
		"lastowner: bob is its last owner, and it still has 2 other member(s)",
		"solo: bob is its only member, but it still owns 1 repositories (legacy)",
		"bob owns 2 repositories (dotfiles, notes); pass --purge"} {
		if !strings.Contains(err.Message, s) {
			t.Errorf("missing %q in %q", s, err.Message)
		}
	}

	// --purge lifts the user's own repos, never an org's
	testutil.Set(t, &PurgeUser, true)
	err = WipeUser("bob")
	if err == nil || strings.Contains(err.Message, "dotfiles") || !strings.Contains(err.Message, "legacy") {
		t.Fatalf("purge: %+v", err)
	}
}

func TestWipeUser_Purge(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &PurgeUser, true)
	f := &wipeForge{admin: true, userRepos: []string{"dotfiles"}}
	f.start(t)

	out := testutil.Run(t, func() { _ = WipeUser("bob") })
	if !slices.Equal(f.calls, []string{"/api/v1/admin/users/bob?purge=true"}) || !strings.Contains(out, "purge all of their content") {
		t.Fatalf("calls=%q, %q", f.calls, out)
	}
}

func TestWipeUser_ConfirmationAndDryRun(t *testing.T) {
	defaults(t)
	f := happyForge()
	f.start(t)

	out := testutil.Run(t, func() { _ = WipeUser("bob") }, "n")
	if len(f.calls) != 0 || !strings.Contains(out, "Wiping user bob will:") || !strings.Contains(out, "Aborted") {
		t.Fatalf("calls=%q, %q", f.calls, out)
	}

	// --dry-run: the plan, no prompt (testutil.Run fails the test on an unanswered prompt), even with --quiet
	testutil.Set(t, &WipeDryRun, true)
	testutil.Set(t, &shared.QuietOutput, true)
	out = testutil.Run(t, func() { _ = WipeUser("bob") })
	if len(f.calls) != 0 || !strings.Contains(out, "delete user bob") {
		t.Fatalf("dry run: calls=%q, %q", f.calls, out)
	}

	// --quiet --yes: no output at all
	testutil.Set(t, &WipeDryRun, false)
	testutil.Set(t, &AssumeYes, true)
	if out = testutil.Run(t, func() { _ = WipeUser("bob") }); len(f.calls) != 4 || out != "" {
		t.Fatalf("quiet: calls=%q, %q", f.calls, out)
	}
}

func TestWipeUser_Refusals(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)

	f := happyForge()
	f.admin = false
	f.start(t)
	if err := WipeUser("bob"); err == nil || err.Title != "Admin rights required" || len(f.calls) != 0 {
		t.Fatalf("not admin: calls=%q, %+v", f.calls, err)
	}

	f = happyForge()
	f.start(t)
	if err := WipeUser("JFG"); err == nil || err.Title != "Cannot wipe yourself" {
		t.Fatalf("self: %+v", err)
	}
	if err := WipeUser("nobody"); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("404: %+v", err)
	}
}

func TestWipeUser_PartialFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &shared.QuietOutput, true)
	f := happyForge()
	f.failOn = "/api/v1/orgs/shared/members/bob"
	f.start(t)

	err := WipeUser("bob")
	if err == nil || err.Code != http.StatusInternalServerError || len(f.calls) != 3 ||
		!strings.Contains(err.Message, "already done: removed from acme, deleted org sandbox") {
		t.Fatalf("calls=%q, %+v", f.calls, err)
	}
}
