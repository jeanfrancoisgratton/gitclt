// gitclt
// Tests the token commands against a fake Forgejo which, like the real one, only creates and deletes tokens
// with username and password

package tokens

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const tokenList = `[{"id":1,"name":"ci","scopes":["all"],"token_last_eight":"aaaaaaaa"},
	{"id":42,"name":"123","scopes":["read:user","write:repository"],"token_last_eight":"bbbbbbbb",
	"repositories":[{"full_name":"acme/one"},{"full_name":"acme/two"}]}]`

func defaults(t *testing.T) {
	t.Helper()
	testutil.Set(t, &JSONoutput, false)
	testutil.Set(t, &AssumeYes, false)
	testutil.Set(t, &TokenID, 0)
	testutil.Set(t, &shared.OTP, "")
	testutil.Set(t, &CreateTokenScopes, nil)
	testutil.Set(t, &CreateTokenRepos, nil)
	testutil.Set(t, &shared.QuietOutput, false)
	testutil.Set(t, &shared.ConnectionName, "")
	testutil.Set(t, &shared.DebugMode, true) // GetPassword needs a terminal otherwise
}

// What the fake server has seen
type seen struct {
	bodies  []map[string]any
	deleted []string
	otp     string
}

func passwordOnly(s *seen, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "acme" || p != "pw" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message":"auth method not allowed"}`)
			return
		}
		s.otp = r.Header.Get("X-FORGEJO-OTP")
		h(w, r)
	}
}

// Starts the fake Forgejo; auth is the credentials part of the connection (the connection's username is acme)
func tokenServer(t *testing.T, auth string) *seen {
	t.Helper()
	s := &seen{}
	created := `{"id":7,"name":"new","scopes":["read:user"],"sha1":"thesecret12345678","token_last_eight":"12345678"}`
	var srv *httptest.Server
	srv = testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user":               testutil.Reply(`{"id":1,"login":"acme"}`),
		"GET /api/v1/users/acme/tokens":  testutil.Reply(tokenList),
		"POST /api/v1/users/acme/tokens": passwordOnly(s, testutil.Record(t, &s.bodies, http.StatusCreated, created)),
		"DELETE /api/v1/users/acme/tokens/{id}": passwordOnly(s, func(w http.ResponseWriter, r *http.Request) {
			s.deleted = append(s.deleted, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	testutil.WriteRegistry(t, `{"auths":{"test":{"url":"`+srv.URL+`",`+auth+`,"servertype":"forgejo"}},"default_connection":"test"}`)
	return s
}

const (
	tokenAuth    = `"token":"xxxxaaaaaaaa","username":"acme"`
	bothAuth     = `"token":"xxxxaaaaaaaa","username":"acme","password":"pw"`
	passwordAuth = `"username":"acme","password":"pw"`
)

func TestListTokens(t *testing.T) {
	defaults(t)
	tokenServer(t, tokenAuth)

	list, err := ListTokens(false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(list) != 2 || list[1].ID != 42 || list[1].Repositories[1] != "acme/two" {
		t.Fatalf("got %+v", list)
	}

	out := testutil.Run(t, func() { _, err = ListTokens(true) })
	for _, want := range []string{"ci", "all", "read:user, write:repository", "acme/one, acme/two", "bbbbbbbb"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("%v: missing %q in:\n%s", err, want, out)
		}
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = ListTokens(true) }); err != nil || out != "ci\n123\n" {
		t.Fatalf("quiet: %v, %q", err, out)
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = ListTokens(true) }); err != nil || !strings.Contains(out, `"bbbbbbbb"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

// The forges without a token API are refused cleanly, before asking for anything
func TestTokens_NotSupported(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"gh":{"url":"https://github.com","token":"x","servertype":"github"}},"default_connection":"gh"}`)
	testutil.Set(t, &AssumeYes, true)

	_, err := ListTokens(false)
	if err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("list: got %+v", err)
	}
	if err = CreateToken("x"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("create: got %+v", err)
	}
	if err = DeleteToken("x"); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("delete: got %+v", err)
	}
}

func TestGetToken(t *testing.T) {
	defaults(t)
	tokenServer(t, tokenAuth)

	// A numeric name is a name: only --id looks at the IDs
	tok, err := GetToken("123", false)
	if err != nil || tok.ID != 42 {
		t.Fatalf("by name: %+v, %v", tok, err)
	}

	var out string
	out = testutil.Run(t, func() { _, err = GetToken("ci", true) })
	for _, want := range []string{"Name:", "ci", "Repositories:", "all", "aaaaaaaa"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("%v: missing %q in:\n%s", err, want, out)
		}
	}
	if strings.Contains(out, "Secret") {
		t.Errorf("a listed token has no secret to show:\n%s", out)
	}

	if _, err = GetToken("nope", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("unknown name: got %+v", err)
	}

	testutil.Set(t, &TokenID, 42)
	if tok, err = GetToken("", false); err != nil || tok.Name != "123" {
		t.Fatalf("by ID: %+v, %v", tok, err)
	}
	testutil.Set(t, &TokenID, 99)
	if _, err = GetToken("", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("unknown ID: got %+v", err)
	}
}

