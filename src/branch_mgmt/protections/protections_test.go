// gitclt
// Tests the branch protection commands against a fake Forgejo

package protections

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
)

func defaults(t *testing.T) {
	t.Helper()
	testutil.Set(t, &OwnerOrg, "acme")
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &NoPush, &AnyPush, &AnyMerge, &AnyApprover, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	testutil.Set(t, &RequiredApprovals, 0)
}

// Stands for cmd.Flags().Changed
func flags(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

const mainRule = `{"rule_name":"main","required_approvals":2,"dismiss_stale_approvals":true,"enable_status_check":true,
	"status_check_contexts":["ci/build"],"require_signed_commits":true,"enable_push":true,"enable_push_whitelist":true,
	"push_whitelist_usernames":["alice"],"push_whitelist_teams":["ops"],"enable_merge_whitelist":true,
	"merge_whitelist_usernames":["bob"],"merge_whitelist_teams":[],"protected_file_patterns":"*.lock"}`

const releaseRule = `{"rule_name":"release/*","enable_push":false}`

const repoReply = `{"name":"web","owner":{"login":"acme"}}`

func TestListProtections(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                    testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections": testutil.Reply(`[` + mainRule + `,` + releaseRule + `]`),
	})

	list, ce := ListProtections("web", false)
	if ce != nil || len(list) != 2 || list[0].Pattern != "main" || list[1].Pattern != "release/*" {
		t.Fatalf("got %+v, %v", list, ce)
	}

	out := testutil.Run(t, func() { _, ce = ListProtections("web", true) })
	for _, want := range []string{"main", "release/*", "ci/build", "users: alice; teams: ops", "nobody", "blocked"} {
		if ce != nil || !strings.Contains(out, want) {
			t.Errorf("missing %q (%v) in:\n%s", want, ce, out)
		}
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, ce = ListProtections("web", true) }); ce != nil || out != "main\nrelease/*\n" {
		t.Errorf("-q: %q, %v", out, ce)
	}
	testutil.Set(t, &shared.QuietOutput, false)

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, ce = ListProtections("web", true) }); ce != nil || !strings.Contains(out, `"pattern": "release/*"`) {
		t.Errorf("-j: %q, %v", out, ce)
	}
}

func TestListProtections_EmptyAndErrors(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                    testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections": testutil.Reply(`[]`),
	})
	var err error
	out := testutil.Run(t, func() {
		if _, ce := ListProtections("web", true); ce != nil {
			err = ce
		}
	})
	if err != nil || !strings.Contains(out, "acme/web has no protection rule") {
		t.Errorf("empty: %q, %v", out, err)
	}
	if _, ce := ListProtections("nope", false); ce == nil || ce.Title != "No such repository" {
		t.Errorf("no repo: %+v", ce)
	}

	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                    testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections": testutil.Status(http.StatusForbidden),
	})
	if _, ce := ListProtections("web", false); ce == nil || ce.Code != http.StatusForbidden {
		t.Errorf("403: %+v", ce)
	}
}

// A pattern with a slash is a single path segment
func TestGetProtection(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections/{name}": func(w http.ResponseWriter, r *http.Request) {
			switch r.PathValue("name") {
			case "main":
				testutil.Reply(mainRule)(w, r)
			case "release/*":
				testutil.Reply(releaseRule)(w, r)
			default:
				testutil.Status(http.StatusNotFound)(w, r)
			}
		},
	})

	var err error
	out := testutil.Run(t, func() {
		if _, ce := GetProtection("web", "main", true); ce != nil {
			err = ce
		}
	})
	for _, want := range []string{"Required approvals:", "users: alice; teams: ops", "users: bob", "*.lock", "Protected files:"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("missing %q (%v) in:\n%s", want, err, out)
		}
	}
	if strings.Contains(out, "Linear history") {
		t.Errorf("a GitHub setting on Forgejo:\n%s", out)
	}

	if rule, ce := GetProtection("web", "release/*", false); ce != nil || rule.PushAllowlist == nil || len(rule.PushAllowlist.Users) != 0 {
		t.Errorf("release/*: %+v, %v", rule, ce)
	}
	if _, ce := GetProtection("web", "nope", false); ce == nil || ce.Title != "No such protection rule" || ce.Code != http.StatusNotFound {
		t.Errorf("404: %+v", ce)
	}
}

