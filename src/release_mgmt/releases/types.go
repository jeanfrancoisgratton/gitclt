// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/types.go
// Original timestamp : 2026.10.26 08:14:52

package releases

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

// release upload
var CreateRelease = false
var DraftRelease = false
var Prerelease = false
var ReplaceAssets = false
