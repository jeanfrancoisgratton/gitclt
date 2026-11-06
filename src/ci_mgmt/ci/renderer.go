// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/ci/renderer.go
// Original timestamp : 2026.11.06 18:00:00

package ci

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// A value can be long, or span several lines: the table only shows its beginning, and get shows it whole
const valueInTable = 60

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006.01.02 15:04")
}

func newTable(header table.Row) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(header)
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	return t
}

func shortValue(value string) string {
	first, _, more := strings.Cut(value, "\n")
	if r := []rune(first); len(r) > valueInTable {
		return string(r[:valueInTable]) + "..."
	}
	if more {
		return first + "..."
	}
	return first
}

// The forges do not all report dates: the columns are left out when nothing fills them
func renderSecretTable(list []gol.Secret) {
	hasCreated := slices.ContainsFunc(list, func(s gol.Secret) bool { return !s.CreatedAt.IsZero() })
	hasUpdated := slices.ContainsFunc(list, func(s gol.Secret) bool { return !s.UpdatedAt.IsZero() })
	header := table.Row{"Name"}
	if hasCreated {
		header = append(header, "Created")
	}
	if hasUpdated {
		header = append(header, "Updated")
	}
	t := newTable(header)
	for _, s := range list {
		row := table.Row{s.Name}
		if hasCreated {
			row = append(row, date(s.CreatedAt))
		}
		if hasUpdated {
			row = append(row, date(s.UpdatedAt))
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(222)}
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderVariableTable(list []gol.Variable) {
	hasUpdated := slices.ContainsFunc(list, func(v gol.Variable) bool { return !v.UpdatedAt.IsZero() })
	header := table.Row{"Name", "Value"}
	if hasUpdated {
		header = append(header, "Updated")
	}
	t := newTable(header)
	for _, v := range list {
		row := table.Row{v.Name, shortValue(v.Value)}
		if hasUpdated {
			row = append(row, date(v.UpdatedAt))
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Color.Header = text.Colors{text.FgHiWhite, text.Bg256Color(222)}
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderVariable(v *gol.Variable) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), v.Name)
	if d := date(v.CreatedAt); d != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), d)
	}
	if d := date(v.UpdatedAt); d != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), d)
	}
	_, _ = fmt.Fprintf(w, "%s\n", hftx.Blue("Value:"))
	_ = w.Flush()
	fmt.Println(v.Value)
}
