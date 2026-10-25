// gitclt
// Tests repo create: what is sent to the forge, depending on the flags

package repositories

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
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
func createServer(t *testing.T, got *createBody, status int) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
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
			_, _ = io.WriteString(w, `{"id":5,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`)
		},
	})
}

func createDefaults(t *testing.T) {
	t.Helper()
	defaults(t)
	testutil.Set(t, &OwnerOrg, "acme")
	testutil.Set(t, &TemplateRepo, false)
	testutil.Set(t, &InitializeRepo, false)
	testutil.Set(t, &RepoDefaultBranch, "develop")
	testutil.Set(t, &RepoDescription, "")
	testutil.Set(t, &PublicRepo, true)
	testutil.Set(t, &shared.QuietOutput, false)
}

func TestCreateRepo_Defaults(t *testing.T) {
	createDefaults(t)
	var got createBody
	createServer(t, &got, http.StatusCreated)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateRepo("web") })
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
	out := testutil.Run(t, func() { err = CreateRepo("web") })
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

func TestCreateRepo_Errors(t *testing.T) {
	createDefaults(t)
	var got createBody
	createServer(t, &got, http.StatusConflict)
	if err := CreateRepo("web"); err == nil || err.Title != "Could not create repository" {
		t.Fatalf("got %+v", err)
	}

	testutil.WriteRegistry(t, "")
	if err := CreateRepo("web"); err == nil {
		t.Fatal("expected an error without a connection")
	}
}
