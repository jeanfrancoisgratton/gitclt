// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/pager.go
// Original timestamp : 2026.11.11 16:00:00

package shared

import (
	"fmt"
	"strings"

	hfpager "github.com/jeanfrancoisgratton/helperFunctions/v5/pager"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/text"
)

// TableHeader is how many lines a table's header takes: the colored styles have no top border, and headers are on one
// line, so a table's banner is its first line
const TableHeader = 1

// Page shows a command's whole output through the pager when it does not fit on the terminal, its first banner lines
// kept on top of every page (a table's header, see TableHeader, after any line written above it). Output that fits, or
// that goes to a pipe or a file, is printed as is; so is the output when the pager cannot start
func Page(output string, banner int) {
	output = strings.TrimRight(output, "\n")
	lines := strings.Split(output, "\n")
	if hftx.Interactive() {
		width, height := hftx.GetTerminalSize()
		if !fits(lines, width, height) && hfpager.Page(lines, banner, 0) == nil {
			return
		}
	}
	fmt.Println(output)
}

// fits tells if lines show on a width x height terminal without scrolling, the last row left to the shell's prompt; the
// lines wider than the terminal take as many rows as it wraps them on, and colors take no room
// An unknown size (0) is taken as fitting: the pager would only guess it
func fits(lines []string, width, height int) bool {
	if width <= 0 || height <= 0 {
		return true
	}
	rows := 0
	for _, l := range lines {
		rows += max(1, (text.StringWidthWithoutEscSequences(l)+width-1)/width)
		if rows >= height {
			return false
		}
	}
	return true
}
