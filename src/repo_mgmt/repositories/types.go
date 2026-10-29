// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/types.go
// Original timestamp : 2026.10.19 10:00:00

package repositories

var JSONoutput = false

// Owner / organization, uses the current user if left empty
var OwnerOrg = ""

// SortBy : how to show the output (repo ls), in descending order; SortByAsc does the same in ascending order (they are mutually exclusive)
// supported values: ID (the default; Repository on Bitbucket, which has no IDs), Owner, Created At, Last Updated. Any value outside of this list reverts to the default
var SortBy = ""
var SortByAsc = ""

// Various variables used at repo creation, with their default values
var TemplateRepo = false
var InitializeRepo = false
var RepoDefaultBranch = "develop"
var RepoDescription = ""
var PublicRepo = true

// repo edit: only the flags explicitly passed are sent (see EditRepo)
var EditRepoName = ""
var EditRepoDescription = ""
var EditRepoWebsite = ""
var EditRepoDefaultBranch = ""
var EditRepoPublic = false
var EditRepoArchived = false
var EditRepoTemplate = false
