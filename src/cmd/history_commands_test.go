// gitclt
// Tests the history commands' wiring: their arguments, and the flag checks made before any connection

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestHistoryCommands(t *testing.T) {
	// Everything here fails before any connection; the empty registry makes sure of it
	testutil.WriteRegistry(t, "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"history", "ls"}, "accepts 1 arg(s)"},
		{[]string{"log", "ls", "a", "b"}, "accepts 1 arg(s)"},
		{[]string{"history", "get", "web"}, "accepts 2 arg(s)"},
		{[]string{"commits", "show", "web", "a", "b"}, "accepts 2 arg(s)"},
		{[]string{"history", "ls", "-n", "-1", "web"}, "-1 is not a commit count"},
		{[]string{"history", "ls", "-n", "ten", "web"}, "invalid argument"},
		{[]string{"history", "ls", "--since", "yesterday", "web"}, "yesterday is neither YYYY-MM-DD nor an RFC 3339 timestamp"},
		{[]string{"history", "ls", "--until", "2026-13-01", "web"}, "2026-13-01 is neither"},
	} {
		out, stderr := execute(t, tc.args)
		if !strings.Contains(out+stderr, tc.want) {
			t.Errorf("%v: output = %q, stderr = %q", tc.args, out, stderr)
		}
	}
}