func TestCreateProtection(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                     testutil.Reply(repoReply),
		"POST /api/v1/repos/acme/web/branch_protections": testutil.Record(t, &bodies, http.StatusCreated, `{"rule_name":"main"}`),
	})
	testutil.Set(t, &RequiredApprovals, 2)
	testutil.Set(t, &StatusChecks, []string{"test (1.27, linux)", ""})
	testutil.Set(t, &PushUsers, []string{"alice"})
	// Not passed, so left out
	testutil.Set(t, &RequireSignedCommits, true)

	var err error
	out := testutil.Run(t, func() {
		if ce := CreateProtection("web", "main", flags("approvals", "status-check", "push-users")); ce != nil {
			err = ce
		}
	})
	if err != nil || !strings.Contains(out, "Created protection rule main on acme/web") {
		t.Fatalf("%q, %v", out, err)
	}
	b := bodies[0]
	if b["rule_name"] != "main" || b["required_approvals"] != float64(2) || b["require_signed_commits"] != false ||
		b["enable_push_whitelist"] != true || b["enable_status_check"] != true {
		t.Errorf("body: %v", b)
	}
	if checks := b["status_check_contexts"].([]any); len(checks) != 1 || checks[0] != "test (1.27, linux)" {
		t.Errorf("checks: %v", checks)
	}
	if users := b["push_whitelist_usernames"].([]any); len(users) != 1 || users[0] != "alice" {
		t.Errorf("users: %v", users)
	}
}

// Without flags, the rule is created as is: an open push, a blocked deletion
func TestCreateProtection_NoFlags(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                     testutil.Reply(repoReply),
		"POST /api/v1/repos/acme/web/branch_protections": testutil.Record(t, &bodies, http.StatusCreated, `{"rule_name":"main"}`),
	})
	testutil.Set(t, &shared.QuietOutput, true)
	var err error
	out := testutil.Run(t, func() {
		if ce := CreateProtection("web", "main", flags()); ce != nil {
			err = ce
		}
	})
	if err != nil || out != "" || bodies[0]["enable_push_whitelist"] != false {
		t.Errorf("%q, %v, %v", out, err, bodies)
	}
}

func TestCreateProtection_Refusals(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                          testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections/main":  testutil.Reply(mainRule),
		"POST /api/v1/repos/acme/web/branch_protections":      testutil.Record(t, &bodies, http.StatusCreated, `{}`),
		"GET /api/v1/repos/acme/web/branch_protections/other": testutil.Status(http.StatusNotFound),
	})
	if ce := CreateProtection("web", "main", flags()); ce == nil || ce.Code != http.StatusConflict {
		t.Errorf("exists: %+v", ce)
	}

	// The library refuses what Forgejo does not have
	testutil.Set(t, &RequireLinearHistory, true)
	if ce := CreateProtection("web", "other", flags("linear-history")); ce == nil || !strings.Contains(ce.Error(), "linear history") {
		t.Errorf("not supported: %+v", ce)
	}
	testutil.Set(t, &RequireLinearHistory, false)

	testutil.Set(t, &RequiredApprovals, -1)
	if ce := CreateProtection("web", "other", flags("approvals")); ce == nil || ce.Title != "Invalid approval count" {
		t.Errorf("negative: %+v", ce)
	}
	if len(bodies) != 0 {
		t.Errorf("nothing should have been created: %v", bodies)
	}
}

func editServer(t *testing.T, bodies *[]map[string]any) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":                           testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections/main":   testutil.Reply(mainRule),
		"PATCH /api/v1/repos/acme/web/branch_protections/main": testutil.Record(t, bodies, http.StatusOK, mainRule),
	})
}

// What was not passed keeps the server's value: the library's Update replaces the whole rule
func TestEditProtection_KeepsTheRest(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	editServer(t, &bodies)
	testutil.Set(t, &RequiredApprovals, 5)
	testutil.Set(t, &PushTeams, []string{"dev"})

	var err error
	out := testutil.Run(t, func() {
		if ce := EditProtection("web", "main", flags("approvals", "push-teams")); ce != nil {
			err = ce
		}
	})
	if err != nil || !strings.Contains(out, "Updated protection rule main on acme/web") {
		t.Fatalf("%q, %v", out, err)
	}
	b := bodies[0]
	if b["required_approvals"] != float64(5) || b["dismiss_stale_approvals"] != true || b["require_signed_commits"] != true ||
		b["protected_file_patterns"] != "*.lock" || b["enable_merge_whitelist"] != true {
		t.Errorf("body: %v", b)
	}
	if u, tm := b["push_whitelist_usernames"].([]any), b["push_whitelist_teams"].([]any); len(u) != 1 || u[0] != "alice" || len(tm) != 1 || tm[0] != "dev" {
		t.Errorf("the users should stay, the teams change: %v, %v", u, tm)
	}
	if checks := b["status_check_contexts"].([]any); len(checks) != 1 || checks[0] != "ci/build" {
		t.Errorf("checks: %v", checks)
	}
	if _, set := b["rule_name"]; set {
		t.Errorf("rule_name is not accepted on edit: %v", b)
	}
}

