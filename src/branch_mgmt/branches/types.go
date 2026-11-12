// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/types.go
// Original timestamp : 2026.10.26 08:14:52

package branches

import (
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

var JSONoutput = false
var AssumeYes = false
var SuppressWarnings = false

// Owner / organization of the repository on the server
var OwnerOrg = ""

// Which sides branch ls and get show: AllBranches (ls only) overrides the other two, and neither (or both) of them also
// means both
// branch rm and rename need RemoteOnly (-r): they only act on the server
var LocalOnly = false
var RemoteOnly = false
var AllBranches = false

// Branch is one row of branch ls, or what branch get shows: a branch as seen in the local working copy and/or on the
// server; the parts that one side cannot tell stay empty
type Branch struct {
	Name       string `json:"name"`                // the local name, or the server's for a branch not in the clone
	ServerName string `json:"serverName,omitzero"` // the server's name, when the local branch tracks another one
	LocalSHA   string `json:"localSha,omitzero"`   // empty: not in the local working copy
	ServerSHA  string `json:"serverSha,omitzero"`  // empty: not on the server
	Current    bool   `json:"current,omitzero"`    // checked out in the local working copy
	IsDefault  bool   `json:"isDefault,omitzero"`  // the default branch, which the others are compared with
	Protected  bool   `json:"protected,omitzero"`  // a protection rule applies on the server
	// The head commit: the server's, else the local one's; GitHub's branch list does not report it
	LastCommit *gol.CommitSummary `json:"lastCommit,omitzero"`

	// Read in the clone: the branch it was created from (Exact when its reflogs tell, else a guess, with the branches
	// as likely in Others), and the branch it tracks (Upstream.SHA is empty when it is gone from the remote)
	Parent   *local.Origin   `json:"parent,omitzero"`
	Upstream *local.Upstream `json:"upstream,omitzero"`

	// Against the default branch: the clone's count for a local branch, the server's otherwise; Merged when it has
	// nothing the default lacks. MergeBase is their newest common commit, read in the clone
	VsDefault *gol.Comparison `json:"vsDefault,omitzero"`
	Merged    *bool           `json:"merged,omitzero"`
	MergeBase string          `json:"mergeBase,omitzero"`

	// On the server: the CI state of the head commit, and the open pull requests from the branch
	CI           *gol.CombinedStatus `json:"ci,omitzero"`
	PullRequests []PullRequest       `json:"pullRequests,omitzero"`

	// When both sides differ: how the local branch relates to the server's, as git status puts it
	Ahead    int  `json:"ahead,omitzero"`    // local commits the server lacks
	Behind   int  `json:"behind,omitzero"`   // server commits the local branch lacks
	Unpushed bool `json:"unpushed,omitzero"` // the local head is unknown to the server: it cannot count

	// What could not be read: refused by the server, unknown to the forge, or missing from the clone (branch get)
	Missing []gol.PartError `json:"missing,omitzero"`
}

// PullRequest is an open pull request from a branch
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Base   string `json:"base"` // the branch it merges into
	Author string `json:"author"`
}

func pullRequest(pr gol.PullRequest) PullRequest {
	return PullRequest{Number: pr.Number, Title: pr.Title, Base: pr.Base.Ref, Author: pr.Author}
}
