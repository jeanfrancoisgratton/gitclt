// gitclt
// Tests the conn commands' wiring: arguments, and which connection function each one reaches

package cmd

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

const connRegistry = `{"auths":{"x":{"url":"https://x","token":"t","servertype":"forgejo"}},"default_connection":"x"}`

func TestConnCommands(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		answers []string
		want    string
	}{
		{[]string{"conn", "ls"}, nil, "Number of connections: 1"},
		{[]string{"conn", "info"}, nil, "https://x"},
		{[]string{"conn", "info", "nosuch"}, nil, "Connection nosuch does not exist"},
		{[]string{"conn", "default"}, nil, "Default connection: x"},
		{[]string{"conn", "default", "nosuch"}, nil, "Connection nosuch does not exist"},
		{[]string{"conn", "add", "x"}, nil, "Connection x already exists"},
		{[]string{"conn", "edit", "nosuch"}, nil, "Connection nosuch does not exist"},
		// -D must reach the secret prompts: without it, GetPassword needs a terminal and panics
		{[]string{"conn", "edit", "-D", "x"}, []string{"", "", "", "", "", ""}, "Connection x updated"},
		{[]string{"conn", "rm", "nosuch"}, nil, "Connection nosuch does not exist"},
		{[]string{"conn", "rm", "x"}, nil, "Connection x removed"},
	} {
		testutil.WriteRegistry(t, connRegistry)
		if out, _ := execute(t, tc.args, tc.answers...); !strings.Contains(out, tc.want) {
			t.Errorf("%v: output = %q", tc.args, out)
		}
	}
}

func TestConnCommands_Arguments(t *testing.T) {
	testutil.WriteRegistry(t, connRegistry)
	for _, args := range [][]string{{"conn", "add"}, {"conn", "rm"}, {"conn", "edit", "a", "b"}, {"conn", "default", "a", "b"}} {
		if _, stderr := execute(t, args); !strings.Contains(stderr, "arg(s)") {
			t.Errorf("%v: stderr = %q", args, stderr)
		}
	}
}