func TestEditProtection_Clears(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	editServer(t, &bodies)
	testutil.Set(t, &AnyPush, true)
	testutil.Set(t, &AnyMerge, true)
	testutil.Set(t, &StatusChecks, []string{""})
	testutil.Set(t, &RequireSignedCommits, false)

	if ce := EditProtection("web", "main", flags("any-push", "any-merge", "status-check", "signed")); ce != nil {
		t.Fatal(ce.Error())
	}
	b := bodies[0]
	if b["enable_push_whitelist"] != false || b["enable_merge_whitelist"] != false || b["enable_status_check"] != false || b["require_signed_commits"] != false {
		t.Errorf("body: %v", b)
	}
}

func TestEditProtection_NoPush(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	editServer(t, &bodies)
	testutil.Set(t, &NoPush, true)
	if ce := EditProtection("web", "main", flags("no-push")); ce != nil {
		t.Fatal(ce.Error())
	}
	if b := bodies[0]; b["enable_push_whitelist"] != true || len(b["push_whitelist_usernames"].([]any)) != 0 || len(b["push_whitelist_teams"].([]any)) != 0 {
		t.Errorf("body: %v", b)
	}
}

func TestEditProtection_Refusals(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	editServer(t, &bodies)

	if ce := EditProtection("web", "main", flags()); ce == nil || ce.Title != "Nothing to edit" {
		t.Errorf("no flag: %+v", ce)
	}
	// --no-push=false asks for nothing
	if ce := EditProtection("web", "main", flags("no-push")); ce == nil || ce.Title != "Nothing to edit" {
		t.Errorf("--no-push=false: %+v", ce)
	}
	if ce := EditProtection("web", "nope", flags("signed")); ce == nil || ce.Title != "No such protection rule" {
		t.Errorf("no rule: %+v", ce)
	}
	if len(bodies) != 0 {
		t.Errorf("nothing should have been sent: %v", bodies)
	}
}

func deleteServer(t *testing.T, deletes *[]string, failOn string) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(repoReply),
		"GET /api/v1/repos/acme/web/branch_protections/{name}": func(w http.ResponseWriter, r *http.Request) {
			if n := r.PathValue("name"); n == "main" || n == "release/*" {
				testutil.Reply(`{"rule_name":"`+n+`"}`)(w, r)
				return
			}
			testutil.Status(http.StatusNotFound)(w, r)
		},
		"DELETE /api/v1/repos/acme/web/branch_protections/{name}": func(w http.ResponseWriter, r *http.Request) {
			if r.PathValue("name") == failOn {
				testutil.Status(http.StatusForbidden)(w, r)
				return
			}
			*deletes = append(*deletes, r.PathValue("name"))
			w.WriteHeader(http.StatusNoContent)
		},
	})
}

func TestDeleteProtections(t *testing.T) {
	defaults(t)
	var deletes []string
	deleteServer(t, &deletes, "")

	// Asked once for the lot; anything but yes aborts
	var err error
	out := testutil.Run(t, func() {
		if ce := DeleteProtections("web", []string{"main", "release/*"}); ce != nil {
			err = ce
		}
	}, "n")
	if err != nil || len(deletes) != 0 || !strings.Contains(out, "Aborted") {
		t.Fatalf("no: %q, %v, %v", out, err, deletes)
	}

	testutil.Run(t, func() {
		if ce := DeleteProtections("web", []string{"main", "release/*", "main"}); ce != nil {
			err = ce
		}
	}, "y")
	if err != nil || !slices.Equal(deletes, []string{"main", "release/*"}) {
		t.Fatalf("yes: %v, %v", err, deletes)
	}

	deletes = nil
	testutil.Set(t, &AssumeYes, true)
	testutil.Run(t, func() {
		if ce := DeleteProtections("web", []string{"main"}); ce != nil {
			err = ce
		}
	})
	if err != nil || !slices.Equal(deletes, []string{"main"}) {
		t.Fatalf("-y: %v, %v", err, deletes)
	}
}

// Every rule is checked first: a missing one means nothing is deleted
func TestDeleteProtections_MissingRule(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	var deletes []string
	deleteServer(t, &deletes, "")
	ce := DeleteProtections("web", []string{"main", "nope", "gone"})
	if ce == nil || ce.Code != http.StatusNotFound || !strings.Contains(ce.Error(), "nope, gone") || len(deletes) != 0 {
		t.Fatalf("%+v, %v", ce, deletes)
	}
}

func TestDeleteProtections_StopsAtFirstFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	var deletes []string
	deleteServer(t, &deletes, "release/*")
	var err error
	out := testutil.Run(t, func() {
		if ce := DeleteProtections("web", []string{"main", "release/*"}); ce != nil {
			err = ce
		}
	})
	if err == nil || !slices.Equal(deletes, []string{"main"}) || !strings.Contains(out, "Deleted protection rule main") {
		t.Fatalf("%q, %v, %v", out, err, deletes)
	}
}
