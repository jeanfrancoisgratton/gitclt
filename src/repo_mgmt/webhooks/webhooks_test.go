// gitclt
// Tests the webhook commands against a fake Forgejo: the library's own behaviour is checked in the library

package webhooks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const hookFive = `{"id":5,"type":"forgejo","url":"https://ci.example.com/hook","content_type":"json","events":["push"],"active":true,"branch_filter":"main"}`
const hookSix = `{"id":6,"type":"slack","url":"https://mm.example.com/hooks/abc","content_type":"json","events":["push","create","delete","fork","release"],"active":false,
	"metadata":{"channel":"#ci","username":"bot"}}`
const hookAuth = `{"id":7,"type":"forgejo","url":"https://ci.example.com/auth","content_type":"json","events":["push"],"active":true,"authorization_header":"Bearer topsecret"}`

func defaults(t *testing.T) {
	t.Helper()
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &HookActive, &HookInsecureSSL, &HookPromptSecret, &HookPromptAuth, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	for _, s := range []*string{&OwnerOrg, &HookURL, &HookType, &HookContentType, &HookBranchFilter, &HookSecretFile, &HookAuthFile,
		&HookSlackChannel, &HookSlackUsername, &HookSlackIconURL, &HookSlackColor, &shared.ConnectionName} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &HookEvents, nil)
	testutil.Set(t, &shared.DebugMode, true) // GetPassword needs a terminal otherwise
}

// Stands in for cmd.Flags().Changed: only the named flags were passed
func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

func secretFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
	return path
}

// Every kind of webhook lives at its own path; the system and default ones share theirs, and differ by the list
func TestScopes(t *testing.T) {
	for name, tc := range map[string]struct {
		target Target
		base   string
		noun   string
	}{
		"user": {User(), "/api/v1/user/hooks", "user webhook"},
		"repo": {Repo("web"), "/api/v1/repos/acme/web/hooks", "webhook of acme/web"},
		"org":  {Org("corp"), "/api/v1/orgs/corp/hooks", "webhook of corp"},
	} {
		defaults(t)
		testutil.Forgejo(t, map[string]http.HandlerFunc{
			"GET " + tc.base:        testutil.Reply(`[` + hookFive + `]`),
			"GET " + tc.base + "/5": testutil.Reply(hookFive),
		})
		var err *cerr.CustomError
		out := testutil.Run(t, func() { _, err = ListHooks(tc.target, true) })
		if err != nil || !strings.Contains(out, "https://ci.example.com/hook") {
			t.Errorf("%s: list: %v\n%s", name, err, out)
		}
		out = testutil.Run(t, func() { _, err = GetHook(tc.target, "5", true) })
		if err != nil || !strings.Contains(out, "Branch filter:") {
			t.Errorf("%s: get: %v\n%s", name, err, out)
		}
		// A 404 says which webhook it was about
		if _, err = GetHook(tc.target, "9", false); err == nil || err.Code != http.StatusNotFound || !strings.Contains(err.Error(), tc.noun[:7]) {
			t.Errorf("%s: unknown: %+v", name, err)
		}
	}
}

func TestSiteHooks_KindsAreSeparate(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/admin/hooks":   testutil.Reply(`[` + hookFive + `]`),
		"GET /api/v1/admin/hooks/5": testutil.Reply(hookFive),
		"GET /api/v1/admin/hooks/6": testutil.Reply(hookSix),
	})
	if h, err := GetHook(System(), "5", false); err != nil || h.ID != 5 {
		t.Fatalf("system: %+v, %+v", h, err)
	}
	if h, err := GetHook(Default(), "6", false); err != nil || h.ID != 6 {
		t.Fatalf("default: %+v, %+v", h, err)
	}
	// Each refuses the other's
	if _, err := GetHook(System(), "6", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("default ID on system: %+v", err)
	}
	if _, err := GetHook(Default(), "5", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("system ID on default: %+v", err)
	}
	if _, err := ListHooks(Default(), false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("default list: %+v", err)
	}
}

