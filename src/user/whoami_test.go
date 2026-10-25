// gitclt
// Tests whoami against a fake Forgejo, where some parts of the profile cannot be read

package user

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Only the profile, settings and e-mails are served: everything else is a 404, reported as missing
func whoamiServer(t *testing.T) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user":          testutil.Reply(jfgUser),
		"GET /api/v1/user/settings": testutil.Reply(`{"theme":"forgejo-dark"}`),
		"GET /api/v1/user/emails":   testutil.Reply(`[{"email":"jfg@example.com","primary":true}]`),
	})
}

func TestWhoAmI(t *testing.T) {
	defaults(t)
	whoamiServer(t)

	cu, err := WhoAmI(false)
	if err != nil || cu.Login != "jfg" || len(cu.Missing) == 0 {
		t.Fatalf("got %+v, %v", cu, err)
	}

	out := testutil.Run(t, func() { _, err = WhoAmI(true) })
	// Without -x, the parts that could not be read are not shown, so they are not reported either
	if err != nil || !strings.Contains(out, "J.F. Gratton") || strings.Contains(out, "forgejo-dark") || strings.Contains(out, "could not be fetched") {
		t.Fatalf("basic: %v\n%s", err, out)
	}

	testutil.Set(t, &ExtendedInfo, true)
	out = testutil.Run(t, func() { _, err = WhoAmI(true) })
	for _, want := range []string{"forgejo-dark", "jfg@example.com", "Some information could not be fetched", "SSH keys (HTTP 404)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = WhoAmI(true) }); err != nil || !strings.Contains(out, `"login"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestWhoAmI_Errors(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	testutil.WriteRegistry(t, "")
	if _, err = WhoAmI(false); err == nil {
		t.Fatal("expected an error without a connection")
	}
	// A bad token: the profile itself is refused
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user": testutil.Status(http.StatusUnauthorized)})
	if _, err = WhoAmI(false); err == nil || err.Code != http.StatusUnauthorized {
		t.Fatalf("got %+v", err)
	}
}
