// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/renderer.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

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

const timeFormat = "2006.01.02 15:04"

// A webhook subscribed to everything lists dozens of events: the table only shows the first ones, and get shows them all
const eventsInTable = 3

func eventNames(events []gol.HookEvent) []string {
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = string(e)
	}
	return names
}

func summarizeEvents(events []gol.HookEvent) string {
	names := eventNames(events)
	if len(names) <= eventsInTable {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(names[:eventsInTable], ", "), len(names)-eventsInTable)
}

// The type and branch filter columns only exist on Forgejo: they are left out when nothing fills them
func renderHookTable(list []gol.Webhook) {
	hasType := slices.ContainsFunc(list, func(h gol.Webhook) bool { return h.Type != "" })
	hasFilter := slices.ContainsFunc(list, func(h gol.Webhook) bool { return h.BranchFilter != "" })

	header := table.Row{"ID"}
	if hasType {
		header = append(header, "Type")
	}
	header = append(header, "URL", "Active", "Events")
	if hasFilter {
		header = append(header, "Branch filter")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(header)
	for _, h := range list {
		row := table.Row{h.ID}
		if hasType {
			row = append(row, string(h.Type))
		}
		row = append(row, h.URL, h.Active, summarizeEvents(h.Events))
		if hasFilter {
			row = append(row, h.BranchFilter)
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderHook(h *gol.Webhook) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), h.ID)
	printIfSet(w, "Type:", string(h.Type))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("URL:"), h.URL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Content type:"), h.ContentType)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Active:"), h.Active)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Events:"), strings.Join(eventNames(h.Events), ", "))
	printIfSet(w, "Branch filter:", h.BranchFilter)
	// Its value is a credential: neither the library nor gitclt shows it
	if h.HasAuthorizationHeader {
		_, _ = fmt.Fprintf(w, "%s\tset\n", hftx.Blue("Authorization header:"))
	}
	if h.InsecureSSL {
		_, _ = fmt.Fprintf(w, "%s\tnot checked\n", hftx.Blue("TLS certificate:"))
	}
	if s := h.Slack; s != nil {
		printIfSet(w, "Slack channel:", s.Channel)
		printIfSet(w, "Slack username:", s.Username)
		printIfSet(w, "Slack icon URL:", s.IconURL)
		printIfSet(w, "Slack color:", s.Color)
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), formatTime(h.CreatedAt))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), formatTime(h.UpdatedAt))
	_ = w.Flush()
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

// formatTime returns "-" for what the forge did not report, instead of 0001.01.01
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format(timeFormat)
}
