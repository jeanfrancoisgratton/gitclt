// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/renderer.go
// Original timestamp : 2026.10.29 19:10:00

package releases

import (
	"fmt"
	"os"
	"time"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006.01.02 15:04")
}

// kind words what a release is: draft and prerelease can go together
func kind(r gol.Release) string {
	k := "release"
	if r.Prerelease {
		k = "prerelease"
	}
	if r.Draft {
		k = "draft " + k
	}
	return k
}

func renderReleaseTable(list []gol.Release) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Tag", "Title", "Type", "Published", "Author", "Assets"})
	for _, r := range list {
		// drafts are not published: their creation date stands in
		published := date(r.PublishedAt)
		if r.Draft {
			published = "(created " + date(r.CreatedAt) + ")"
		}
		t.AppendRow(table.Row{r.TagName, r.Name, kind(r), published, r.Author, len(r.Assets)})
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func renderAssetTable(list []gol.Asset) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Asset", "Size", "Downloads", "Uploaded"})
	for _, a := range list {
		// Forgejo's external assets are links: no size of their own
		size := humanSize(a.Size)
		if a.External {
			size = "external"
		}
		t.AppendRow(table.Row{a.Name, size, fmt.Sprint(a.DownloadCount), date(a.CreatedAt)})
	}
	t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}
