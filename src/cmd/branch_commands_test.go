// gitclt
// Tests the branch commands' wiring: flag exclusions, and the repository being required for the server's side

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestBranchCommands(t *testing.T) {
	// The checks happen before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		// -a, or -l with -r, or neither, means both sides: the server's needs the repository
		{[]string{"branch", "ls", "-l", "-a"}, "Missing repository"},
		{[]string{"branch", "ls", "-l", "-r"}, "Missing repository"},
		{[]string{"branch", "ls"}, "Missing repository"},
		{[]string{"branch", "ls", "-r", "-o", "acme"}, "Missing repository"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
