// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/comments.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func fetchComments(ctx context.Context, c *forge.Client, owner, repo string, number int) ([]gol.Comment, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Comment], error) {
		return c.Issues().ListComments(ctx, owner, repo, number, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the comments of issue #%d", number), e)
	}
	return list, nil
}

// ListComments lists what was said on an issue
func ListComments(repo, number string, displayoutput bool) ([]gol.Comment, *cerr.CustomError) {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return nil, ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	if _, ce = getIssue(ctx, c, owner, repo, nums[0]); ce != nil {
		return nil, ce
	}
	list, ce := fetchComments(ctx, c, owner, repo, nums[0])
	if ce != nil {
		return nil, ce
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "comment list")
	}
	if shared.QuietOutput {
		for _, m := range list {
			fmt.Println(m.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("Issue #%d has no comment\n", nums[0])
		return list, nil
	}
	shared.Page(renderComments(list), 0)
	return list, nil
}

// commentBody is the text of a comment: it needs one, unlike an issue
func commentBody(changed func(string) bool) (string, *cerr.CustomError) {
	text, _, ce := body(changed)
	if ce != nil {
		return "", ce
	}
	if strings.TrimSpace(text) == "" {
		return "", &cerr.CustomError{Title: "Missing text", Message: "a comment needs a text: pass -b, --body or -F, --body-file"}
	}
	return text, nil
}

// AddComment comments on an issue; it shows the new comment's ID, which is how it can be edited or deleted
func AddComment(repo, number string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}
	text, ce := commentBody(changed)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if _, ce = getIssue(ctx, c, owner, repo, nums[0]); ce != nil {
		return ce
	}

	comment, e := c.Issues().CreateComment(ctx, owner, repo, nums[0], text)
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to comment on issue #%d", nums[0]), e)
	}
	if shared.QuietOutput {
		fmt.Println(comment.ID)
		return nil
	}
	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Added comment %d to issue #%d", comment.ID, nums[0])))
	return nil
}

// EditComment replaces a comment's text
func EditComment(repo, id string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{id})
	if ce != nil {
		return ce
	}
	text, ce := commentBody(changed)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if _, e := c.Issues().EditComment(ctx, owner, repo, ids[0], text); e != nil {
		return forge.Error(fmt.Sprintf("Unable to edit comment %d", ids[0]), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated comment %d", ids[0])))
	}
	return nil
}

// DeleteComments removes comments, after confirmation
// The run stops at the first failure; the comments deleted before it stay deleted
func DeleteComments(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs(args)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	shown := make([]string, len(ids))
	for i, id := range ids {
		shown[i] = strconv.FormatInt(id, 10)
	}
	if !confirm(fmt.Sprintf("Delete comment(s) %s of %s/%s ? [y/N]: ", strings.Join(shown, ", "), owner, repo)) {
		fmt.Println("Aborted, no comment is deleted")
		return nil
	}

	for _, id := range ids {
		if e := c.Issues().DeleteComment(ctx, owner, repo, id); e != nil {
			return forge.Error(fmt.Sprintf("Unable to delete comment %d", id), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted comment %d", id)))
		}
	}
	return nil
}
