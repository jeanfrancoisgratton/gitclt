// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/create.go
// Original timestamp : 2026.10.29 22:30:00

package tags

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// CreateTag tags --target (a branch, tag or commit; the default branch when empty) on the server
// -m makes an annotated tag, otherwise a lightweight one (Bitbucket always annotates, with its own message)
func CreateTag(repo, tag string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}

	// Some forges answer an existing tag with a vague 409 or 422: say it plainly
	if _, e := c.Tags().Get(ctx, owner, repo, tag); e == nil {
		return &cerr.CustomError{Title: "Tag already exists", Code: http.StatusConflict,
			Message: fmt.Sprintf("%s/%s already has a tag %s", owner, repo, tag)}
	} else if !gol.IsNotFound(e) {
		return forge.Error("Unable to get the tag "+tag, e)
	}

	t, e := c.Tags().Create(ctx, owner, repo, gol.CreateTagOptions{Name: tag, Target: TagTarget, Message: TagMessage})
	if e != nil {
		return forge.Error("Unable to create the tag "+tag, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created tag %s at %s", hftx.Brown(tag), short(t.SHA))))
	}
	return nil
}
