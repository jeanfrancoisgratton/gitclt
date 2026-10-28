// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/types.go
// Original timestamp : 2026.10.26 08:14:52

package branches

import gol "github.com/jeanfrancoisgratton/gitoperationslib"

var JSONoutput = false
var AssumeYes = false
var SuppressWarnings = false

// Owner / organization of the repository on the server
var OwnerOrg = ""

// Which branches branch ls shows: AllBranches overrides the other two, and neither (or both) of them also means all
var LocalOnly = false
var RemoteOnly = false
var AllBranches = false

// Branch is one row of branch ls: a branch name, as seen in the local working copy and/or on the server
type Branch struct {
	Name      string `json:"name"`
	LocalSHA  string `json:"localSha,omitzero"`  // empty: not in the local working copy
	ServerSHA string `json:"serverSha,omitzero"` // empty: not on the server
	Current   bool   `json:"current,omitzero"`   // checked out in the local working copy
	IsDefault bool   `json:"isDefault,omitzero"` // the server's default branch
	Protected bool   `json:"protected,omitzero"` // a protection rule applies on the server
	// The server's head commit; GitHub's branch list does not report it (nil)
	LastCommit *gol.CommitSummary `json:"lastCommit,omitzero"`

	// When both sides differ: how the local branch relates to the server's, as git status puts it
	Ahead    int  `json:"ahead,omitzero"`    // local commits the server lacks
	Behind   int  `json:"behind,omitzero"`   // server commits the local branch lacks
	Unpushed bool `json:"unpushed,omitzero"` // the local head is unknown to the server: it cannot count
}
