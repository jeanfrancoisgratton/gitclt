// gitclt
// Tests the connection -> gitoperationslib mapping against an in-process HTTP server

package forge

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Points HOME to a temporary directory, and writes a one-connection registry in it
func writeRegistry(t *testing.T, connection string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dir := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "gitclt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	reg := `{"auths":{"test":` + connection + `},"default_connection":"test"}`
	if err := os.WriteFile(filepath.Join(dir, "connections.json"), []byte(reg), 0600); err != nil {
		t.Fatal(err)
	}
}

func listAll(t *testing.T, c *Client) []gol.Repo {
	t.Helper()
	var repos []gol.Repo
	fetch := func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return c.Repos().List(ctx, "", o)
	}
	for r, err := range gol.All(context.Background(), gol.ListOptions{PerPage: 2}, fetch) {
		if err != nil {
			t.Fatal(err)
		}
		repos = append(repos, r)
	}
	return repos
}

func TestConnect_ForgejoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token abc" {
			t.Errorf("%s: Authorization = %q", r.URL.Path, got)
		}
		if got := r.Header.Get("User-Agent"); got != "gitclt" {
			t.Errorf("%s: User-Agent = %q", r.URL.Path, got)
		}
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"15.0.2"}`)
		case "/api/v1/user/repos":
			fmt.Fprint(w, `[{"full_name":"me/one","private":true},{"full_name":"me/two"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`","token":"abc","servertype":"forgejo"}`)

	c, err := Connect(context.Background(), "")
	if err != nil {
		t.Fatal(err.Error())
	}
	if c.Backend() != gol.BackendForgejo || c.Connection.Name != "test" {
		t.Fatalf("backend %s, connection %s", c.Backend(), c.Connection.Name)
	}
	repos := listAll(t, c)
	if len(repos) != 2 || repos[0].Visibility != gol.VisibilityPrivate {
		t.Fatalf("got %+v", repos)
	}
}

func TestConnect_ForgejoBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "me" || p != "pw" {
			t.Errorf("%s: no basic auth", r.URL.Path)
		}
		fmt.Fprint(w, `{"version":"15.0.2"}`)
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`","username":"me","password":"pw","servertype":"forgejo"}`)

	if _, err := Connect(context.Background(), ""); err != nil {
		t.Fatal(err.Error())
	}
}

func TestConnect_ForgejoTooOld(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"version":"1.0.0"}`)
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`","token":"abc","servertype":"forgejo"}`)

	if _, err := Connect(context.Background(), ""); err == nil {
		t.Fatal("expected an error on an unsupported Forgejo version")
	}
}

// Any host other than github.com is treated as GitHub Enterprise Server: the API lives under /api/v3/
func TestConnect_GithubEnterprise(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer abc" {
			t.Errorf("%s: Authorization = %q", r.URL.Path, got)
		}
		if r.URL.Path != "/api/v3/user/repos" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
			return
		}
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"full_name":"me/three"}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/user/repos?page=2&per_page=2>; rel="next"`, srv.URL))
		fmt.Fprint(w, `[{"full_name":"me/one"},{"full_name":"me/two"}]`)
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`","token":"abc","servertype":"github"}`)

	c, err := Connect(context.Background(), "")
	if err != nil {
		t.Fatal(err.Error())
	}
	if repos := listAll(t, c); len(repos) != 3 || repos[2].FullName != "me/three" {
		t.Fatalf("got %+v", repos)
	}

	_, gerr := c.Repos().Get(context.Background(), "me", "nope")
	if e := Error("Unable to fetch the repository", gerr); e == nil || e.Code != http.StatusNotFound {
		t.Fatalf("got %+v", e)
	}
}

// Any host other than bitbucket.org is taken as the API root itself
func TestConnect_BitbucketBasicAuth(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "me@example.com" || p != "pw" {
			t.Errorf("%s: basic auth = %q, %q, %v", r.URL.Path, u, p, ok)
		}
		switch r.URL.Path {
		case "/2.0/user/workspaces":
			fmt.Fprint(w, `{"values":[{"workspace":{"slug":"acme"}}]}`)
		case "/2.0/repositories/acme":
			if r.URL.Query().Get("page") == "2" {
				fmt.Fprint(w, `{"values":[{"full_name":"acme/three","workspace":{"slug":"acme"}}]}`)
				return
			}
			fmt.Fprintf(w, `{"values":[{"full_name":"acme/one","is_private":true},{"full_name":"acme/two"}],"next":"%s/2.0/repositories/acme?page=2&pagelen=2"}`, srv.URL)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"type":"error","error":{"message":"Not Found"}}`)
		}
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`/2.0","username":"me@example.com","password":"pw","servertype":"bitbucket"}`)

	c, err := Connect(context.Background(), "")
	if err != nil {
		t.Fatal(err.Error())
	}
	if c.Backend() != gol.BackendBitbucket {
		t.Fatalf("backend = %s", c.Backend())
	}
	repos := listAll(t, c)
	if len(repos) != 3 || repos[0].Owner != "acme" || repos[0].Visibility != gol.VisibilityPrivate || repos[2].FullName != "acme/three" {
		t.Fatalf("got %+v", repos)
	}

	_, gerr := c.Repos().Get(context.Background(), "acme", "nope")
	if e := Error("Unable to fetch the repository", gerr); e == nil || e.Code != http.StatusNotFound {
		t.Fatalf("got %+v", e)
	}
}

