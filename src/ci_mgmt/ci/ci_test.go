// gitclt
// Tests the secret and variable commands against a fake Forgejo

package ci

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &ScopeOrg, &ScopeEnv, &ValueFile, &shared.ConnectionName} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &ScopeUser, &ValueStdin, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	testutil.Set(t, &shared.DebugMode, true) // GetPassword needs a terminal otherwise
}

func valueFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "value")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
	return path
}

type fake struct {
	calls  []string
	bodies []map[string]any
}

// A fake server, whose secrets and variables live under these bases
func server(t *testing.T, bases ...string) *fake {
	t.Helper()
	f := &fake{}
	write := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/api/v1/"))
			if r.Method == http.MethodDelete {
				w.WriteHeader(status)
				return
			}
			testutil.Record(t, &f.bodies, status, `{}`)(w, r)
		}
	}
	routes := map[string]http.HandlerFunc{"GET /api/v1/repos/acme/web": testutil.Reply(`{"name":"web"}`)}
	for _, base := range bases {
		routes["GET "+base+"/actions/secrets"] = testutil.Reply(`[{"name":"API_TOKEN","created_at":"2026-11-01T10:00:00Z"},{"name":"OTHER"}]`)
		routes["PUT "+base+"/actions/secrets/{name}"] = write(http.StatusNoContent)
		routes["DELETE "+base+"/actions/secrets/{name}"] = write(http.StatusNoContent)
		routes["GET "+base+"/actions/variables"] = testutil.Reply(`[{"name":"REGION","data":"eu"},{"name":"CONFIG","data":"line one\nline two"}]`)
		routes["GET "+base+"/actions/variables/REGION"] = testutil.Reply(`{"name":"REGION","data":"eu"}`)
		routes["GET "+base+"/actions/variables/NOSUCH"] = testutil.Status(http.StatusNotFound)
		routes["PUT "+base+"/actions/variables/{name}"] = write(http.StatusNoContent)
		routes["DELETE "+base+"/actions/variables/{name}"] = write(http.StatusNoContent)
	}
	testutil.Forgejo(t, routes)
	return f
}

const repoBase = "/api/v1/repos/acme/web"

func TestSecrets_List(t *testing.T) {
	defaults(t)
	server(t, repoBase)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListSecrets("web", true) })
	if err != nil || !strings.Contains(out, "API_TOKEN") || !strings.Contains(out, "Created") {
		t.Fatalf("%v\n%s", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = ListSecrets("web", true) }); err != nil || out != "API_TOKEN\nOTHER\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	if _, err = ListSecrets("nope", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("no repo: %+v", err)
	}
}

// The three ways to give a value; none of them may show it
func TestSecrets_Set(t *testing.T) {
	defaults(t)
	f := server(t, repoBase)
	var err *cerr.CustomError

	testutil.Set(t, &ValueFile, valueFile(t, "from-a-file\n", 0600))
	out := testutil.Run(t, func() { err = SetSecret("web", "API_TOKEN") })
	if err != nil || !strings.Contains(out, "Set secret API_TOKEN on acme/web") || strings.Contains(out, "from-a-file") {
		t.Fatalf("file: %v\n%s", err, out)
	}
	if f.bodies[0]["data"] != "from-a-file" || !slices.Equal(f.calls, []string{"PUT repos/acme/web/actions/secrets/API_TOKEN"}) {
		t.Fatalf("file: sent %v, %v", f.bodies, f.calls)
	}

	testutil.Set(t, &ValueFile, "")
	out = testutil.Run(t, func() { err = SetSecret("web", "TYPED") }, "typed-value")
	if err != nil || f.bodies[1]["data"] != "typed-value" {
		t.Fatalf("prompt: %v, %v\n%s", err, f.bodies, out)
	}

	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	_, _ = w.WriteString("from-stdin\n")
	_ = w.Close()
	if value, ce := readValue(r, "the standard input"); ce != nil || value != "from-stdin" {
		t.Fatalf("stdin: %q, %v", value, ce)
	}
}

// Nothing is sent if the value cannot be read as it is, and the file of a secret must be private
func TestSecrets_BadValues(t *testing.T) {
	defaults(t)
	f := server(t, repoBase)
	for name, tc := range map[string]struct {
		path string
		want string
	}{
		"readable by others": {valueFile(t, "x", 0644), "chmod 600"},
		"empty":              {valueFile(t, "\n", 0600), "is empty"},
		"too large":          {valueFile(t, strings.Repeat("a", maxValueSize+1), 0600), "holds more than"},
		"directory":          {t.TempDir(), "not a regular file"},
		"missing":            {filepath.Join(t.TempDir(), "nope"), "no such file"},
	} {
		testutil.Set(t, &ValueFile, tc.path)
		if err := SetSecret("web", "X"); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %+v", name, err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("sent %v", f.calls)
	}
}

