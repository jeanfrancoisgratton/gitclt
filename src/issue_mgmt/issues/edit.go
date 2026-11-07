// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/edit.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// CreateIssue opens an issue in owner/repo; changed is cmd.Flags().Changed
func CreateIssue(repo string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	if strings.TrimSpace(IssueTitle) == "" {
		return &cerr.CustomError{Title: "Missing title", Message: "an issue needs a title: pass -t, --title"}
	}
	text, _, ce := body(changed)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	opts := gol.CreateIssueOptions{Title: IssueTitle, Body: text, Labels: IssueLabels, Assignees: IssueAssignees}
	if IssueMilestone != "" {
		m, ce := findMilestone(ctx, c, owner, repo, IssueMilestone)
		if ce != nil {
			return ce
		}
		opts.Milestone = m.ID
	}

	issue, e := c.Issues().Create(ctx, owner, repo, opts)
	if e != nil {
		return labelError(fmt.Sprintf("Unable to create an issue in %s/%s", owner, repo), e)
	}

	if JSONoutput {
		return shared.PrintJSON(issue, "issue")
	}
	if shared.QuietOutput {
		fmt.Println(issue.Number)
		return nil
	}
	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created issue #%d in %s", issue.Number, hftx.Brown(owner+"/"+repo))))
	if issue.HTMLURL != "" {
		fmt.Println(issue.HTMLURL)
	}
	return nil
}

var issueEditFlags = []string{"title", "body", "body-file", "label", "add-label", "remove-label", "assignee", "milestone"}

// EditIssue only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
// --add-label and --remove-label are separate requests, sent once the others went through
func EditIssue(repo, number string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}
	if !slices.ContainsFunc(issueEditFlags, changed) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt issue edit --help"}
	}

	var opts gol.EditIssueOptions
	edited := false
	if changed("title") {
		opts.Title, edited = gol.Ptr(IssueTitle), true
	}
	text, set, ce := body(changed)
	if ce != nil {
		return ce
	}
	if set {
		opts.Body, edited = gol.Ptr(text), true
	}
	if changed("label") {
		opts.Labels, edited = gol.Ptr(nonNil(IssueLabels)), true
	}
	if changed("assignee") {
		opts.Assignees, edited = gol.Ptr(nonNil(IssueAssignees)), true
	}

	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if _, ce = getIssue(ctx, c, owner, repo, nums[0]); ce != nil {
		return ce
	}
	if changed("milestone") {
		// "" removes it
		id := int64(0)
		if IssueMilestone != "" {
			m, ce := findMilestone(ctx, c, owner, repo, IssueMilestone)
			if ce != nil {
				return ce
			}
			id = m.ID
		}
		opts.Milestone, edited = gol.Ptr(id), true
	}

	if edited {
		if _, e := c.Issues().Edit(ctx, owner, repo, nums[0], opts); e != nil {
			return labelError(fmt.Sprintf("Unable to edit issue #%d", nums[0]), e)
		}
	}
	if len(IssueAddLabels) > 0 {
		if e := c.Issues().AddLabels(ctx, owner, repo, nums[0], IssueAddLabels); e != nil {
			return labelError(fmt.Sprintf("Unable to add labels to issue #%d", nums[0]), e)
		}
	}
	for _, label := range IssueRemoveLabels {
		if e := c.Issues().RemoveLabel(ctx, owner, repo, nums[0], label); e != nil {
			return labelError(fmt.Sprintf("Unable to remove label %s from issue #%d", label, nums[0]), e)
		}
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated issue #%d", nums[0])))
	}
	return nil
}

// An empty slice clears the labels or assignees; a nil one would too, but says less
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// CloseIssues closes issues, or reopens them
// Every number is checked first, and those already in the wanted state are skipped
// The run stops at the first failure; the issues changed before it stay changed
func CloseIssues(repo string, numbers []string, reopen bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers(numbers)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	want, verb := gol.IssueStateClosed, "Closed"
	if reopen {
		want, verb = gol.IssueStateOpen, "Reopened"
	}
	var todo []int
	for _, n := range nums {
		issue, ce := getRealIssue(ctx, c, owner, repo, n)
		if ce != nil {
			return ce
		}
		if issue.State == want {
			if !shared.QuietOutput {
				fmt.Printf("Issue #%d is already %s\n", n, want)
			}
			continue
		}
		todo = append(todo, n)
	}

	for _, n := range todo {
		if _, e := c.Issues().Edit(ctx, owner, repo, n, gol.EditIssueOptions{State: gol.Ptr(want)}); e != nil {
			return forge.Error(fmt.Sprintf("Unable to change issue #%d", n), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("%s issue #%d", verb, n)))
		}
	}
	return nil
}

// DeleteIssues removes issues for good, after confirmation; every number is checked before asking
// The run stops at the first failure; the issues deleted before it stay deleted
func DeleteIssues(repo string, numbers []string) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers(numbers)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	described := make([]string, 0, len(nums))
	for _, n := range nums {
		issue, ce := getRealIssue(ctx, c, owner, repo, n)
		if ce != nil {
			return ce
		}
		described = append(described, fmt.Sprintf("#%d (%s)", n, issue.Title))
	}
	if !confirm(fmt.Sprintf("Delete issue(s) %s of %s/%s, with their comments ? This cannot be undone [y/N]: ", strings.Join(described, ", "), owner, repo)) {
		fmt.Println("Aborted, no issue is deleted")
		return nil
	}

	for _, n := range nums {
		if e := c.Issues().Delete(ctx, owner, repo, n); e != nil {
			return forge.Error(fmt.Sprintf("Unable to delete issue #%d", n), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted issue #%d", n)))
		}
	}
	return nil
}

// LockIssue locks or unlocks an issue's conversation (Forgejo 17 and later, GitHub)
func LockIssue(repo, number string, unlock bool) *cerr.CustomError {
	ctx := context.Background()
	nums, ce := parseNumbers([]string{number})
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	locks, ce := c.IssueLocks()
	if ce != nil {
		return ce
	}
	if _, ce = getRealIssue(ctx, c, owner, repo, nums[0]); ce != nil {
		return ce
	}

	verb := "Locked"
	var e error
	if unlock {
		verb = "Unlocked"
		e = locks.UnlockIssue(ctx, owner, repo, nums[0])
	} else {
		if LockReason == "" && c.Backend() == gol.BackendForgejo {
			return &cerr.CustomError{Title: "Missing reason", Message: "Forgejo needs a reason to lock an issue: pass -r, --reason"}
		}
		e = locks.LockIssue(ctx, owner, repo, nums[0], gol.LockReason(LockReason))
	}
	if e != nil {
		if errors.Is(e, gol.ErrNotSupported) {
			return &cerr.CustomError{Title: "Unsupported operation", Message: e.Error()}
		}
		return forge.Error(fmt.Sprintf("Unable to change issue #%d", nums[0]), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("%s issue #%d", verb, nums[0])))
	}
	return nil
}
