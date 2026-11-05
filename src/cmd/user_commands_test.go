// gitclt
// Tests the user commands' wiring: what reaches the server depends on which flags were actually passed

package cmd

import (
	"encoding/json/v2"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

const jfg = `{"id":4,"login":"jfg","full_name":"J.F. Gratton","visibility":"public"}`

// Records the JSON keys sent with each request
func keysRecorder(t *testing.T, sent *[][]string, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		m := map[string]any{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Errorf("body %s: %v", b, err)
		}
		*sent = append(*sent, slices.Sorted(maps.Keys(m)))
		w.WriteHeader(status)
		_, _ = io.WriteString(w, jfg)
	}
}

func TestUserCommands(t *testing.T) {
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user":               testutil.Reply(jfg),
		"GET /api/v1/users/jfg":          testutil.Reply(jfg),
		"GET /api/v1/admin/users":        testutil.Reply(`[` + jfg + `]`),
		"DELETE /api/v1/admin/users/jfg": testutil.Status(http.StatusNoContent),
	})
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"whoami"}, nil, "J.F. Gratton"},
		{[]string{"user", "whoami", "--json"}, nil, `"login": "jfg"`},
		{[]string{"user", "get", "jfg"}, nil, "J.F. Gratton"},
		{[]string{"user", "get", "nobody"}, nil, "Unable to fetch user nobody"},
		{[]string{"user", "ls"}, nil, "J.F. Gratton"},
		{[]string{"user", "rm", "jfg"}, []string{"n"}, "Aborted"},
		{[]string{"user", "rm", "-y", "jfg"}, nil, "Deleted user jfg"},
		{[]string{"user", "edit", "jfg"}, nil, "Nothing to edit"},
		{[]string{"user", "wipe", "-n", "bob"}, nil, "Admin rights required"},
	} {
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

func TestUserSelfServiceCommands(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "id.pub")
	if err := os.WriteFile(keyFile, []byte("ssh-ed25519 AAAA jfg@laptop\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const key = `{"id":5,"title":"laptop","key":"ssh-ed25519 AAAA jfg@laptop","fingerprint":"SHA256:abc"}`
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/emails":    testutil.Reply(`[{"email":"jfg@example.com","primary":true},{"email":"old@example.com"}]`),
		"POST /api/v1/user/emails":   testutil.Reply(`[{"email":"new@example.com"}]`),
		"DELETE /api/v1/user/emails": testutil.Status(http.StatusNoContent),
		"GET /api/v1/user/keys":      testutil.Reply(`[` + key + `]`),
		"POST /api/v1/user/keys":     testutil.Reply(key),
		"DELETE /api/v1/user/keys/5": testutil.Status(http.StatusNoContent),
		"DELETE /api/v1/user/avatar": testutil.Status(http.StatusNoContent),
	})
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"user", "email", "ls"}, nil, "old@example.com"},
		{[]string{"user", "emails", "add", "new@example.com"}, nil, "Added e-mail address new@example.com"},
		{[]string{"user", "email", "rm", "old@example.com"}, []string{"n"}, "Aborted"},
		{[]string{"user", "email", "rm", "-y", "old@example.com"}, nil, "Deleted e-mail address old@example.com"},
		{[]string{"user", "sshkey", "ls"}, nil, "SHA256:abc"},
		{[]string{"user", "ssh", "get", "laptop"}, nil, "SHA256:abc"},
		{[]string{"user", "sshkey", "get", "--id", "5", "-j"}, nil, `"fingerprint": "SHA256:abc"`},
		{[]string{"user", "sshkey", "add", "laptop", keyFile}, nil, "Added SSH key laptop"},
		{[]string{"user", "sshkey", "rm", "-y", "laptop"}, nil, "Deleted SSH key laptop"},
		{[]string{"user", "avatar", "rm", "-y"}, nil, "Avatar deleted"},
		{[]string{"user", "avatar", "set", keyFile}, nil, "Invalid avatar"},
	} {
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

func TestUserSelfService_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"user", "email", "add"}, "requires at least 1 arg(s)"},
		{[]string{"user", "email", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"user", "sshkey", "add", "laptop"}, "accepts 2 arg(s)"},
		{[]string{"user", "sshkey", "get"}, "accepts 1 arg(s)"},
		{[]string{"user", "sshkey", "rm", "--id", "5", "laptop"}, "either the key's title, or --id"},
		{[]string{"user", "sshkey", "rm", "--id", "0"}, "--id must be a positive number"},
		{[]string{"user", "avatar", "set"}, "accepts 1 arg(s)"},
		{[]string{"user", "avatar", "rm", "extra"}, "unknown command"},
	} {
		if _, stderr := execute(t, tc.args); !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: stderr = %q", tc.args, stderr)
		}
	}
}

// --admin=false must be sent (as false); flags not passed must not be sent at all
func TestUserEdit_ChangedFlags(t *testing.T) {
	var sent [][]string
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/admin/users/jfg": keysRecorder(t, &sent, http.StatusOK)})

	out, _ := execute(t, []string{"user", "edit", "--admin=false", "-l", "Montreal", "jfg"})
	if !strings.Contains(out, "Updated user jfg") || len(sent) != 1 || !slices.Equal(sent[0], []string{"admin", "location"}) {
		t.Fatalf("sent %v, output %q", sent, out)
	}
}

// Without -m, must_change_password is left to the server; with -m=false, it is sent
func TestUserCreate_MustChangePasswd(t *testing.T) {
	var sent [][]string
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/admin/users": keysRecorder(t, &sent, http.StatusCreated)})

	// -D: GetPassword needs a terminal otherwise
	execute(t, []string{"user", "create", "-D", "-e", "jfg@example.com", "jfg"}, "pw")
	execute(t, []string{"user", "create", "-D", "-e", "jfg@example.com", "-m=false", "jfg"}, "pw")
	if len(sent) != 2 || slices.Contains(sent[0], "must_change_password") || !slices.Contains(sent[1], "must_change_password") {
		t.Fatalf("sent %v", sent)
	}
}

func TestUserCommands_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"user", "create", "jfg"}, `required flag(s) "email" not set`},
		{[]string{"user", "create", "-e", "a@b", "-v", "secret", "jfg"}, "must be one of public, limited or private"},
		{[]string{"user", "edit", "-p", "file", "-P", "jfg"}, "none of the others can be"},
		{[]string{"user", "get"}, "accepts 1 arg(s)"},
		{[]string{"user", "ls", "extra"}, "unknown command"},
	} {
		if _, stderr := execute(t, tc.args); !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: stderr = %q", tc.args, stderr)
		}
	}
}
