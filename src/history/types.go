// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/types.go
// Original timestamp : 2026.10.30 14:40:32

package history

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

var JSONoutput = false

// history ls: where to start from (empty: the default branch), which file or directory the commits must touch,
// and how many commits to show at most (0: all of them)
var ListRef = ""
var ListPath = ""
var ListLimit = 30

// history ls: the authoring interval, as YYYY-MM-DD or RFC 3339; GitHub only
var ListSince = ""
var ListUntil = ""
