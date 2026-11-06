// gitclt
// Tests the secret, variable and collaborator commands' wiring: where each argument goes

package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestCICommands(t *testing.T) {
	routes := map[string]http.HandlerFunc{"GET /api/v1/repos/acme/web": testutil.Reply(`{"name":"web"}`)}
	for _, base := range []string{"/api/v1/repos/acme/web", "/api/v1/orgs/corp", "/api/v1/user"} {
		routes["GET "+base+"/actions/secrets"] = testutil.Reply(`[{"name":"TOKEN"},{"name":"OTHER"}]`)
		routes["PUT "+base+"/actions/secrets/{name}"] = testutil.Status(http.StatusNoContent)
		routes["DELETE "+base+"/actions/secrets/{name}"] = testutil.Status(http.StatusNoContent)
		routes["GET "+base+"/actions/variables"] = testutil.Reply(`[{"name":"REGION","data":"eu"},{"name":"OTHER","data":"x"}]`)
		routes["GET "+base+"/actions/variables/REGION"] = testutil.Reply(`{"name":"REGION","data":"eu"}`)
		routes["PUT "+base+"/actions/variables/{name}"] = testutil.Status(http.StatusNoContent)
		routes["DELETE "+base+"/actions/variables/{name}"] = testutil.Status(http.StatusNoContent)
	}
	srv := testutil.Forgejo(t, routes)
	file := filepath.Join(t.TempDir(), "value")
	if err := os.WriteFile(file, []byte("v\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[remote \"origin\"]\n\turl = "+srv.URL+"/acme/web.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"secret", "ls", "web"}, "TOKEN"},
		{[]string{"secrets", "set", "-f", file, "web", "TOKEN"}, "Set secret TOKEN on acme/web"},
		{[]string{"secret", "rm", "-y", "web", "TOKEN", "OTHER"}, "Deleted secret OTHER"},
		{[]string{"secret", "ls", "--org", "corp", "-q"}, "TOKEN"},
		{[]string{"secret", "set", "--org", "corp", "-f", file, "TOKEN"}, "Set secret TOKEN on corp"},
		{[]string{"secret", "rm", "--org", "corp", "-y", "TOKEN", "OTHER"}, "Deleted secret OTHER"},
		{[]string{"secret", "set", "--user", "-f", file, "TOKEN"}, "Set secret TOKEN on your account"},
		{[]string{"variable", "ls", "web"}, "REGION"},
		{[]string{"var", "get", "web", "REGION"}, "eu"},
		{[]string{"vars", "set", "web", "REGION", "us"}, "Set variable REGION on acme/web"},
		{[]string{"var", "set", "-f", file, "web", "REGION"}, "Set variable REGION on acme/web"},
		{[]string{"var", "set", "--org", "corp", "REGION", "us"}, "Set variable REGION on corp"},
		{[]string{"var", "get", "--user", "REGION"}, "eu"},
		{[]string{"var", "rm", "-y", "web", "REGION"}, "Deleted variable REGION"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}

	// From a clone
	t.Chdir(root)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"secret", "ls"}, "TOKEN"},
		{[]string{"secret", "set", "-f", file, "TOKEN"}, "Set secret TOKEN on acme/web"},
		{[]string{"secret", "rm", "-y", "TOKEN", "OTHER"}, "pass . for the current one"},
		{[]string{"secret", "rm", "-y", ".", "TOKEN", "OTHER"}, "Deleted secret OTHER"},
		{[]string{"var", "set", "REGION", "us"}, "Set variable REGION on acme/web"},
		{[]string{"var", "set", "-f", file, "REGION"}, "Set variable REGION on acme/web"},
		{[]string{"var", "get", "REGION"}, "eu"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out, tc.want) {
			t.Errorf("clone: %v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestCollaboratorCommands(t *testing.T) {
	base := "/api/v1/repos/acme/web"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base:                                   testutil.Reply(`{"name":"web"}`),
		"GET " + base + "/collaborators":                testutil.Reply(`[{"login":"bob"}]`),
		"GET " + base + "/collaborators/bob":            testutil.Status(http.StatusNoContent),
		"GET " + base + "/collaborators/bob/permission": testutil.Reply(`{"permission":"write"}`),
		"PUT " + base + "/collaborators/bob":            testutil.Status(http.StatusNoContent),
		"DELETE " + base + "/collaborators/bob":         testutil.Status(http.StatusNoContent),
	})
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"collaborator", "ls", "web"}, nil, "bob"},
		{[]string{"collab", "get", "web", "bob"}, nil, "write"},
		{[]string{"collabs", "add", "-p", "read", "web", "bob"}, nil, "Gave bob read access to acme/web"},
		{[]string{"collab", "add", "web", "bob"}, nil, "Gave bob write access to acme/web"},
		{[]string{"collab", "rm", "web", "bob"}, []string{"n"}, "Aborted"},
		{[]string{"collab", "rm", "-y", "web", "bob"}, nil, "Removed bob from acme/web"},
		{[]string{"collab", "invitations", "web"}, nil, "only GitHub has invitations"},
	} {
		if out, stderr := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestCICommands_FlagRules(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"secret", "set"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"secret", "set", "--org", "corp", "a", "b"}, "accepts 1 arg(s)"},
		{[]string{"secret", "ls", "--org", "corp", "web"}, "accepts 0 arg(s)"},
		{[]string{"secret", "ls", "--user", "web"}, "accepts 0 arg(s)"},
		{[]string{"secret", "set", "-f", "x", "--stdin", "A"}, "none of the others can be"},
		{[]string{"secret", "ls", "--org", "corp", "--user"}, "none of the others can be"},
		{[]string{"secret", "ls", "--org", "corp", "--env", "prod"}, "none of the others can be"},
		{[]string{"secret", "ls", "--user", "-o", "acme"}, "none of the others can be"},
		{[]string{"secret", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"var", "set", "NAME"}, "accepts between 2 and 3 arg(s)"},
		{[]string{"var", "set", "-f", "x", "a", "b", "c"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"var", "get"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"collab", "add"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"collab", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"collab", "ls", "a", "b"}, "accepts at most 1 arg(s)"},
	} {
		if out, stderr := execute(t, tc.args); !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}

func TestCICommands_Help(t *testing.T) {
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"secret", "--help"}, []string{"write-only", "--org ORG", "From inside a clone"}},
		{[]string{"secret", "set", "--help"}, []string{"never an argument", "--from-file", "--stdin", "From inside a clone"}},
		{[]string{"var", "--help"}, []string{"not for credentials", "--env ENV"}},
		{[]string{"var", "set", "--help"}, []string{"--from-file", "From inside a clone"}},
		{[]string{"collab", "--help"}, []string{"gitclt team", "From inside a clone"}},
		{[]string{"collab", "add", "--help"}, []string{"invites a new collaborator", "--permission"}},
		{[]string{"collab", "rm", "--help"}, []string{"removing a stranger", "use . for the current clone's"}},
	} {
		out, stderr := execute(t, tc.args)
		for _, want := range tc.want {
			if !strings.Contains(out+stderr, want) {
				t.Errorf("%v: missing %q in:\n%s%s", tc.args, want, out, stderr)
			}
		}
	}
}
