// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/protections/types.go
// Original timestamp : 2026.10.26 08:14:52

package protections

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// The settings of a rule, for branch protection create and edit
var RequiredApprovals = 0
var DismissStaleApprovals = false
var StatusChecks []string
var RequireUpToDate = false
var RequireSignedCommits = false
var ApplyToAdmins = false

// Who may push directly: NoPush blocks everyone, PushUsers and PushTeams name who may; AnyPush removes the restriction
var NoPush = false
var AnyPush = false
var PushUsers []string
var PushTeams []string

// GitHub and Bitbucket; Forgejo always blocks these
var AllowDeletion = false
var AllowForcePush = false

// GitHub only
var RequireLinearHistory = false
var RequireConversationResolution = false

// Forgejo and Bitbucket
var AnyMerge = false
var MergeUsers []string
var MergeTeams []string

// Forgejo only
var AnyApprover = false
var ApproverUsers []string
var ApproverTeams []string
var BlockOnRejectedReviews = false
var BlockOnReviewRequests = false
var ProtectedFiles = ""
var UnprotectedFiles = ""
