// gitclt
// Tests repo create: what is sent to the forge, depending on the flags

package repositories

import (
	"encoding/json/v2"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

type createBody struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	AutoInit      bool   `json:"auto_init"`
	DefaultBranch string `json:"default_branch"`
	Template      bool   `json:"template"`
	Gitignores    string `json:"gitignores"`
}

// A fake Forgejo where "me" is logged in and acme is an organization; the create request lands in *got
// Forgejo creates repositories with Actions on, and its create endpoint takes no features
func createServer(t *testing.T, got *createBody, status int) {
	t.Helper()
	createServerWith(t, got, status, nil)
}

// createServerWith adds routes to createServer's, e.g. the PATCH that sets features afterwards
func createServerWith(t *testing.T, got *createBody, status int, more map[string]http.HandlerFunc) {
	t.Helper()
	routes := map[string]http.HandlerFunc{
		"GET /api/v1/user":      testutil.Reply(`{"id":1,"login":"me"}`),
		"GET /api/v1/orgs/acme": testutil.Reply(`{"id":2,"username":"acme"}`),
		"POST /api/v1/orgs/acme/repos": func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(b, got); err != nil {
				t.Errorf("body %s: %v", b, err)
			}
			if status != http.StatusCreated {
				testutil.Status(status)(w, r)
				return
			}
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"id":5,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_actions":true}`)
		},
	}
	maps.Copy(routes, more)
	testutil.Forgejo(t, routes)
}

func createDefaults(t *testing.T) {
	t.Helper()
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Set(t, &TemplateRepo, false)
	testutil.Set(t, &InitializeRepo, false)
	testutil.Set(t, &RepoDefaultBranch, "")
	testutil.Set(t, &RepoDescription, "")
	testutil.Set(t, &PublicRepo, true)
	testutil.Set(t, &CreateActions, true)
	testutil.Set(t, &shared.QuietOutput, false)
}

func TestCreateRepo_Defaults(t *testing.T) {
	createDefaults(t)
	var got createBody
	createServer(t, &got, http.StatusCreated)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateRepo("web", passed()) })
	if err != nil {
		t.Fatal(err.Error())
	}
	want := createBody{Name: "web", DefaultBranch: "develop"}
	if got != want {
		t.Fatalf("sent %+v, want %+v", got, want)
	}
	// Not initialized: the user is told to set it up locally
	if !strings.Contains(out, "web has been created") || !strings.Contains(out, "The repo is not initialized") {
		t.Fatalf("output = %q", out)
	}
}

// --initialize adds the Go gitignore template; --public=false makes it private
func TestCreateRepo_AllFlags(t *testing.T) {
	createDefaults(t)
	testutil.Set(t, &InitializeRepo, true)
	testutil.Set(t, &TemplateRepo, true)
	testutil.Set(t, &PublicRepo, false)
	testutil.Set(t, &RepoDescription, "The site")
	testutil.Set(t, &RepoDefaultBranch, "main")
	testutil.Set(t, &shared.QuietOutput, true)
	var got createBody
	createServer(t, &got, http.StatusCreated)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateRepo("web", passed()) })
	if err != nil {
		t.Fatal(err.Error())
	}
	want := createBody{Name: "web", Description: "The site", Private: true, AutoInit: true, DefaultBranch: "main", Template: true, Gitignores: "Go"}
	if got != want {
		t.Fatalf("sent %+v, want %+v", got, want)
	}
	if out != "" {
		t.Fatalf("-q should print nothing, got %q", out)
	}
}

// develop is only asked for where the forge can set it; an explicit branch is always passed on, for the forge to refuse
func TestCreateBranch(t *testing.T) {
	for _, tc := range []struct {
		backend    gol.Backend
		flag       string
		initialize bool
		want       string
	}{
		{gol.BackendForgejo, "", false, "develop"},
		{gol.BackendForgejo, "", true, "develop"},
		{gol.BackendForgejo, "main", false, "main"},
		{gol.BackendGitHub, "", false, ""},
		{gol.BackendGitHub, "", true, ""},
		{gol.BackendGitHub, "main", false, "main"},
		{gol.BackendBitbucket, "", false, ""},
		{gol.BackendBitbucket, "", true, "develop"},
		{gol.BackendBitbucket, "main", false, "main"},
	} {
		createDefaults(t)
		testutil.Set(t, &RepoDefaultBranch, tc.flag)
		testutil.Set(t, &InitializeRepo, tc.initialize)
		if got := createBranch(tc.backend); got != tc.want {
			t.Errorf("%+v: got %q", tc, got)
		}
	}
}

func TestCreateRepo_Errors(t *testing.T) {
	createDefaults(t)
	var got createBody
	createServer(t, &got, http.StatusConflict)
	if err := CreateRepo("web", passed()); err == nil || err.Title != "Could not create repository" {
		t.Fatalf("got %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if err := CreateRepo("web", passed()); err == nil {
		t.Fatal("expected an error without a connection")
	}
}

// --actions=false: Forgejo's create endpoint takes no features, so Actions is turned off afterwards
func TestCreateRepo_ActionsOff(t *testing.T) {
	createDefaults(t)
	testutil.Set(t, &CreateActions, false)
	var got createBody
	var bodies []map[string]any
	createServerWith(t, &got, http.StatusCreated, map[string]http.HandlerFunc{
		"PATCH /api/v1/repos/acme/web": testutil.Record(t, &bodies, http.StatusOK,
			`{"id":5,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_actions":false}`),
	})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateRepo("web", passed("actions")) })
	if err != nil || len(bodies) != 1 || bodies[0]["has_actions"] != false {
		t.Fatalf("bodies %v, %v", bodies, err)
	}
	if !strings.Contains(out, "web has been created") {
		t.Fatalf("output = %q", out)
	}
}

// The server keeps Actions on (or off, for the whole instance): the repository exists, so it is a warning, not an error
func TestCreateRepo_FeatureNotApplied(t *testing.T) {
	createDefaults(t)
	testutil.Set(t, &CreateActions, false)
	testutil.Set(t, &shared.QuietOutput, true)
	var got createBody
	createServerWith(t, &got, http.StatusCreated, map[string]http.HandlerFunc{
		"PATCH /api/v1/repos/acme/web": testutil.Reply(`{"id":5,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","has_actions":true}`),
	})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateRepo("web", passed("actions")) })
	if err != nil {
		t.Fatal(err.Error())
	}
	// Shown even with -q
	if !strings.Contains(out, "web was created, but") || !strings.Contains(out, "actions off") {
		t.Fatalf("output = %q", out)
	}
}

// Actions is always asked for, except on Bitbucket when --actions was not passed: it has no features
func TestCreateFeatures(t *testing.T) {
	for _, tc := range []struct {
		backend gol.Backend
		passed  bool
		value   bool
		want    *bool
	}{
		{gol.BackendForgejo, false, true, gol.Ptr(true)},
		{gol.BackendForgejo, true, false, gol.Ptr(false)},
		{gol.BackendGitHub, false, true, gol.Ptr(true)},
		{gol.BackendBitbucket, false, true, nil},
		{gol.BackendBitbucket, true, true, gol.Ptr(true)},
	} {
		createDefaults(t)
		testutil.Set(t, &CreateActions, tc.value)
		changed := passed()
		if tc.passed {
			changed = passed("actions")
		}
		got := createFeatures(tc.backend, changed).Actions
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%+v: got %v", tc, got)
		}
	}
}
