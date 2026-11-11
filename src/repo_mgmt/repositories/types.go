// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/types.go
// Original timestamp : 2026.10.19 10:00:00

package repositories

var JSONoutput = false
var AssumeYes = false

// Owner / organization, uses the current user if left empty
var OwnerOrg = ""

// SortBy : how to show the output (repo ls), in descending order; SortByAsc does the same in ascending order (they are mutually exclusive)
// supported values: ID (the default; Repository on Bitbucket, which has no IDs), Owner, Created At, Last Updated. Any value outside of this list reverts to the default
var SortBy = ""
var SortByAsc = ""

// Various variables used at repo creation, with their default values
var TemplateRepo = false
var InitializeRepo = false
var RepoDefaultBranch = "" // empty: see createBranch
var RepoDescription = ""
var PublicRepo = true
var CreateActions = true // --actions=false creates the repository with Actions off

// repo edit: only the flags explicitly passed are sent (see EditRepo)
var EditRepoName = ""
var EditRepoDescription = ""
var EditRepoWebsite = ""
var EditRepoDefaultBranch = ""
var EditRepoPublic = false
var EditRepoArchived = false
var EditRepoTemplate = false

// repo edit: features (Forgejo units, GitHub features); see editOptionsFromFlags for which forge has which
var EditActions, EditIssues, EditWiki, EditPullRequests, EditProjects, EditReleases, EditPackages, EditDiscussions bool

// repo edit: merge settings
var EditMergeCommit, EditRebase, EditRebaseMerge, EditSquash, EditFastForwardOnly, EditDeleteBranch bool
var EditAutoMerge, EditUpdateBranch, EditRebaseUpdate bool
var EditDefaultMerge = ""

// repo edit: issue tracker and wiki settings (Forgejo)
var EditTimeTracking, EditTimeTrackingContributorsOnly, EditDependencies bool
var EditExternalTracker, EditTrackerFormat, EditTrackerStyle, EditTrackerPattern = "", "", "", ""
var EditExternalWiki = ""
var EditWikiEditableByAll = false

// repo edit: Actions settings (GitHub)
var EditActionsAllowed, EditActionsToken = "", ""
var EditActionsApprovePRs = false
