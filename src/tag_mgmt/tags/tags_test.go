// gitclt
// Tests the tag commands against a fake Forgejo that keeps its tags in memory

package tags

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &TagTarget, &TagMessage} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &WithRelease, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
}

type fakeTag struct {
	name, sha, message string
}

// What the fake forge holds for acme/web (the fake connection's username is the owner); released lists the tags
// that have a release, drafts are the released tags whose release is a draft
type forgeState struct {
	mu       sync.Mutex
	tags     []fakeTag
	released []string
	drafts   []string
	creates  []string // "name@target:message" per create request
	deletes  []string
}

func (s *forgeState) tag(name string) *fakeTag {
	for i := range s.tags {
		if s.tags[i].name == name {
			return &s.tags[i]
		}
	}
	return nil
}

// Like Forgejo, an annotated tag's id is its tag object's, a lightweight tag's is its commit's
func tagJSON(t fakeTag) string {
	id := t.sha
	if t.message != "" {
		id = "obj" + t.sha
	}
	return fmt.Sprintf(`{"id":%q,"name":%q,"message":%q,"commit":{"sha":%q},"tarball_url":"https://forge/acme/web/archive/%s.tar.gz"}`,
		id, t.name, t.message, t.sha, t.name)
}

func tagServer(t *testing.T, s *forgeState) {
	t.Helper()
	const base = "/api/v1/repos/acme/web"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base: testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`),
		"GET " + base + "/tags": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var out []string
			for _, tg := range s.tags {
				out = append(out, tagJSON(tg))
			}
			testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
		},
		"GET " + base + "/tags/{tag}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if tg := s.tag(r.PathValue("tag")); tg != nil {
				testutil.Reply(tagJSON(*tg))(w, r)
				return
			}
			testutil.Status(http.StatusNotFound)(w, r)
		},
		"POST " + base + "/tags": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var body struct {
				Name    string `json:"tag_name"`
				Target  string `json:"target"`
				Message string `json:"message"`
			}
			if err := json.UnmarshalRead(r.Body, &body); err != nil {
				testutil.Status(http.StatusBadRequest)(w, r)
				return
			}
			s.creates = append(s.creates, body.Name+"@"+body.Target+":"+body.Message)
			tg := fakeTag{name: body.Name, sha: "1234567890abcdef", message: body.Message}
			s.tags = append(s.tags, tg)
			w.WriteHeader(http.StatusCreated)
			testutil.Reply(tagJSON(tg))(w, r)
		},
		"DELETE " + base + "/tags/{tag}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			name := r.PathValue("tag")
			if name == "FAIL" {
				testutil.Status(http.StatusForbidden)(w, r)
				return
			}
			s.deletes = append(s.deletes, name)
			s.tags = slices.DeleteFunc(s.tags, func(tg fakeTag) bool { return tg.name == name })
			w.WriteHeader(http.StatusNoContent)
		},
		"DELETE " + base + "/releases/{id}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var i int
			fmt.Sscan(r.PathValue("id"), &i)
			tag := s.released[i-1]
			s.deletes = append(s.deletes, "release "+tag)
			s.released[i-1] = "" // ids are positions: keep them
			w.WriteHeader(http.StatusNoContent)
		},
		"GET " + base + "/releases": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var out []string
			for i, tag := range s.released {
				if tag == "" {
					continue
				}
				out = append(out, fmt.Sprintf(`{"id":%d,"tag_name":%q,"draft":%t,"html_url":"https://forge/acme/web/releases/tag/%s"}`,
					i+1, tag, slices.Contains(s.drafts, tag), tag))
			}
			testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
		},
	})
}

func TestListTags(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []fakeTag{{"v1.1.0", "bbbbbbbbbb", ""}, {"v1.0.0", "aaaaaaaaaa", "Version 1.0.0\n\nfirst"}, {"v1.2.0", "cccccccccc", ""}},
		released: []string{"v1.0.0", "v1.2.0"}, drafts: []string{"v1.2.0"}}
	tagServer(t, s)

	var list []Tag
	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { list, err = ListTags("web", true) }))
	if err != nil || len(list) != 3 {
		t.Fatalf("%v, %+v", err, list)
	}
	// Newest version first; the release and the annotation's first line are shown
	if list[0].Name != "v1.2.0" || list[0].Release != "draft release" || list[1].Release != "" || list[2].Release != "release" ||
		list[2].ReleaseURL != "https://forge/acme/web/releases/tag/v1.0.0" || list[2].Message != "Version 1.0.0" {
		t.Fatalf("list %+v", list)
	}
	for _, want := range []string{"aaaaaaa", "Version 1.0.0", "draft release"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "first") {
		t.Errorf("the annotation's body is shown:\n%s", out)
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, err = ListTags("web", true) })
	if err != nil || !strings.Contains(out, `"releaseUrl"`) || !strings.Contains(out, `"sha"`) {
		t.Fatalf("%v:\n%s", err, out)
	}
}

func TestGetTag(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []fakeTag{{"v1.0.0", "aaaaaaaaaa", "Version 1.0.0"}, {"v1.1.0", "bbbbbbbbbb", ""}}, released: []string{"v1.0.0"}}
	tagServer(t, s)

	var tg *Tag
	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { tg, err = GetTag("web", "v1.0.0", true) }))
	if err != nil || tg.Release != "release" {
		t.Fatalf("%v, %+v", err, tg)
	}
	for _, want := range []string{"aaaaaaaaaa", "annotated", "Version 1.0.0", "release (https://forge/acme/web/releases/tag/v1.0.0)", "v1.0.0.tar.gz"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}

	out = testutil.StripANSI(testutil.Run(t, func() { _, err = GetTag("web", "v1.1.0", true) }))
	if err != nil || !strings.Contains(out, "lightweight") || !strings.Contains(out, "none") {
		t.Fatalf("%v:\n%s", err, out)
	}
	if _, err = GetTag("web", "v9", false); err == nil || err.Title != "No such tag" {
		t.Fatalf("unknown tag: %+v", err)
	}
	if _, err = GetTag("nope", "v1.0.0", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("unknown repo: %+v", err)
	}
}

func TestCreateTag(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []fakeTag{{"v1.0.0", "aaaaaaaaaa", ""}}}
	tagServer(t, s)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateTag("web", "v1.1.0") })
	if err != nil || !slices.Equal(s.creates, []string{"v1.1.0@:"}) || !strings.Contains(out, "Created tag v1.1.0 at 1234567") {
		t.Fatalf("%v, creates %v:\n%s", err, s.creates, out)
	}

	testutil.Set(t, &TagTarget, "develop")
	testutil.Set(t, &TagMessage, "Version 1.2.0")
	testutil.Run(t, func() { err = CreateTag("web", "v1.2.0") })
	if err != nil || s.creates[1] != "v1.2.0@develop:Version 1.2.0" {
		t.Fatalf("%v, creates %v", err, s.creates)
	}

	// An existing tag is refused before the request
	if err = CreateTag("web", "v1.0.0"); err == nil || err.Title != "Tag already exists" || len(s.creates) != 2 {
		t.Fatalf("existing tag: %+v, creates %v", err, s.creates)
	}
}

func TestDeleteTags(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []fakeTag{{"v1", "a", ""}, {"v2", "b", ""}, {"v3", "c", ""}, {"v4", "d", ""}}, released: []string{"v3", "v4"}}
	tagServer(t, s)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteTags("web", []string{"v1"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || len(s.deletes) != 0 {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}

	// Unknown tags, then tags with a release, are all reported before anything is deleted
	if err = DeleteTags("web", []string{"v1", "v9", "v8"}); err == nil || err.Title != "No such tag" ||
		!strings.Contains(err.Message, "v9, v8") || len(s.deletes) != 0 {
		t.Fatalf("unknown tag: %+v, deleted %v", err, s.deletes)
	}
	if err = DeleteTags("web", []string{"v1", "v3", "v4"}); err == nil || err.Title != "Tag has a release" ||
		!strings.Contains(err.Message, "v3, v4; pass --with-release") || len(s.deletes) != 0 {
		t.Fatalf("released tag: %+v, deleted %v", err, s.deletes)
	}

	// A repeated tag is only deleted once
	out = testutil.Run(t, func() { err = DeleteTags("web", []string{"v1", "v2", "v1"}) }, "y")
	if err != nil || !slices.Equal(s.deletes, []string{"v1", "v2"}) || !strings.Contains(out, "Deleted tag v2") {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}
}

// The first failure stops the run
func TestDeleteTags_Failure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	s := &forgeState{tags: []fakeTag{{"v1", "a", ""}, {"FAIL", "b", ""}, {"v3", "c", ""}}}
	tagServer(t, s)

	var err *cerr.CustomError
	testutil.Run(t, func() { err = DeleteTags("web", []string{"v1", "FAIL", "v3"}) })
	if err == nil || err.Code != http.StatusForbidden || !slices.Equal(s.deletes, []string{"v1"}) {
		t.Fatalf("got %+v, deleted %v", err, s.deletes)
	}
}

// --with-release deletes the release, then its tag
func TestDeleteTags_WithRelease(t *testing.T) {
	defaults(t)
	testutil.Set(t, &WithRelease, true)
	s := &forgeState{tags: []fakeTag{{"v1", "a", ""}, {"v2", "b", ""}}, released: []string{"v1"}}
	tagServer(t, s)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteTags("web", []string{"v1", "v2"}) }, "y")
	if err != nil || !slices.Equal(s.deletes, []string{"release v1", "v1", "v2"}) {
		t.Fatalf("%v, deleted %v:\n%s", err, s.deletes, out)
	}
	for _, want := range []string{"and the release(s) of v1 with their assets", "Deleted release v1", "Deleted tag v1", "Deleted tag v2"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"v0.10.0", "v0.9.0", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"v1.02", "v1.2", 0},
		{"v2", "v10", -1},
		{"alpha", "beta", -1},
		{"v1.0.0-rc1", "v1.0.0", 1},
	} {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
