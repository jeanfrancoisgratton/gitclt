// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/types.go
// Original timestamp : 2026.10.19 10:00:00

package repositories

var RepoListJSONOutput = false

// Forgejo's maximum page size is a server setting (50 by default); GitHub's is 100
const listPageSize = 50
