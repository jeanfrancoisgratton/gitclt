// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/types.go
// Original timestamp : 2026.10.19 10:00:00

package repositories

var JSONoutput = false

// Forgejo's maximum page size is a server setting (50 by default); GitHub's and Bitbucket's are 100
const listPageSize = 50

// Owner / organization, uses the current user if left empty
var OwnerOrg = ""

// SortBy : how to show the output (repo ls), in descending order; SortByAsc does the same in ascending order (they are mutually exclusive)
// supported values: ID (the default; Repository on Bitbucket, which has no IDs), Owner, Created At, Last Updated. Any value outside of this list reverts to the default
var SortBy = ""
var SortByAsc = ""
