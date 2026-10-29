// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/edit.go
// Original timestamp : 2026.10.29 21:30:00

package releases

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditRelease only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
// The release is found by its tag, drafts included; --draft=false publishes a draft, like release publish does
func EditRelease(repo, tag string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()

	ero, ce := editOptionsFromFlags(changed)
	if ce != nil {
		return ce
	}
	if ero == (gol.EditReleaseOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt release edit --help"}
	}

	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	rel, ce := findRelease(ctx, c, owner, repo, tag)
	if ce != nil {
		return ce
	}
	if rel == nil {
		return &cerr.CustomError{Title: "No such release", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no release for tag %s", owner, repo, tag)}
	}
	// Publishing a draft whose tag is not on the server would create the tag from the release's target
	if rel.Draft && ero.Draft != nil && !*ero.Draft {
		if ce = checkTags(ctx, c, owner, repo, []string{tag}); ce != nil {
			return ce
		}
	}

	if rel, e := c.Releases().Edit(ctx, owner, repo, rel.ID, ero); e != nil {
		return forge.Error("Unable to edit the release of tag "+tag, e)
	} else if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated %s %s", kind(*rel), hftx.Brown(tag))))
	}
	return nil
}

// PublishReleases turns the draft releases of owner/repo's tags into published ones
// Every tag is checked before the first publication: it must have a release, and be on the server (a draft's tag
// is otherwise created from its target when published); releases already published are skipped
// The run stops at the first failure; the releases published before it stay published
func PublishReleases(repo string, tags []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	tags = unique(tags)

	// One listing finds every release, drafts included (GetByTag does not see them)
	all, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Release], error) {
		return c.Releases().List(ctx, owner, repo, o)
	})
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to list the releases of %s/%s", owner, repo), e)
	}
	byTag := map[string]gol.Release{}
	for _, r := range all {
		byTag[r.TagName] = r
	}
	var missing, drafts []string
	for _, tag := range tags {
		r, ok := byTag[tag]
		switch {
		case !ok:
			missing = append(missing, tag)
		case r.Draft:
			drafts = append(drafts, tag)
		case !shared.QuietOutput:
			fmt.Printf("Release %s is already published\n", hftx.Brown(tag))
		}
	}
	if len(missing) > 0 {
		return &cerr.CustomError{Title: "No such release", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no release for %s", owner, repo, strings.Join(missing, ", "))}
	}
	if ce = checkTags(ctx, c, owner, repo, drafts); ce != nil {
		return ce
	}

	for _, tag := range drafts {
		rel, e := c.Releases().Edit(ctx, owner, repo, byTag[tag].ID, gol.EditReleaseOptions{Draft: gol.Ptr(false)})
		if e != nil {
			return forge.Error("Unable to publish the release of tag "+tag, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Published %s %s", kind(*rel), hftx.Brown(tag))))
			if rel.HTMLURL != "" {
				fmt.Printf("Release %s: %s\n", hftx.Green(tag), rel.HTMLURL)
			}
		}
	}
	return nil
}

// checkTags makes sure every tag is on the server, and reports all the missing ones at once
func checkTags(ctx context.Context, c *forge.Client, owner, repo string, tags []string) *cerr.CustomError {
	var missing []string
	for _, tag := range tags {
		if _, e := c.Tags().Get(ctx, owner, repo, tag); e != nil {
			if !gol.IsNotFound(e) {
				return forge.Error("Unable to get the tag "+tag, e)
			}
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		return &cerr.CustomError{Title: "No such tag", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no tag %s; push it first (git push origin %s)", owner, repo, strings.Join(missing, ", "), strings.Join(missing, " "))}
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) (gol.EditReleaseOptions, *cerr.CustomError) {
	var ero gol.EditReleaseOptions

	if changed("title") {
		ero.Name = gol.Ptr(EditTitle)
	}
	if changed("notes") {
		ero.Body = gol.Ptr(EditNotes)
	}
	if changed("notes-file") {
		b, e := os.ReadFile(EditNotesFile)
		if e != nil {
			return ero, &cerr.CustomError{Title: "Unable to read " + EditNotesFile, Message: e.Error()}
		}
		ero.Body = gol.Ptr(string(b))
	}
	if changed("draft") {
		ero.Draft = gol.Ptr(EditDraft)
	}
	if changed("prerelease") {
		ero.Prerelease = gol.Ptr(EditPrerelease)
	}
	return ero, nil
}
