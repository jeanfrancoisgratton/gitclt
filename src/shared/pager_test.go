// gitclt
// Tests when an output needs the pager, and that it is printed as is off a terminal

package shared

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestFits(t *testing.T) {
	red := "\x1b[31m"
	reset := "\x1b[0m"
	tests := []struct {
		name          string
		lines         []string
		width, height int
		want          bool
	}{
		{"fewer lines than rows", []string{"a", "b"}, 80, 3, true},
		// the last row is the prompt's
		{"as many lines as rows", []string{"a", "b", "c"}, 80, 3, false},
		// 10 cells on a 4-cell terminal take 3 rows
		{"wrapped line", []string{"abcdefghij"}, 4, 4, true},
		{"wrapped line overflows", []string{"abcdefghij", "k"}, 4, 4, false},
		// 4 visible cells, many more bytes: one row
		{"colors take no room", []string{red + "abcd" + reset, "e"}, 4, 3, true},
		{"empty line takes a row", []string{"", "", ""}, 80, 3, false},
		{"unknown size", []string{"a", "b", "c"}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fits(tt.lines, tt.width, tt.height); got != tt.want {
				t.Errorf("fits(%q, %d, %d) = %v, want %v", tt.lines, tt.width, tt.height, got, tt.want)
			}
		})
	}
}

// The tests' standard output is a pipe: the output is printed once, with one final newline
func TestPage_NotATerminal(t *testing.T) {
	long := strings.Repeat("line\n", 500)
	out := testutil.Run(t, func() { Page(long, TableHeader) })
	if out != long {
		t.Fatalf("got %d bytes, want the %d of the output", len(out), len(long))
	}
	if out = testutil.Run(t, func() { Page("no newline", 0) }); out != "no newline\n" {
		t.Fatalf("output = %q", out)
	}
}
