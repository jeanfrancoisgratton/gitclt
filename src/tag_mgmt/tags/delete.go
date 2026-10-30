// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/delete.go
// Original timestamp : 2026.10.29 22:30:00

package tags

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteTags deletes tags of owner/repo on the server
// Every tag is checked before asking: it must exist, and a tag with a release is refused unless --with-release
// deletes the release too (Forgejo refuses to delete such a tag, GitHub would leave the release without its tag)
// The first failure stops the run; the tags deleted before it stay deleted
func DeleteTags(repo string, tags []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	tags = unique(tags)

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
			Message: fmt.Sprintf("%s/%s has no tag %s", owner, repo, strings.Join(missing, ", "))}
	}

	byTag, ce := releasesByTag(ctx, c, owner, repo)
	if ce != nil {
		return ce
	}
	var released []string
	for _, tag := range tags {
		if _, ok := byTag[tag]; ok {
			released = append(released, tag)
		}
	}
	if len(released) > 0 && !WithRelease {
		return &cerr.CustomError{Title: "Tag has a release", Code: http.StatusConflict,
			Message: fmt.Sprintf("these tags have a release: %s; pass --with-release to delete the releases too, with their assets",
				strings.Join(released, ", "))}
	}

	question := fmt.Sprintf("Delete tag(s) %s on the server ? [y/N]: ", strings.Join(tags, ", "))
	if len(released) > 0 {
		question = fmt.Sprintf("Delete tag(s) %s on the server, and the release(s) of %s with their assets ? [y/N]: ",
			strings.Join(tags, ", "), strings.Join(released, ", "))
	}
	if !confirm(question) {
		fmt.Println("Aborted, no tag is deleted")
		return nil
	}

	for _, tag := range tags {
		if r, ok := byTag[tag]; ok {
			if e := c.Releases().Delete(ctx, owner, repo, r.ID); e != nil {
				return forge.Error("Unable to delete the release of tag "+tag, e)
			}
			if !shared.QuietOutput {
				fmt.Println(hftx.EnabledSign("Deleted release " + hftx.Brown(tag)))
			}
		}
		if e := c.Tags().Delete(ctx, owner, repo, tag); e != nil {
			return forge.Error("Unable to delete the tag "+tag, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted tag " + hftx.Brown(tag)))
		}
	}
	return nil
}
