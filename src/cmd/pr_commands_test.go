// gitclt
// Tests the pull request commands' wiring: their arguments, and the flag checks made before any connection

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestPRCommands(t *testing.T) {
	// Everything here fails before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"pr", "ls"}, "accepts 1 arg(s)"},
		{[]string{"pr", "get", "web"}, "accepts 2 arg(s)"},
		{[]string{"pr", "create", "web"}, "accepts between 2 and 3 arg(s)"},
		{[]string{"pr", "create", "-b", "x", "-F", "f.md", "web", "feature"}, "[body body-file] were all set"},
		{[]string{"pr", "create", "-F", "/nonexistent/body.md", "web", "feature"}, "Unable to read /nonexistent/body.md"},
		{[]string{"pr", "edit", "web", "1"}, "Nothing to edit"},
		{[]string{"pr", "edit", "web", "one", "--title", "x"}, "one is not a pull request number"},
		{[]string{"pr", "merge", "-m", "octopus", "web", "1"}, "Invalid merge method"},
		{[]string{"pr", "merge", "--auto", "--cancel-auto", "web", "1"}, "[auto cancel-auto] were all set"},
		{[]string{"pr", "close", "web"}, "requires at least 2 arg(s)"},
		{[]string{"pr", "reopen", "web", "0"}, "0 is not a pull request number"},
		{[]string{"pr", "review", "web", "1"}, "pass exactly one of --approve, --request-changes, --comment"},
		{[]string{"pr", "review", "-a", "-x", "web", "1"}, "were all set"},
		{[]string{"pr", "review", "-n", "web", "1"}, "need a text (-m)"},
		{[]string{"pr", "reviewers", "web", "1"}, "requires at least 3 arg(s)"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
