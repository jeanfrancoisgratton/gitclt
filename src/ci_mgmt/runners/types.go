// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runners/types.go
// Original timestamp : 2026.11.11 18:00:00

package runners

// Owner of the repository, for a repository's runners; empty means the connection's default (see forge.Client.Target)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// Whose runners, instead of a repository's: an organization's, the connection's user's (Forgejo), or the whole
// instance's (Forgejo site admins); the cobra flags make them mutually exclusive
var OrgScope = ""
var UserScope = false
var InstanceScope = false

// runner list: add the runners the scope can use from above it, such as a repository's organization's (Forgejo)
var Inherited = false

// runner register
var RegisterName = ""
var RegisterDescription = ""
var RegisterEphemeral = false
var RegisterLabels []string
