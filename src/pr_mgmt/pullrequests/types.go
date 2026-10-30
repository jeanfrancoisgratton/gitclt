// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/pr_mgmt/pullrequests/types.go
// Original timestamp : 2026.10.30 01:00:00

package pullrequests

import gol "github.com/jeanfrancoisgratton/gitoperationslib"

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

var JSONoutput = false

// pr ls: open, closed (merged included) or all
var ListState = "open"

// pr create, pr edit: only the flags explicitly passed are sent by pr edit
var PRTitle = ""
var PRBody = ""
var PRBodyFile = ""
var PRBase = ""
var PRDraft = false
var PRReviewers []string

// pr merge
var MergeMethod = "merge"
var MergeTitle = ""
var MergeMessage = ""
var DeleteBranch = false
var AutoMerge = false
var CancelAutoMerge = false

// pr review: one verdict, and its text
var ReviewApprove = false
var ReviewRequestChanges = false
var ReviewComment = false
var ReviewBody = ""

// pr reviewers: remove the users instead of requesting their review
var RemoveReviewers = false

// PullRequest is what pr get shows: the pull request, and its reviews (requested ones included, on Forgejo)
type PullRequest struct {
	gol.PullRequest
	Reviews []gol.Review `json:"reviews"`
}
