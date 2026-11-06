// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/ci/types.go
// Original timestamp : 2026.11.06 18:00:00

package ci

// Owner / organization / workspace of the repository; empty means the connection's default (see forge.Client.Target)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// Where the secrets and variables live: a repository by default, or one of these
var ScopeOrg = ""
var ScopeUser = false
var ScopeEnv = ""

// Where a value comes from, when it is not typed or passed as an argument
var ValueFile = ""
var ValueStdin = false
