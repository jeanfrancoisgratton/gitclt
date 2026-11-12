// gitclt
// Tests the runner commands against a fake Forgejo

package runners

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
	for _, s := range []*string{&OwnerOrg, &OrgScope, &RegisterName, &RegisterDescription, &shared.ConnectionName} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &UserScope, &InstanceScope, &Inherited, &RegisterEphemeral, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	testutil.Set(t, &RegisterLabels, nil)
}

const (
	idle    = `{"id":3,"uuid":"u-3","name":"build-1","status":"idle","labels":["self"],"version":"v12.1.0"}`
	busy    = `{"id":4,"uuid":"u-4","name":"build-2","status":"active","labels":["self"],"ephemeral":true}`
	offline = `{"id":5,"uuid":"u-5","name":"old","status":"offline"}`
)

type fake struct{ calls []string }

// A fake Forgejo with the same runners under every scope; calls records the requests, with their scope's path
func server(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	record := func(status int, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			call := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/api/v1/")
			if r.URL.RawQuery != "" {
				call += "?" + r.URL.RawQuery
			}
			f.calls = append(f.calls, call)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}
	}
	routes := map[string]http.HandlerFunc{}
	for _, base := range []string{"/api/v1/repos/acme/web/actions/runners", "/api/v1/orgs/devops/actions/runners",
		"/api/v1/user/actions/runners", "/api/v1/admin/actions/runners"} {
		routes["GET "+base] = record(http.StatusOK, `[`+idle+`,`+busy+`,`+offline+`]`)
		routes["GET "+base+"/3"] = record(http.StatusOK, idle)
		routes["GET "+base+"/4"] = record(http.StatusOK, busy)
		routes["GET "+base+"/9"] = record(http.StatusNotFound, `{"message":"not found"}`)
		routes["POST "+base] = record(http.StatusCreated, `{"id":6,"uuid":"u-6","token":"s3cr3t"}`)
		routes["DELETE "+base+"/{id}"] = record(http.StatusNoContent, ``)
		routes["GET "+base+"/registration-token"] = record(http.StatusOK, `{"token":"reg-t0ken"}`)
	}
	testutil.Forgejo(t, routes)
	return f
}

func TestListRunners(t *testing.T) {
	defaults(t)
	f := server(t)

	list, err := ListRunners("web", false)
	if err != nil || len(list) != 3 || state(list[0]) != "idle" || state(list[1]) != "busy" || state(list[2]) != "offline" {
		t.Fatalf("%v, %+v", err, list)
	}

	out := testutil.Run(t, func() { _, err = ListRunners("web", true) })
	for _, want := range []string{"build-1", "idle", "busy", "offline", "v12.1.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = ListRunners("web", true) }); out != "3\n4\n5\n" {
		t.Fatalf("-q: %q", out)
	}

	// --inherited is Forgejo's visible=true
	testutil.Set(t, &Inherited, true)
	f.calls = nil
	if _, err = ListRunners("web", false); err != nil || len(f.calls) != 1 || !strings.Contains(f.calls[0], "visible=true") {
		t.Fatalf("%v, calls %v", err, f.calls)
	}
}

// --org, --user and --instance pick the scope; a repository with one of them is refused before connecting
func TestScopes(t *testing.T) {
	for _, tc := range []struct {
		set  func()
		path string
	}{
		{func() { OrgScope = "devops" }, "orgs/devops/actions/runners"},
		{func() { UserScope = true }, "user/actions/runners"},
		{func() { InstanceScope = true }, "admin/actions/runners"},
		{func() {}, "repos/acme/web/actions/runners"},
	} {
		defaults(t)
		f := server(t)
		tc.set()
		repo := ""
		if tc.path == "repos/acme/web/actions/runners" {
			repo = "web"
		}
		if _, err := ListRunners(repo, false); err != nil || len(f.calls) != 1 || !strings.HasPrefix(f.calls[0], "GET "+tc.path) {
			t.Errorf("%s: %v, calls %v", tc.path, err, f.calls)
		}
	}

	defaults(t)
	testutil.Set(t, &OrgScope, "devops")
	testutil.WriteRegistry(t, "")
	if _, err := ListRunners("web", false); err == nil || err.Title != "Too many scopes" {
		t.Fatalf("got %+v", err)
	}
}

