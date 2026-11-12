// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runners/renderer.go
// Original timestamp : 2026.11.11 18:00:00

package runners

import (
	"fmt"
	"gitclt/shared"
	"os"
	"strings"
	"text/tabwriter"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// State is the same word on every forge (offline, idle, busy); Status is the forge's own
func renderRunnerTable(list []gol.Runner) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "State", "Labels", "Version", "Ephemeral"})
	for _, r := range list {
		t.AppendRow(table.Row{r.ID, r.Name, state(r), strings.Join(r.Labels, ", "), r.Version, yesNo(r.Ephemeral)})
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(77)}
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

func renderRunner(r *gol.Runner) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), r.ID)
	printIfSet(w, "UUID:", r.UUID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), r.Name)
	printIfSet(w, "Description:", r.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s (%s)\n", hftx.Blue("State:"), state(*r), r.Status)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Online:"), shared.Sign(r.Online))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Busy:"), shared.Sign(r.Busy))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Ephemeral:"), shared.Sign(r.Ephemeral))
	printIfSet(w, "Labels:", strings.Join(r.Labels, ", "))
	printIfSet(w, "Version:", r.Version)
	printIfSet(w, "OS:", r.OS)
	_ = w.Flush()
}
