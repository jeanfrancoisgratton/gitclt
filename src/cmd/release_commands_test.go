// gitclt
// Tests the release commands' wiring: arguments and flags reach the release commands

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
		{[]string{"release", "upload", "v1"}, "requires at least 2 arg(s)"},
		{[]string{"rel", "up", "--draft", "web", "v1", "a.rpm"}, "--draft and --prerelease only apply to a release created with --create"},
		{[]string{"release", "upload", "-o", "acme", "--create", "web", "v1", "/nonexistent/a.rpm"}, "Unable to read /nonexistent/a.rpm"},
		{[]string{"release", "ls", "a", "b"}, "accepts at most 1 arg(s)"},
		{[]string{"rel", "lsa"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"rel", "rm"}, "requires at least 1 arg(s)"},
		{[]string{"rel", "rma", "v1"}, "requires at least 2 arg(s)"},
		{[]string{"rel", "edit"}, "accepts between 1 and 2 arg(s)"},
		{[]string{"rel", "edit", "web", "v1"}, "Nothing to edit"},
		{[]string{"rel", "edit", "--notes", "x", "--notes-file", "f.md", "web", "v1"}, "[notes notes-file] were all set"},
		{[]string{"rel", "edit", "--notes-file", "/nonexistent/notes.md", "web", "v1"}, "Unable to read /nonexistent/notes.md"},
		{[]string{"rel", "pub"}, "requires at least 1 arg(s)"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