func TestGetRunner(t *testing.T) {
	defaults(t)
	server(t)
	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { _, err = GetRunner("web", "4", true) }))
	for _, want := range []string{"UUID:", "u-4", "build-2", "busy (active)", "Ephemeral:"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("%v, missing %q in:\n%s", err, want, out)
		}
	}
	if _, err = GetRunner("web", "9", false); err == nil || err.Code != http.StatusNotFound || !strings.Contains(err.Message, "--org") {
		t.Fatalf("missing runner: %+v", err)
	}
	if _, err = GetRunner("web", "x", false); err == nil || err.Title != "Invalid runner ID" {
		t.Fatalf("bad ID: %+v", err)
	}
}

// The secret is printed once, with a warning; -q prints it alone. Forgejo refuses labels, before any request
func TestRegisterRunner(t *testing.T) {
	defaults(t)
	f := server(t)
	if err := RegisterRunner("web"); err == nil || err.Title != "Missing name" {
		t.Fatalf("no name: %+v", err)
	}

	testutil.Set(t, &RegisterName, "build-3")
	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { err = RegisterRunner("web") }))
	for _, want := range []string{"Registered runner build-3 (ID 6) for acme/web", "UUID:  u-6", "Token: s3cr3t", "shown only this once"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("%v, missing %q in:\n%s", err, want, out)
		}
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = RegisterRunner("web") }); err != nil || out != "s3cr3t\n" {
		t.Fatalf("-q: %v, %q", err, out)
	}

	f.calls = nil
	testutil.Set(t, &RegisterLabels, []string{"linux"})
	if err = RegisterRunner("web"); err == nil || err.Title != "Unsupported operation" || len(f.calls) != 0 {
		t.Fatalf("labels: %+v, calls %v", err, f.calls)
	}
}

func TestRunnerToken(t *testing.T) {
	defaults(t)
	server(t)
	testutil.Set(t, &InstanceScope, true)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = RunnerToken("") })
	// Forgejo's token does not expire: no expiry line
	if err != nil || !strings.Contains(out, "the whole instance") || !strings.Contains(out, "reg-t0ken") || strings.Contains(out, "expires") {
		t.Fatalf("%v, output:\n%s", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = RunnerToken("") }); out != "reg-t0ken\n" {
		t.Fatalf("-q: %q", out)
	}
}

// Every runner is checked before asking; a missing one stops everything, before any deletion
func TestDeleteRunners(t *testing.T) {
	defaults(t)
	f := server(t)
	testutil.Set(t, &AssumeYes, true)
	if err := DeleteRunners("web", []string{"3", "9"}); err == nil || err.Code != http.StatusNotFound ||
		slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasPrefix(c, "DELETE") }) {
		t.Fatalf("missing runner: %+v, calls %v", err, f.calls)
	}

	f.calls = nil
	out := testutil.Run(t, func() {
		if err := DeleteRunners("web", []string{"3", "4"}); err != nil {
			t.Error(err.Error())
		}
	})
	if !slices.Contains(f.calls, "DELETE repos/acme/web/actions/runners/3") || !slices.Contains(f.calls, "DELETE repos/acme/web/actions/runners/4") ||
		!strings.Contains(out, "Deleted runner 4 of acme/web") {
		t.Fatalf("calls %v\n%s", f.calls, out)
	}

	// Without --yes, a no keeps them
	testutil.Set(t, &AssumeYes, false)
	f.calls = nil
	out = testutil.Run(t, func() { _ = DeleteRunners("web", []string{"3"}) }, "n")
	if slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasPrefix(c, "DELETE") }) || !strings.Contains(out, "Aborted") {
		t.Fatalf("calls %v\n%s", f.calls, out)
	}
}