// The prompt comes after the connection: nobody types a secret for a repository that does not exist
func TestSecrets_NoPromptForAMissingRepo(t *testing.T) {
	defaults(t)
	server(t, repoBase)
	var err *cerr.CustomError
	testutil.Run(t, func() { err = SetSecret("nope", "X") }) // an unused answer, or a prompt without one, fails the test
	if err == nil || err.Title != "No such repository" {
		t.Fatalf("got %+v", err)
	}
}

func TestSecrets_Delete(t *testing.T) {
	defaults(t)
	f := server(t, repoBase)
	var err *cerr.CustomError
	if err = DeleteSecrets("web", []string{"API_TOKEN", "NOSUCH"}); err == nil || err.Title != "No such secret" || len(f.calls) != 0 {
		t.Fatalf("unknown: %+v, %v", err, f.calls)
	}
	out := testutil.Run(t, func() { err = DeleteSecrets("web", []string{"API_TOKEN"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || !strings.Contains(out, "acme/web") || len(f.calls) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, f.calls, out)
	}
	// Names ignore case, and repeats count once
	testutil.Run(t, func() { err = DeleteSecrets("web", []string{"api_token", "api_token"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE repos/acme/web/actions/secrets/api_token"}) {
		t.Fatalf("yes: %v, %v", err, f.calls)
	}
}

func TestScopes(t *testing.T) {
	defaults(t)
	f := server(t, "/api/v1/orgs/corp", "/api/v1/user")
	testutil.Set(t, &ValueFile, valueFile(t, "v", 0600))

	testutil.Set(t, &ScopeOrg, "corp")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = SetSecret("", "X") })
	if err != nil || !strings.Contains(out, "on corp") || f.calls[0] != "PUT orgs/corp/actions/secrets/X" {
		t.Fatalf("org: %v, %v\n%s", err, f.calls, out)
	}

	testutil.Set(t, &ScopeOrg, "")
	testutil.Set(t, &ScopeUser, true)
	out = testutil.Run(t, func() { err = SetSecret("", "X") })
	if err != nil || !strings.Contains(out, "on your account") || f.calls[1] != "PUT user/actions/secrets/X" {
		t.Fatalf("user: %v, %v\n%s", err, f.calls, out)
	}

	// Forgejo has no environments
	testutil.Set(t, &ScopeUser, false)
	testutil.Set(t, &ScopeEnv, "prod")
	server(t, repoBase)
	if _, err = ListSecrets("web", false); err == nil || !strings.Contains(err.Error(), "no environments") {
		t.Fatalf("env: %+v", err)
	}
}

func TestVariables(t *testing.T) {
	defaults(t)
	f := server(t, repoBase)
	var err *cerr.CustomError

	out := testutil.Run(t, func() { _, err = ListVariables("web", true) })
	// A long or multi-line value is cut short in the table
	if err != nil || !strings.Contains(out, "REGION") || !strings.Contains(out, "line one...") || strings.Contains(out, "line two") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if out = testutil.Run(t, func() { _, err = GetVariable("web", "REGION", true) }); err != nil || !strings.Contains(out, "eu") {
		t.Fatalf("get: %v\n%s", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = GetVariable("web", "REGION", true) }); err != nil || out != "eu\n" {
		t.Fatalf("get -q: %v\n%q", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, false)
	if _, err = GetVariable("web", "NOSUCH", false); err == nil || err.Title != "No such variable" {
		t.Fatalf("unknown: %+v", err)
	}

	if testutil.Run(t, func() { err = SetVariable("web", "REGION", "us") }); err != nil || f.bodies[0]["value"] != "us" {
		t.Fatalf("set: %v, %v", err, f.bodies)
	}
	// From a file: a variable's need not be private, and it keeps its inner lines
	testutil.Set(t, &ValueFile, valueFile(t, "a\nb\n", 0644))
	if testutil.Run(t, func() { err = SetVariable("web", "CONFIG", "") }); err != nil || f.bodies[1]["value"] != "a\nb" {
		t.Fatalf("file: %v, %v", err, f.bodies)
	}
	testutil.Set(t, &ValueFile, "")

	f.calls = nil
	if err = DeleteVariables("web", []string{"REGION", "NOSUCH"}); err == nil || err.Title != "No such variable" || len(f.calls) != 0 {
		t.Fatalf("unknown: %+v, %v", err, f.calls)
	}
	testutil.Run(t, func() { err = DeleteVariables("web", []string{"REGION"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE repos/acme/web/actions/variables/REGION"}) {
		t.Fatalf("delete: %v, %v", err, f.calls)
	}
}

func TestShortValue(t *testing.T) {
	for in, want := range map[string]string{"eu": "eu", "a\nb": "a...", strings.Repeat("é", 61): strings.Repeat("é", 60) + "..."} {
		if got := shortValue(in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}

// GitHub and Bitbucket have no user scope: the library says so
func TestGitHub_NoUserScope(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"gh":{"url":"https://github.com","token":"x","servertype":"github"}},"default_connection":"gh"}`)
	testutil.Set(t, &ScopeUser, true)
	if _, err := ListSecrets("", false); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("got %+v", err)
	}
}
