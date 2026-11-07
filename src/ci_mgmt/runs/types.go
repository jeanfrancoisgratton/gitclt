// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/types.go
// Original timestamp : 2026.11.07 10:00:00

package runs

// Owner / organization; empty means the connection's default (see forge.Client.Target)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// run list: the filters, and how many runs at most (0: all of them)
var ListWorkflow = ""
var ListStatus = ""
var ListEvent = ""
var ListBranch = ""
var ListSHA = ""
var ListLimit = 30

// artifact list
var ArtifactRun int64
var ArtifactName = ""

// run logs, artifact download: where the file goes, and whether an existing one is replaced
var OutputFile = ""
var Overwrite = false
