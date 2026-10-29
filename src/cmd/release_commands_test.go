// gitclt
// Tests the release commands' wiring: arguments and flags reach release upload

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestReleaseCommands(t *testing.T) {
	// Everything here fails before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"release", "upload", "web", "v1"}, "requires at least 3 arg(s)"},
		{[]string{"rel", "up", "--draft", "web", "v1", "a.rpm"}, "--draft and --prerelease only apply to a release created with --create"},
		{[]string{"release", "upload", "-o", "acme", "--create", "web", "v1", "/nonexistent/a.rpm"}, "Unable to read /nonexistent/a.rpm"},
		{[]string{"release", "ls"}, "accepts 1 arg(s)"},
		{[]string{"rel", "lsa", "web"}, "accepts 2 arg(s)"},
		{[]string{"rel", "rm", "web"}, "requires at least 2 arg(s)"},
		{[]string{"rel", "rma", "web", "v1"}, "requires at least 3 arg(s)"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
