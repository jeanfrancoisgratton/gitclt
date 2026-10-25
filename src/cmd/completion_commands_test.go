// gitclt
// Tests the shell completion commands

package cmd

import (
	"strings"
	"testing"
)

func TestCompletion(t *testing.T) {
	for shell, want := range map[string]string{"bash": "bash completion V2 for gitclt", "zsh": "#compdef gitclt"} {
		if out, stderr := execute(t, []string{"completion", shell}); !strings.Contains(out, want) {
			t.Errorf("%s: stderr %q, output starts with %.80q", shell, stderr, out)
		}
	}
}
