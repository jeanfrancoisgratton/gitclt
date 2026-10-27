// gitclt
// Tests the user commands' wiring: what reaches the server depends on which flags were actually passed

package cmd

import (
	"encoding/json/v2"
	"io"
	"maps"
	"net/http"
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
