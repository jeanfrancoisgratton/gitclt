// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/types.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import gol "github.com/jeanfrancoisgratton/gitoperationslib"

// Owner / organization; empty means the connection's default (see forge.Client.Target)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// issue list
var ListState = "open"
var ListLabels []string

// issue get
var WithComments = false

// issue create, issue edit: only the flags explicitly passed are sent by issue edit
var IssueTitle = ""
var IssueBody = ""
var IssueBodyFile = ""
var IssueLabels []string
var IssueAddLabels []string
var IssueRemoveLabels []string
var IssueAssignees []string
var IssueMilestone = ""

// issue lock
var LockReason = ""

// issue label
var LabelName = ""
var LabelColor = ""
var LabelDescription = ""

// issue milestone
var MilestoneListState = "open"
var MilestoneTitle = ""
var MilestoneDescription = ""
var MilestoneDue = ""
var MilestoneState = ""

// Issue is what issue get shows: the issue and, with --comments, what was said on it
type Issue struct {
	gol.Issue
	Discussion []gol.Comment `json:"discussion,omitzero"`
}
