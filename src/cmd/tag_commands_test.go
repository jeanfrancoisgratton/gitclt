// gitclt
// Tests the tag commands' wiring: their arguments and flags

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestTagCommands(t *testing.T) {
	// Everything here fails before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"tag", "ls"}, "accepts 1 arg(s)"},
		// The local side is gone: so are its flags
		{[]string{"tag", "ls", "-l", "web"}, "unknown shorthand flag: 'l'"},
		{[]string{"tag", "rm", "--all", "web", "v1"}, "unknown flag: --all"},
		{[]string{"tag", "get", "web"}, "accepts 2 arg(s)"},
		{[]string{"tag", "create", "web"}, "accepts 2 arg(s)"},
		{[]string{"tag", "rm", "web"}, "requires at least 2 arg(s)"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
