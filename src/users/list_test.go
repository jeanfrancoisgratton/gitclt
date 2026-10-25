// gitclt
// Tests user ls against a fake Forgejo, including a second page

package users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestListUsers(t *testing.T) {
	defaults(t)
	var srv *httptest.Server
	srv = testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/admin/users": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "2" {
				fmt.Fprint(w, `[{"id":9,"login":"zed"}]`)
				return
			}
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/admin/users?page=2>; rel="next"`, srv.URL))
			fmt.Fprint(w, `[`+jfgUser+`,{"id":5,"login":"other","email":"other@example.com"}]`)
		},
	})

	users, err := ListUsers(false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(users) != 3 || users[0].Login != "jfg" || users[2].Login != "zed" {
		t.Fatalf("got %+v", users)
	}

	out := testutil.Run(t, func() { _, err = ListUsers(true) })
	for _, want := range []string{"LOGIN", "jfg", "other@example.com", "zed", "limited"} {
		if !strings.Contains(strings.ToUpper(out), strings.ToUpper(want)) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListUsers(true) }); err != nil || !strings.Contains(out, `"zed"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestListUsers_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	testutil.WriteRegistry(t, "")
	if _, err = ListUsers(false); err == nil {
		t.Fatal("expected an error without a connection")
	}
	// Not a Forgejo: refused before any request is made
	testutil.WriteRegistry(t, `{"auths":{"gh":{"url":"https://github.com","token":"x","servertype":"github"}},"default_connection":"gh"}`)
	if _, err = ListUsers(false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("github: got %+v", err)
	}
	// Not a site admin
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/admin/users": testutil.Status(http.StatusForbidden)})
	if _, err = ListUsers(false); err == nil || err.Code != http.StatusForbidden {
		t.Fatalf("got %+v", err)
	}
}
