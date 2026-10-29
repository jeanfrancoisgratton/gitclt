// gitclt
// Tests release list and release assets against the fake Forgejo of upload_test.go

package releases

import (
	"net/http"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestListReleases(t *testing.T) {
	defaults(t)
	s := &forgeState{}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 2, tag: "v2.0.0-rc1", draft: true, pre: true}, {id: 1, tag: "v1.0.0"}}

	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { _, err = ListReleases("web", true) }))
	if err != nil {
		t.Fatal(err.Error())
	}
	// Drafts are listed too: the list sees them, GetByTag does not
	if !strings.Contains(out, "v2.0.0-rc1") || !strings.Contains(out, "draft prerelease") || !strings.Contains(out, "v1.0.0") {
		t.Fatalf("output:\n%s", out)
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.StripANSI(testutil.Run(t, func() { _, err = ListReleases("web", true) }))
	if err != nil || !strings.Contains(out, `"tagName": "v1.0.0"`) {
		t.Fatalf("%v, json:\n%s", err, out)
	}

	if _, err = ListReleases("nope", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("unknown repo: %+v", err)
	}
}

func TestListAssets(t *testing.T) {
	defaults(t)
	s := &forgeState{}
	releaseServer(t, s)
	s.releases = []*fakeRelease{
		{id: 1, tag: "v1.0.0", assets: []fakeAsset{{id: 7, name: "a.rpm", content: "aaa"}, {id: 8, name: "b.deb", content: "b"}}},
		{id: 2, tag: "v2.0.0", draft: true, assets: []fakeAsset{{id: 9, name: "c.apk", content: "c"}}},
	}

	assets, err := ListAssets("web", "v1.0.0", false)
	if err != nil || len(assets) != 2 || assets[0].Name != "a.rpm" || assets[0].Size != 3 {
		t.Fatalf("%v, assets %+v", err, assets)
	}
	// A draft's release is found through the list
	var out string
	out = testutil.StripANSI(testutil.Run(t, func() { _, err = ListAssets("web", "v2.0.0", true) }))
	if err != nil || !strings.Contains(out, "c.apk") {
		t.Fatalf("%v, draft:\n%s", err, out)
	}

	if _, err = ListAssets("web", "v3.0.0", false); err == nil || err.Title != "No such release" || err.Code != http.StatusNotFound {
		t.Fatalf("no release: %+v", err)
	}
}

// Bitbucket is refused right after connecting, like upload
func TestListReleases_Bitbucket(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"bb":{"url":"https://bitbucket.org","token":"t","owner":"ws","servertype":"bitbucket"}},"default_connection":"bb"}`)
	if _, err := ListReleases("web", false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("got %+v", err)
	}
}
