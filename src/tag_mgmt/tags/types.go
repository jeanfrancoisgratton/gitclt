// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/types.go
// Original timestamp : 2026.10.26 08:14:52

package tags

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Owner)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// tag create
var TagTarget = ""  // branch, tag or commit to tag; empty means the default branch
var TagMessage = "" // makes an annotated tag; empty makes a lightweight one

// tag rm: delete the server's releases (and their assets) along with their tags
var WithRelease = false

// Tag is a tag on the server, with its release if it has one (Forgejo and GitHub; Bitbucket has no releases)
// gitclt only works on the server's tags: the local ones are git's business
type Tag struct {
	Name       string `json:"name"`
	SHA        string `json:"sha"`              // the commit it points to (annotated tags are peeled)
	Message    string `json:"message,omitzero"` // the annotation's first line; empty: lightweight, or not reported (GitHub's list)
	Release    string `json:"release,omitzero"` // what the release is (release, draft prerelease...); empty: none
	ReleaseURL string `json:"releaseUrl,omitzero"`
	TarballURL string `json:"tarballUrl,omitzero"`
	ZipballURL string `json:"zipballUrl,omitzero"`
}
