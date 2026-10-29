// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/types.go
// Original timestamp : 2026.10.26 08:14:52

package releases

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// release upload
var CreateRelease = false
var DraftRelease = false
var Prerelease = false
var ReplaceAssets = false

// release edit: only the flags explicitly passed are sent
var EditTitle = ""
var EditNotes = ""
var EditNotesFile = ""
var EditDraft = false
var EditPrerelease = false

// A local file to upload, already checked; name is what the asset will be called
type localFile struct {
	path, name string
	size       int64
}
