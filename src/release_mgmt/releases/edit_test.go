// gitclt
// Tests release edit and release publish against the fake Forgejo of upload_test.go

package releases

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// changedFlags stands in for cmd.Flags().Changed
func changedFlags(names ...string) func(string) bool {
	return func(n string) bool { return slices.Contains(names, n) }
}

func TestEditRelease(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0", notes: "old notes", pre: true}}

	// Only the flags passed are sent: the notes and the draft flag stay as they were
	testutil.Set(t, &EditTitle, "Version 1.0.0")
	testutil.Set(t, &EditPrerelease, false)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditRelease("web", "v1.0.0", changedFlags("title", "prerelease")) })
	if err != nil || !slices.Equal(s.edits, []string{"v1.0.0:name,prerelease"}) || !strings.Contains(out, "Updated release v1.0.0") {
		t.Fatalf("%v, edits %v:\n%s", err, s.edits, out)
	}
	if r := s.releases[0]; r.title != "Version 1.0.0" || r.notes != "old notes" || r.pre {
		t.Fatalf("release %+v", r)
	}

	// The notes come from a file; an empty --notes clears them
	notes := filepath.Join(t.TempDir(), "notes.md")
	if e := os.WriteFile(notes, []byte("# 1.0.0\n- first\n"), 0600); e != nil {
		t.Fatal(e)
	}
	testutil.Set(t, &EditNotesFile, notes)
	if err = EditRelease("web", "v1.0.0", changedFlags("notes-file")); err != nil || s.releases[0].notes != "# 1.0.0\n- first\n" {
		t.Fatalf("%v, notes %q", err, s.releases[0].notes)
	}
	if err = EditRelease("web", "v1.0.0", changedFlags("notes")); err != nil || s.releases[0].notes != "" {
		t.Fatalf("%v, notes %q", err, s.releases[0].notes)
	}

	if err = EditRelease("web", "v2.0.0", changedFlags("title")); err == nil || err.Title != "No such release" {
		t.Fatalf("no release: %+v", err)
	}
	if err = EditRelease("web", "v1.0.0", changedFlags()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}
}

// A draft is found (GetByTag does not see it), and --draft=false publishes it, provided its tag is on the server
func TestEditRelease_Draft(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0", draft: true}, {id: 2, tag: "v2.0.0", draft: true}}

	var err *cerr.CustomError
	if err = EditRelease("web", "v2.0.0", changedFlags("draft")); err == nil || err.Title != "No such tag" || len(s.edits) != 0 {
		t.Fatalf("missing tag: %+v, edits %v", err, s.edits)
	}
	// Edits that do not publish need no tag
	testutil.Set(t, &EditTitle, "next")
	testutil.Run(t, func() { err = EditRelease("web", "v2.0.0", changedFlags("title")) })
	if err != nil || !s.releases[1].draft || s.releases[1].title != "next" {
		t.Fatalf("%v, release %+v", err, s.releases[1])
	}

	out := testutil.Run(t, func() { err = EditRelease("web", "v1.0.0", changedFlags("draft")) })
	if err != nil || s.releases[0].draft || !strings.Contains(out, "Updated release v1.0.0") {
		t.Fatalf("%v, release %+v:\n%s", err, s.releases[0], out)
	}
}

func TestPublishReleases(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0", "v1.1.0-rc1", "v1.1.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0"}, {id: 2, tag: "v1.1.0-rc1", draft: true, pre: true},
		{id: 3, tag: "v1.1.0", draft: true}, {id: 4, tag: "v2.0.0", draft: true}}

	// Every tag is checked first: one without a release, or not on the server, stops everything
	var err *cerr.CustomError
	testutil.Run(t, func() { err = PublishReleases("web", []string{"v1.1.0", "v9", "v8"}) })
	if err == nil || err.Title != "No such release" || !strings.Contains(err.Message, "v9, v8") || len(s.edits) != 0 {
		t.Fatalf("unknown tag: %+v, edits %v", err, s.edits)
	}
	if err = PublishReleases("web", []string{"v1.1.0", "v2.0.0"}); err == nil || err.Title != "No such tag" ||
		!strings.Contains(err.Message, "git push origin v2.0.0") || len(s.edits) != 0 {
		t.Fatalf("missing tag: %+v, edits %v", err, s.edits)
	}

	// The release already published is skipped; a repeated tag is only published once; only draft is sent
	out := testutil.Run(t, func() { err = PublishReleases("web", []string{"v1.0.0", "v1.1.0-rc1", "v1.1.0", "v1.1.0"}) })
	if err != nil || !slices.Equal(s.edits, []string{"v1.1.0-rc1:draft", "v1.1.0:draft"}) {
		t.Fatalf("%v, edits %v:\n%s", err, s.edits, out)
	}
	for _, want := range []string{"Release v1.0.0 is already published", "Published prerelease v1.1.0-rc1", "Published release v1.1.0",
		"https://forge/acme/web/releases/tag/v1.1.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if s.releases[1].draft || s.releases[2].draft || !s.releases[1].pre {
		t.Fatalf("releases %+v %+v", s.releases[1], s.releases[2])
	}
}

// The first failure stops the run
func TestPublishReleases_Failure(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1", "FAIL", "v3"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1", draft: true}, {id: 2, tag: "FAIL", draft: true}, {id: 3, tag: "v3", draft: true}}

	var err *cerr.CustomError
	testutil.Run(t, func() { err = PublishReleases("web", []string{"v1", "FAIL", "v3"}) })
	if err == nil || err.Code != http.StatusForbidden || !slices.Equal(s.edits, []string{"v1:draft"}) || !s.releases[2].draft {
		t.Fatalf("got %+v, edits %v", err, s.edits)
	}
}
