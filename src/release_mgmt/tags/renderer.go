// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/renderer.go
// Original timestamp : 2026.10.29 22:30:00

package tags

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Commits are shown abbreviated, like git does
func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// title is the first line of an annotation
func title(message string) string {
	t, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	return t
}

// The Release column is dropped where the forge has no releases (Bitbucket), the Message column when no tag has
// an annotation (or the forge does not list them: GitHub)
func renderTagTable(list []Tag, withRelease bool) {
	withMessage := false
	for _, tg := range list {
		withMessage = withMessage || tg.Message != ""
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	header := table.Row{"Tag", "Commit"}
	if withRelease {
		header = append(header, "Release")
	}
	if withMessage {
		header = append(header, "Message")
	}
	t.AppendHeader(header)
	for _, tg := range list {
		row := table.Row{tg.Name, short(tg.SHA)}
		if withRelease {
			row = append(row, tg.Release)
		}
		if withMessage {
			row = append(row, tg.Message)
		}
		t.AppendRow(row)
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderTag(tg Tag, withRelease bool) {
	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Tag name:"), tg.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Commit:"), tg.SHA)
	if tg.Message == "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Type:"), "lightweight")
	} else {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Type:"), "annotated")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Message:"), tg.Message)
	}
	if withRelease {
		switch {
		case tg.Release == "":
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Release:"), "none")
		case tg.ReleaseURL != "":
			_, _ = fmt.Fprintf(w, "%s\t%s (%s)\n", hftx.Blue("Release:"), tg.Release, tg.ReleaseURL)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Release:"), tg.Release)
		}
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Tarball:"), tg.TarballURL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Zipball:"), tg.ZipballURL)
	_ = w.Flush()
}
