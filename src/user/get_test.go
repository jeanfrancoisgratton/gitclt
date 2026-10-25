// gitclt
// Tests user get against a fake Forgejo

package user

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestUserGet(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/users/jfg": testutil.Reply(jfgUser)})

	u, err := UserGet("jfg", false)
	if err != nil || u.Login != "jfg" || !u.IsAdmin {
		t.Fatalf("got %+v, %v", u, err)
	}

	out := testutil.Run(t, func() { _, err = UserGet("jfg", true) })
	if err != nil || !strings.Contains(out, "J.F. Gratton") || !strings.Contains(out, "Montreal") || strings.Contains(out, "Pronouns:") {
		t.Fatalf("table: %v\n%s", err, out)
	}

	testutil.Set(t, &ExtendedInfo, true)
	if out = testutil.Run(t, func() { _, err = UserGet("jfg", true) }); !strings.Contains(out, "they/them") {
		t.Fatalf("extended: %v\n%s", err, out)
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = UserGet("jfg", true) }); err != nil || !strings.Contains(out, `"login"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestUserGet_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	testutil.WriteRegistry(t, "")
	if _, err = UserGet("jfg", false); err == nil {
		t.Fatal("expected an error without a connection")
	}
	testutil.Forgejo(t, nil)
	if _, err = UserGet("nobody", false); err == nil || err.Code != http.StatusNotFound || err.Title != "Unable to fetch user nobody" {
		t.Fatalf("got %+v", err)
	}
}