func TestList_Outputs(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/hooks":           testutil.Reply(`[` + hookFive + `,` + hookSix + `]`),
		"GET /api/v1/repos/acme/web/hooks": testutil.Reply(`[]`),
	})
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListHooks(User(), true) })
	// A long event list is cut short in the table
	for _, want := range []string{"Type", "Branch filter", "main", "+2 more"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("table: missing %q: %v\n%s", want, err, out)
		}
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = ListHooks(User(), true) }); err != nil || out != "5\n6\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, false)
	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListHooks(User(), true) }); err != nil || !strings.Contains(out, `"branchFilter": "main"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	testutil.Set(t, &JSONoutput, false)
	if out = testutil.Run(t, func() { _, err = ListHooks(Repo("web"), true) }); err != nil || !strings.Contains(out, "There is no webhook of acme/web") {
		t.Fatalf("empty: %v\n%s", err, out)
	}
}

// A credential is never shown, by any output
func TestGet_HidesTheAuthorizationHeader(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/hooks/7": testutil.Reply(hookAuth)})
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = GetHook(User(), "7", true) })
	if err != nil || !strings.Contains(out, "Authorization header:") || strings.Contains(out, "topsecret") {
		t.Fatalf("%v\n%s", err, out)
	}
	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = GetHook(User(), "7", true) }); err != nil || strings.Contains(out, "topsecret") {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestCreate(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/hooks": testutil.Record(t, &bodies, http.StatusCreated, hookFive)})

	testutil.Set(t, &HookSecretFile, secretFile(t, "s3cr3t\n", 0600))
	testutil.Set(t, &HookAuthFile, secretFile(t, "Bearer x\n", 0600))
	testutil.Set(t, &HookEvents, []string{"push"})
	testutil.Set(t, &HookBranchFilter, "main")
	testutil.Set(t, &HookContentType, "form")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateHook(User(), "https://ci.example.com/hook", passed()) })
	if err != nil || !strings.Contains(out, "Created user webhook 5") {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "s3cr3t") {
		t.Fatalf("the secret was shown:\n%s", out)
	}
	config := bodies[0]["config"].(map[string]any)
	if config["url"] != "https://ci.example.com/hook" || config["secret"] != "s3cr3t" || config["content_type"] != "form" ||
		bodies[0]["branch_filter"] != "main" || bodies[0]["authorization_header"] != "Bearer x" {
		t.Fatalf("sent %v", bodies[0])
	}

	// -q is for scripts: only the ID
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = CreateHook(User(), "https://ci.example.com/hook", passed()) }); err != nil || out != "5\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
}

func TestCreate_Defaults(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"POST /api/v1/admin/hooks": testutil.Record(t, &bodies, http.StatusCreated, hookFive),
		"GET /api/v1/admin/hooks":  testutil.Reply(`[]`),
	})
	var err *cerr.CustomError
	// Default webhooks cannot be listed: the ID must be told
	out := testutil.Run(t, func() { err = CreateHook(Default(), "https://ci.example.com/hook", passed()) })
	if err != nil || !strings.Contains(out, "Keep this ID") {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := fmt.Sprint(bodies[0]["events"]); got != "[push]" || bodies[0]["active"] != true {
		t.Fatalf("sent %v", bodies[0])
	}

	// --active=false, and "all" being the wildcard the library expands
	testutil.Set(t, &HookEvents, []string{"all"})
	testutil.Set(t, &HookActive, false)
	bodies = nil
	testutil.Run(t, func() { err = CreateHook(Default(), "https://ci.example.com/hook", passed("active")) })
	if err != nil || bodies[0]["active"] != false || len(bodies[0]["events"].([]any)) < 5 {
		t.Fatalf("%v, sent %v", err, bodies[0])
	}
}

// Forgejo drops the event names it does not know: the webhook exists, and whoever asked must be told both
func TestCreate_DroppedEvents(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/hooks": testutil.Reply(hookFive)})
	testutil.Set(t, &HookEvents, []string{"push", "nonsense"})
	err := CreateHook(User(), "https://ci.example.com/hook", passed())
	if err == nil || !strings.Contains(err.Error(), "nonsense") || !strings.Contains(err.Error(), "user webhook 5 exists") {
		t.Fatalf("got %+v", err)
	}
}

func TestCreate_NothingSentOnBadInput(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/hooks": testutil.Record(t, &bodies, http.StatusCreated, hookFive)})

	testutil.Set(t, &HookSecretFile, secretFile(t, "s3cr3t", 0644))
	if err := CreateHook(User(), "https://x", passed()); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("readable secret: %+v", err)
	}
	testutil.Set(t, &HookSecretFile, "")
	testutil.Set(t, &HookContentType, "xml")
	if err := CreateHook(User(), "https://x", passed()); err == nil || err.Title != "Invalid content type" {
		t.Errorf("content type: %+v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("sent %v", bodies)
	}
}

func TestEdit(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/hooks/5":   testutil.Reply(hookFive),
		"PATCH /api/v1/user/hooks/5": testutil.Record(t, &bodies, http.StatusOK, hookFive),
		"GET /api/v1/user/hooks/6":   testutil.Reply(hookSix),
		"PATCH /api/v1/user/hooks/6": testutil.Record(t, &bodies, http.StatusOK, hookSix),
	})

	if err := EditHook(User(), "5", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}

	testutil.Set(t, &HookActive, false)
	testutil.Set(t, &HookURL, "https://new.example.com")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditHook(User(), "5", passed("active", "url")) })
	if err != nil || !strings.Contains(out, "Updated user webhook 5") {
		t.Fatalf("%v\n%s", err, out)
	}
	if bodies[0]["active"] != false || bodies[0]["config"].(map[string]any)["url"] != "https://new.example.com" {
		t.Fatalf("sent %v", bodies[0])
	}

	// One Slack setting changes; the others stay
	bodies = nil
	testutil.Set(t, &HookSlackChannel, "#new")
	if testutil.Run(t, func() { err = EditHook(User(), "6", passed("slack-channel")) }); err != nil {
		t.Fatalf("slack: %v", err)
	}
	cfg := bodies[0]["config"].(map[string]any)
	if cfg["channel"] != "#new" || cfg["username"] != "bot" {
		t.Fatalf("sent %v", bodies[0])
	}
	if err = EditHook(User(), "5", passed("slack-channel")); err == nil || err.Title != "Not a Slack webhook" {
		t.Fatalf("not slack: %+v", err)
	}
}

// What Forgejo's API would silently ignore is refused, and nothing changes
func TestEdit_LibraryRefusals(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/hooks/5":   testutil.Reply(hookFive),
		"PATCH /api/v1/user/hooks/5": testutil.Record(t, &bodies, http.StatusOK, hookFive),
	})
	testutil.Set(t, &HookSecretFile, secretFile(t, "s3cr3t", 0600))
	if err := EditHook(User(), "5", passed("secret-file")); err == nil || !strings.Contains(err.Error(), "secret") {
		t.Errorf("secret: %+v", err)
	}
	testutil.Set(t, &HookSecretFile, "")
	testutil.Set(t, &HookEvents, []string{"push", "workflow_run_failure"})
	if err := EditHook(User(), "5", passed("event")); err == nil || !strings.Contains(err.Error(), "workflow_run_failure") {
		t.Errorf("workflow event: %+v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("sent %v", bodies)
	}
}

func TestDelete(t *testing.T) {
	defaults(t)
	var deleted []string
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web/hooks/5": testutil.Reply(hookFive),
		"GET /api/v1/repos/acme/web/hooks/6": testutil.Reply(hookSix),
		"GET /api/v1/repos/acme/web/hooks/9": testutil.Status(http.StatusNotFound),
		"DELETE /api/v1/repos/acme/web/hooks/{id}": func(w http.ResponseWriter, r *http.Request) {
			deleted = append(deleted, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		},
	})
	var err *cerr.CustomError

	// Everything is checked first: 9 does not exist, so 5 stays
	if err = DeleteHooks(Repo("web"), []string{"5", "9"}); err == nil || err.Code != http.StatusNotFound || len(deleted) != 0 {
		t.Fatalf("unknown: %+v, %v", err, deleted)
	}
	if err = DeleteHooks(Repo("web"), []string{"five"}); err == nil || err.Title != "Invalid webhook ID" {
		t.Fatalf("not an ID: %+v", err)
	}

	// The question tells what would go, and where
	out := testutil.Run(t, func() { err = DeleteHooks(Repo("web"), []string{"5"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || !strings.Contains(out, "acme/web") || !strings.Contains(out, "ci.example.com") || len(deleted) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, deleted, out)
	}

	out = testutil.Run(t, func() { err = DeleteHooks(Repo("web"), []string{"5", "6", "5"}) }, "y")
	if err != nil || !slices.Equal(deleted, []string{"5", "6"}) || !strings.Contains(out, "Deleted webhook 6 of acme/web") {
		t.Fatalf("yes: %v, %v\n%s", err, deleted, out)
	}
}

func TestPing(t *testing.T) {
	defaults(t)
	calls := 0
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/repos/acme/web/hooks/5/tests": func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}})
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = PingHook(Repo("web"), "5") })
	if err != nil || calls != 1 || !strings.Contains(out, "Sent a test delivery") {
		t.Fatalf("%v, %d\n%s", err, calls, out)
	}
	// Forgejo has no test delivery for organizations, nor for the other kinds
	if err = PingHook(Org("corp"), "5"); err == nil {
		t.Fatal("org: expected a refusal")
	}
	if err = PingHook(User(), "5"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("user: %+v", err)
	}
}

// Only Forgejo has these kinds: the other forges get a clean refusal, with no request sent
func TestOtherForges(t *testing.T) {
	defaults(t)
	for _, serverType := range []string{"github", "bitbucket"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("%s: request sent: %s %s", serverType, r.Method, r.URL)
		}))
		defer srv.Close()
		testutil.WriteRegistry(t, fmt.Sprintf(`{"auths":{"test":{"url":%q,"token":"abc","username":"acme","owner":"acme","servertype":%q}},"default_connection":"test"}`, srv.URL, serverType))
		for name, target := range map[string]Target{"user": User(), "system": System(), "default": Default()} {
			if _, err := GetHook(target, "5", false); err == nil || err.Title != "Unsupported operation" {
				t.Errorf("%s: %s: %+v", serverType, name, err)
			}
		}
	}
}
