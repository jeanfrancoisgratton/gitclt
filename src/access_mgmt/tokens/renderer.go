// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/renderer.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// A token without repositories is not limited to any of them
func repositories(tok *gol.AccessToken) string {
	if len(tok.Repositories) == 0 {
		return "all"
	}
	return strings.Join(tok.Repositories, ", ")
}

func renderTokenTable(list []gol.AccessToken) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Scopes", "Repositories", "Ends with"})
	for i := range list {
		tok := &list[i]
		t.AppendRow(table.Row{tok.ID, tok.Name, strings.Join(tok.Scopes, ", "), repositories(tok), tok.LastEight})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(58)}
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

// Shows a token; secret is empty, except right after its creation
func printToken(tok *gol.AccessToken, secret string) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), tok.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), tok.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Scopes:"), strings.Join(tok.Scopes, ", "))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Repositories:"), repositories(tok))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Ends with:"), tok.LastEight)
	if secret != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Secret:"), secret)
	}
	_ = w.Flush()
}
