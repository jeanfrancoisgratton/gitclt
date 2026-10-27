// gitclt
// Tests the branch commands' wiring: flag exclusions, and the owner being required for the server's side

package cmd

import (
	"strings"
	"testing"
)

func TestBranchCommands(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		// -a, or -l with -r, means both sides: the server's needs -o
		{[]string{"branch", "ls", "-l", "-a", "web"}, "Missing repository"},
		{[]string{"branch", "ls", "-l", "-r", "web"}, "Missing repository"},
		{[]string{"branch", "ls", "web"}, "Missing repository"},
		{[]string{"branch", "ls", "-r", "-o", "acme"}, "Missing repository"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