// Access tokens go as bearer tokens; Atlassian user API tokens (ATATT...) go as basic auth with the account e-mail
func TestConnect_BitbucketTokens(t *testing.T) {
	for _, tc := range []struct{ conn, want string }{
		{`"token":"ATCTTaccess"`, "Bearer ATCTTaccess"},
		{`"token":"ATATTuser","username":"me@example.com","password":"ignored"`, "Basic " + basic("me@example.com", "ATATTuser")},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != tc.want {
				t.Errorf("%s: Authorization = %q, want %q", tc.conn, got, tc.want)
			}
			fmt.Fprint(w, `{"values":[]}`)
		}))
		writeRegistry(t, `{"url":"`+srv.URL+`/2.0",`+tc.conn+`,"servertype":"bitbucket"}`)
		c, err := Connect(context.Background(), "")
		if err != nil {
			t.Fatal(err.Error())
		}
		listAll(t, c)
		srv.Close()
	}

	// An API token without the e-mail cannot work: refused upfront
	writeRegistry(t, `{"url":"https://bitbucket.org","token":"ATATTuser","servertype":"bitbucket"}`)
	if _, err := Connect(context.Background(), ""); err == nil {
		t.Fatal("expected an error on an API token without username")
	}
}

func basic(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

func TestIsBitbucketDotOrg(t *testing.T) {
	for u, want := range map[string]bool{"https://bitbucket.org": true, "https://BitBucket.org/acme": true,
		"https://api.bitbucket.org/2.0": true, "https://bitbucket.example.com": false} {
		if got := isBitbucketDotOrg(u); got != want {
			t.Errorf("%s: got %v", u, got)
		}
	}
}

func TestConnect_UnknownConnection(t *testing.T) {
	writeRegistry(t, `{"url":"https://example.com","token":"abc","servertype":"forgejo"}`)
	if _, err := Connect(context.Background(), "nosuch"); err == nil {
		t.Fatal("expected an error on an unknown connection")
	}
}

func TestIsGithubDotCom(t *testing.T) {
	for u, want := range map[string]bool{"https://github.com": true, "https://API.github.com/": true,
		"https://www.github.com/me": true, "https://github.example.com": false, "::bad::": false} {
		if got := isGithubDotCom(u); got != want {
			t.Errorf("%s: got %v", u, got)
		}
	}
}

func TestIsAtlassianAPIToken(t *testing.T) {
	if !isAtlassianAPIToken("ATATTxyz") || isAtlassianAPIToken("ATCTTxyz") || isAtlassianAPIToken("") {
		t.Fatal("wrong token classification")
	}
}

func TestError(t *testing.T) {
	if Error("title", nil) != nil {
		t.Fatal("a nil error must stay nil (a non-nil *CustomError holding nil would look like a failure)")
	}
	if e := Error("title", fmt.Errorf("boom")); e == nil || e.Title != "title" || e.Message != "boom" || e.Code != 0 {
		t.Fatalf("got %+v", e)
	}
}

// Account administration is Forgejo-only: GitHub and Bitbucket clients refuse it
func TestAdmin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"version":"15.0.2"}`)
	}))
	defer srv.Close()
	writeRegistry(t, `{"url":"`+srv.URL+`","token":"abc","servertype":"forgejo"}`)
	c, err := Connect(context.Background(), "")
	if err != nil {
		t.Fatal(err.Error())
	}
	if adm, aerr := c.Admin(); aerr != nil || adm == nil {
		t.Fatalf("forgejo: got %v, %v", adm, aerr)
	}

	writeRegistry(t, `{"url":"https://github.com","token":"abc","servertype":"github"}`)
	if c, err = Connect(context.Background(), ""); err != nil {
		t.Fatal(err.Error())
	}
	if _, aerr := c.Admin(); aerr == nil || aerr.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", aerr)
	}
}