func TestCreateToken(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateTokenScopes, []string{"read:user"})
	testutil.Set(t, &CreateTokenRepos, []string{"acme/one"})
	s := tokenServer(t, bothAuth)

	// The stored password is used, without a prompt, even if the connection has a token
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateToken("new") })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Created token new", "thesecret12345678", "cannot be shown again"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	body := s.bodies[0]
	if body["name"] != "new" || fmt.Sprint(body["scopes"]) != "[read:user]" || fmt.Sprint(body["repositories"]) != "[map[name:one owner:acme]]" {
		t.Errorf("body = %v", body)
	}

	// The library never serialises the secret: we must
	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { err = CreateToken("new") }); err != nil ||
		!strings.Contains(out, `"secret"`) || !strings.Contains(out, "thesecret12345678") || !strings.Contains(out, `"lastEight"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}

	// -q: the secret alone
	testutil.Set(t, &JSONoutput, false)
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = CreateToken("new") }); err != nil || out != "thesecret12345678\n" {
		t.Fatalf("quiet: %v, %q", err, out)
	}
}

// What the connection does not store is asked for; and an empty answer stops everything
func TestCreateToken_Prompts(t *testing.T) {
	defaults(t)
	s := tokenServer(t, tokenAuth)
	var err *cerr.CustomError
	if out := testutil.Run(t, func() { err = CreateToken("new") }, "pw"); err != nil || !strings.Contains(out, "password of acme") {
		t.Fatalf("password: %v, %q", err, out)
	}

	s = tokenServer(t, `"token":"xxxxaaaaaaaa"`)
	if testutil.Run(t, func() { err = CreateToken("new") }, " acme ", "pw"); err != nil || len(s.bodies) != 1 {
		t.Fatalf("username and password: %v, %+v", err, s)
	}

	s = tokenServer(t, tokenAuth)
	if testutil.Run(t, func() { err = CreateToken("new") }, ""); err == nil || len(s.bodies) != 0 {
		t.Fatalf("empty password: %v, %+v", err, s)
	}
	s = tokenServer(t, `"token":"xxxxaaaaaaaa"`)
	if testutil.Run(t, func() { err = CreateToken("new") }, ""); err == nil || len(s.bodies) != 0 {
		t.Fatalf("empty username: %v, %+v", err, s)
	}

	// A wrong password
	tokenServer(t, tokenAuth)
	if testutil.Run(t, func() { err = CreateToken("new") }, "wrong"); err == nil || err.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %+v", err)
	}
}

// Without a terminal, there is no way to ask for the password: a clean error, not GetPassword's panic
func TestCreateToken_NoTerminal(t *testing.T) {
	defaults(t)
	testutil.Set(t, &shared.DebugMode, false)
	s := tokenServer(t, tokenAuth)
	var err *cerr.CustomError
	testutil.Run(t, func() { err = CreateToken("new") })
	if err == nil || !strings.Contains(err.Message, "gitclt conn edit test") || len(s.bodies) != 0 {
		t.Fatalf("got %+v, %+v", err, s)
	}
}

func TestCreateToken_OTP(t *testing.T) {
	defaults(t)
	testutil.Set(t, &shared.OTP, "123456")
	s := tokenServer(t, bothAuth)
	var err *cerr.CustomError
	testutil.Run(t, func() { err = CreateToken("new") })
	if err != nil || s.otp != "123456" {
		t.Fatalf("%v, otp = %q", err, s.otp)
	}
}

func TestDeleteToken_Confirmation(t *testing.T) {
	for _, answer := range []string{"", "n", "nope"} {
		defaults(t)
		s := tokenServer(t, bothAuth)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteToken("123") }, answer)
		if err != nil || len(s.deleted) != 0 || !strings.Contains(out, "Aborted") {
			t.Errorf("%q: deleted %v, %v, %q", answer, s.deleted, err, out)
		}
	}

	for _, answer := range []string{"y", " YES "} {
		defaults(t)
		s := tokenServer(t, bothAuth)
		var err *cerr.CustomError
		out := testutil.Run(t, func() { err = DeleteToken("123") }, answer)
		// Deleted by its ID: Forgejo would take the numeric name for one
		if err != nil || fmt.Sprint(s.deleted) != "[42]" || !strings.Contains(out, "Deleted token 123") {
			t.Errorf("%q: deleted %v, %v, %q", answer, s.deleted, err, out)
		}
		if strings.Contains(out, "no longer works") {
			t.Errorf("%q: another token than the connection's: %q", answer, out)
		}
	}
}

func TestDeleteToken_ByIDAndYes(t *testing.T) {
	defaults(t)
	testutil.Set(t, &TokenID, 42)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &shared.QuietOutput, true)
	s := tokenServer(t, passwordAuth)

	// --yes: no prompt (testutil.Run fails the test on an unanswered prompt); -q: no output
	if out := testutil.Run(t, func() { _ = DeleteToken("") }); fmt.Sprint(s.deleted) != "[42]" || out != "" {
		t.Fatalf("deleted %v, %q", s.deleted, out)
	}
}

// The connection's own token: the confirmation says so, and so does the result
func TestDeleteToken_ConnectionsOwnToken(t *testing.T) {
	defaults(t)
	s := tokenServer(t, bothAuth)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteToken("ci") }, "y")
	if err != nil || fmt.Sprint(s.deleted) != "[1]" {
		t.Fatalf("deleted %v, %v", s.deleted, err)
	}
	for _, want := range []string{"which connection test authenticates with", "Connection test no longer works", "gitclt conn edit test"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestDeleteToken_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	s := tokenServer(t, bothAuth)
	if err := DeleteToken("nope"); err == nil || err.Code != http.StatusNotFound || len(s.deleted) != 0 {
		t.Fatalf("unknown token: %+v, deleted %v", err, s.deleted)
	}

	testutil.WriteRegistry(t, "")
	if err := DeleteToken("ci"); err == nil || err.Title != "No connection specified" {
		t.Fatalf("no connection: got %+v", err)
	}
}
