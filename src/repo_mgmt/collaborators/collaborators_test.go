// gitclt
// Tests the collaborator commands against a fake Forgejo

package collaborators

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func defaults(t *testing.T) {
	t.Helper()
	testutil.Set(t, &OwnerOrg, "")
	testutil.Set(t, &Permission, "write")
	testutil.Set(t, &shared.ConnectionName, "")
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
}

type fake struct {
	calls  []string
	bodies []map[string]any
}

func server(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	base := "/api/v1/repos/acme/web"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base:                                     testutil.Reply(`{"name":"web"}`),
		"GET " + base + "/collaborators":                  testutil.Reply(`[{"login":"bob"},{"login":"carol"}]`),
		"GET " + base + "/collaborators/bob":              testutil.Status(http.StatusNoContent),
		"GET " + base + "/collaborators/carol":            testutil.Status(http.StatusNoContent),
		"GET " + base + "/collaborators/bob/permission":   testutil.Reply(`{"permission":"write"}`),
		"GET " + base + "/collaborators/carol/permission": testutil.Reply(`{"permission":"read"}`),
		"GET " + base + "/collaborators/stranger":         testutil.Status(http.StatusNotFound),
		"PUT " + base + "/collaborators/{user}": func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, "PUT "+r.PathValue("user"))
			testutil.Record(t, &f.bodies, http.StatusNoContent, ``)(w, r)
		},
		"DELETE " + base + "/collaborators/{user}": func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, "DELETE "+r.PathValue("user"))
			w.WriteHeader(http.StatusNoContent)
		},
	})
	return f
}

func TestList(t *testing.T) {
	defaults(t)
	server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = List("web", true) })
	if err != nil || !strings.Contains(out, "bob") || !strings.Contains(out, "read") || strings.Contains(out, "Account ID") {
		t.Fatalf("%v\n%s", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = List("web", true) }); err != nil || out != "bob\ncarol\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	if _, err = List("nope", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("no repo: %+v", err)
	}
}

func TestGet(t *testing.T) {
	defaults(t)
	server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = Get("web", "bob", true) })
	if err != nil || !strings.Contains(out, "write") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err = Get("web", "stranger", false); err == nil || err.Title != "No such collaborator" {
		t.Fatalf("stranger: %+v", err)
	}
}

func TestAdd(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = Add("web", "bob") })
	if err != nil || !strings.Contains(out, "Gave bob write access to acme/web") || f.bodies[0]["permission"] != "write" {
		t.Fatalf("%v, %v\n%s", err, f.bodies, out)
	}
	testutil.Set(t, &Permission, "ADMIN")
	if testutil.Run(t, func() { err = Add("web", "bob") }); err != nil || f.bodies[1]["permission"] != "admin" {
		t.Fatalf("admin: %v, %v", err, f.bodies)
	}

	f.calls = nil
	testutil.Set(t, &Permission, "nonsense")
	if err = Add("web", "bob"); err == nil || err.Title != "Invalid permission" {
		t.Fatalf("invalid: %+v", err)
	}
	// GitHub's only: the library refuses it on Forgejo
	testutil.Set(t, &Permission, "triage")
	if err = Add("web", "bob"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("triage: %+v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("sent %v", f.calls)
	}
}

// The forges take removing a stranger for a success: this must say otherwise, before anything is removed
func TestRemove(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	if err = Remove("web", []string{"bob", "stranger"}); err == nil || err.Title != "No such collaborator" || len(f.calls) != 0 {
		t.Fatalf("stranger: %+v, %v", err, f.calls)
	}
	out := testutil.Run(t, func() { err = Remove("web", []string{"bob"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || !strings.Contains(out, "acme/web") || len(f.calls) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, f.calls, out)
	}
	testutil.Run(t, func() { err = Remove("web", []string{"bob", "carol", "bob"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE bob", "DELETE carol"}) {
		t.Fatalf("yes: %v, %v", err, f.calls)
	}
	f.calls = nil
	testutil.Set(t, &AssumeYes, true)
	if testutil.Run(t, func() { err = Remove("web", []string{"bob"}) }); err != nil || len(f.calls) != 1 {
		t.Fatalf("-y: %v, %v", err, f.calls)
	}
}

func TestInvitations_GitHubOnly(t *testing.T) {
	defaults(t)
	server(t)
	if _, err := Invitations("web", false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("got %+v", err)
	}
}
