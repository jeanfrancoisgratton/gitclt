// gitclt
// Tests release delete and release delete-assets against the fake Forgejo of upload_test.go

package releases

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestDeleteReleases(t *testing.T) {
	defaults(t)
	s := &forgeState{}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0"}, {id: 2, tag: "v1.1.0-rc1", draft: true}, {id: 3, tag: "v1.1.0"}}

	// Anything but y/yes aborts
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteReleases("web", []string{"v1.0.0"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || len(s.deletes) != 0 {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}

	// An unknown tag is caught before anything is deleted, even when it is not the first
	if err = DeleteReleases("web", []string{"v1.0.0", "v9", "v8"}); err == nil || err.Title != "No such release" ||
		!strings.Contains(err.Message, "v9, v8") || len(s.deletes) != 0 {
		t.Fatalf("unknown tag: %+v, deleted %v", err, s.deletes)
	}

	// Drafts included; a repeated tag is only deleted once
	out = testutil.Run(t, func() { err = DeleteReleases("web", []string{"v1.0.0", "v1.1.0-rc1", "v1.0.0"}) }, "y")
	if err != nil || !slices.Equal(s.deletes, []string{"v1.0.0", "v1.1.0-rc1"}) || !strings.Contains(out, "Deleted release v1.1.0-rc1") {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}
	if len(s.releases) != 1 || s.releases[0].tag != "v1.1.0" {
		t.Fatalf("left %+v", s.releases)
	}
}

// The first failure stops the run
func TestDeleteReleases_Failure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	s := &forgeState{}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1"}, {id: 2, tag: "FAIL"}, {id: 3, tag: "v3"}}

	var err *cerr.CustomError
	testutil.Run(t, func() { err = DeleteReleases("web", []string{"v1", "FAIL", "v3"}) })
	if err == nil || err.Code != http.StatusForbidden || !slices.Equal(s.deletes, []string{"v1"}) {
		t.Fatalf("got %+v, deleted %v", err, s.deletes)
	}
}

func TestDeleteAssets(t *testing.T) {
	defaults(t)
	s := &forgeState{}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0", assets: []fakeAsset{{id: 7, name: "a.rpm"}, {id: 8, name: "b.deb"}, {id: 9, name: "c.apk"}}}}

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteAssets("web", "v1.0.0", []string{"a.rpm"}) }, "")
	if err != nil || !strings.Contains(out, "Aborted") || len(s.deletes) != 0 {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}

	testutil.Set(t, &AssumeYes, true)
	if err = DeleteAssets("web", "v1.0.0", []string{"a.rpm", "nope.rpm"}); err == nil || err.Title != "No such asset" || len(s.deletes) != 0 {
		t.Fatalf("unknown asset: %+v, deleted %v", err, s.deletes)
	}
	if err = DeleteAssets("web", "v2.0.0", []string{"a.rpm"}); err == nil || err.Title != "No such release" {
		t.Fatalf("no release: %+v", err)
	}

	out = testutil.Run(t, func() { err = DeleteAssets("web", "v1.0.0", []string{"c.apk", "a.rpm"}) })
	if err != nil || !slices.Equal(s.deletes, []string{"c.apk", "a.rpm"}) || !strings.Contains(out, "Deleted asset a.rpm") {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"b.deb"}) {
		t.Fatalf("assets %v", got)
	}
}
