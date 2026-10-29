// gitclt
// Tests release upload against a fake Forgejo that keeps its releases and assets in memory

package releases

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	testutil.Set(t, &OwnerOrg, "")
	for _, b := range []*bool{&CreateRelease, &DraftRelease, &Prerelease, &ReplaceAssets, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
}

type fakeAsset struct {
	id      int64
	name    string
	content string
}

type fakeRelease struct {
	id         int64
	tag        string
	draft, pre bool
	assets     []fakeAsset
}

// What the fake forge holds for acme/web (the fake connection's username is the owner); tags lists the tags it has
type forgeState struct {
	mu       sync.Mutex
	tags     []string
	releases []*fakeRelease
	nextID   int64
	creates  int
	deletes  []string
}

func (s *forgeState) release(id string) *fakeRelease {
	for _, r := range s.releases {
		if fmt.Sprint(r.id) == id {
			return r
		}
	}
	return nil
}

func (s *forgeState) listAssets(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, a := range s.release(r.PathValue("id")).assets {
		out = append(out, fmt.Sprintf(`{"id":%d,"name":%q,"size":%d}`, a.id, a.name, len(a.content)))
	}
	testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
}

func releaseJSON(r *fakeRelease) string {
	return fmt.Sprintf(`{"id":%d,"tag_name":%q,"name":%q,"draft":%t,"prerelease":%t,"html_url":"https://forge/acme/web/releases/tag/%s"}`,
		r.id, r.tag, r.tag, r.draft, r.pre, r.tag)
}

// Like the real servers, GetByTag does not see drafts; the list does
func releaseServer(t *testing.T, s *forgeState) {
	t.Helper()
	s.nextID = 100
	const base = "/api/v1/repos/acme/web"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base: testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`),
		"GET " + base + "/tags/{tag}": func(w http.ResponseWriter, r *http.Request) {
			if !slices.Contains(s.tags, r.PathValue("tag")) {
				testutil.Status(http.StatusNotFound)(w, r)
				return
			}
			testutil.Reply(fmt.Sprintf(`{"name":%q,"commit":{"sha":"abc"}}`, r.PathValue("tag")))(w, r)
		},
		// releases/tags/{tag} and releases/{id}/assets: one pattern, as the mux cannot tell them apart
		"GET " + base + "/releases/{id}/{what}": func(w http.ResponseWriter, r *http.Request) {
			if r.PathValue("id") != "tags" {
				s.listAssets(w, r)
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, rel := range s.releases {
				if rel.tag == r.PathValue("what") && !rel.draft {
					testutil.Reply(releaseJSON(rel))(w, r)
					return
				}
			}
			testutil.Status(http.StatusNotFound)(w, r)
		},
		"GET " + base + "/releases": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var out []string
			for _, rel := range s.releases {
				out = append(out, releaseJSON(rel))
			}
			testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
		},
		"POST " + base + "/releases": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			rel := &fakeRelease{id: s.nextID, draft: strings.Contains(body, `"draft":true`), pre: strings.Contains(body, `"prerelease":true`)}
			for _, tag := range s.tags {
				if strings.Contains(body, fmt.Sprintf(`"tag_name":%q`, tag)) {
					rel.tag = tag
				}
			}
			s.nextID++
			s.creates++
			s.releases = append(s.releases, rel)
			w.WriteHeader(http.StatusCreated)
			testutil.Reply(releaseJSON(rel))(w, r)
		},
		"DELETE " + base + "/releases/{id}/assets/{aid}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			rel := s.release(r.PathValue("id"))
			rel.assets = slices.DeleteFunc(rel.assets, func(a fakeAsset) bool {
				if fmt.Sprint(a.id) == r.PathValue("aid") {
					s.deletes = append(s.deletes, a.name)
					return true
				}
				return false
			})
			w.WriteHeader(http.StatusNoContent)
		},
		"POST " + base + "/releases/{id}/assets": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			f, _, err := r.FormFile("attachment")
			if err != nil {
				t.Errorf("upload: %v", err)
				testutil.Status(http.StatusBadRequest)(w, r)
				return
			}
			content, _ := io.ReadAll(f)
			if strings.Contains(string(content), "FAIL") {
				testutil.Status(http.StatusRequestEntityTooLarge)(w, r)
				return
			}
			rel := s.release(r.PathValue("id"))
			a := fakeAsset{id: s.nextID, name: r.URL.Query().Get("name"), content: string(content)}
			s.nextID++
			rel.assets = append(rel.assets, a)
			w.WriteHeader(http.StatusCreated)
			testutil.Reply(fmt.Sprintf(`{"id":%d,"name":%q,"size":%d}`, a.id, a.name, len(a.content)))(w, r)
		},
	})
}

// Writes the named files (name -> content) in a temporary directory, and returns their paths in the given order
func writeFiles(t *testing.T, files ...string) []string {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	for i := 0; i+1 < len(files); i += 2 {
		p := filepath.Join(dir, files[i])
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(files[i+1]), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

func assetNames(r *fakeRelease) []string {
	var names []string
	for _, a := range r.assets {
		names = append(names, a.name)
	}
	slices.Sort(names)
	return names
}

func upload(t *testing.T, tag string, paths []string) (string, *cerr.CustomError) {
	t.Helper()
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = UploadAssets("web", tag, paths) })
	return out, err
}

func TestUpload_ExistingRelease(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0"}}

	paths := writeFiles(t, "pkg-1.0.0.rpm", "rpm", "deb/pkg_1.0.0.deb", "deb")
	out, err := upload(t, "v1.0.0", paths)
	if err != nil {
		t.Fatal(err.Error())
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"pkg-1.0.0.rpm", "pkg_1.0.0.deb"}) {
		t.Fatalf("assets %v", got)
	}
	if s.creates != 0 || !strings.Contains(out, "Uploaded pkg_1.0.0.deb (3 B)") ||
		!strings.Contains(out, "https://forge/acme/web/releases/tag/v1.0.0") {
		t.Fatalf("creates %d, output:\n%s", s.creates, out)
	}
}

func TestUpload_NoRelease(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	paths := writeFiles(t, "a.rpm", "a")

	// Without --create: refused, nothing created
	if _, err := upload(t, "v1.0.0", paths); err == nil || err.Title != "No such release" {
		t.Fatalf("got %+v", err)
	}
	// --create on a tag the server does not have: refused, the library would otherwise tag the default branch
	testutil.Set(t, &CreateRelease, true)
	if _, err := upload(t, "v2.0.0", paths); err == nil || err.Title != "No such tag" {
		t.Fatalf("got %+v", err)
	}
	if s.creates != 0 {
		t.Fatalf("%d releases created", s.creates)
	}

	out, err := upload(t, "v1.0.0", paths)
	if err != nil || s.creates != 1 || !strings.Contains(out, "Created release v1.0.0") {
		t.Fatalf("%v, creates %d:\n%s", err, s.creates, out)
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"a.rpm"}) {
		t.Fatalf("assets %v", got)
	}
}

// A re-run of --create --draft finds the draft (GetByTag does not), instead of creating a second one
func TestUpload_DraftRerun(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateRelease, true)
	testutil.Set(t, &DraftRelease, true)
	testutil.Set(t, &Prerelease, true)
	s := &forgeState{tags: []string{"v1.0.0-rc1"}}
	releaseServer(t, s)

	out, err := upload(t, "v1.0.0-rc1", writeFiles(t, "a.rpm", "a"))
	if err != nil || !strings.Contains(out, "Created draft prerelease v1.0.0-rc1") {
		t.Fatalf("%v:\n%s", err, out)
	}
	if _, err = upload(t, "v1.0.0-rc1", writeFiles(t, "b.rpm", "b")); err != nil {
		t.Fatal(err.Error())
	}
	if s.creates != 1 || !s.releases[0].draft || !s.releases[0].pre {
		t.Fatalf("creates %d, release %+v", s.creates, s.releases[0])
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"a.rpm", "b.rpm"}) {
		t.Fatalf("assets %v", got)
	}
}

func TestUpload_Replace(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0", assets: []fakeAsset{{id: 7, name: "a.rpm", content: "old"}}}}
	paths := writeFiles(t, "a.rpm", "new", "b.rpm", "b")

	// Without --replace, nothing is uploaded, not even b.rpm
	if _, err := upload(t, "v1.0.0", paths); err == nil || err.Title != "Assets already exist" || !strings.Contains(err.Message, "a.rpm") {
		t.Fatalf("got %+v", err)
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"a.rpm"}) {
		t.Fatalf("assets %v", got)
	}

	testutil.Set(t, &ReplaceAssets, true)
	if _, err := upload(t, "v1.0.0", paths); err != nil {
		t.Fatal(err.Error())
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"a.rpm", "b.rpm"}) || !slices.Equal(s.deletes, []string{"a.rpm"}) {
		t.Fatalf("assets %v, deleted %v", got, s.deletes)
	}
	for _, a := range s.releases[0].assets {
		if a.name == "a.rpm" && a.content != "new" {
			t.Fatalf("a.rpm = %q", a.content)
		}
	}
}

func TestUpload_StopsAtFirstFailure(t *testing.T) {
	defaults(t)
	s := &forgeState{tags: []string{"v1.0.0"}}
	releaseServer(t, s)
	s.releases = []*fakeRelease{{id: 1, tag: "v1.0.0"}}

	_, err := upload(t, "v1.0.0", writeFiles(t, "a.rpm", "a", "b.rpm", "FAIL", "c.rpm", "c"))
	if err == nil || err.Code != http.StatusRequestEntityTooLarge || !strings.Contains(err.Title, "b.rpm") {
		t.Fatalf("got %+v", err)
	}
	if got := assetNames(s.releases[0]); !slices.Equal(got, []string{"a.rpm"}) {
		t.Fatalf("assets %v", got)
	}
}

// Checked before connecting: no registry needed
func TestUpload_LocalChecks(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, "")
	dir := t.TempDir()

	for _, tc := range []struct {
		name  string
		paths []string
		want  string
	}{
		{"missing file", []string{filepath.Join(dir, "nope.rpm")}, "Unable to read"},
		{"directory", []string{dir}, "Not a file"},
		{"same base name", writeFiles(t, "x/a.rpm", "1", "y/a.rpm", "2"), "Duplicate asset name"},
	} {
		if _, err := upload(t, "v1", tc.paths); err == nil || err.Title != tc.want && !strings.HasPrefix(err.Title, tc.want) {
			t.Errorf("%s: got %+v", tc.name, err)
		}
	}

	testutil.Set(t, &DraftRelease, true)
	if _, err := upload(t, "v1", writeFiles(t, "a.rpm", "a")); err == nil || err.Title != "Invalid flags" {
		t.Fatalf("--draft without --create: got %+v", err)
	}
}

// Bitbucket is refused right after connecting (which makes no request there), before any release lookup
func TestUpload_Bitbucket(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"bb":{"url":"https://bitbucket.org","token":"t","owner":"ws","servertype":"bitbucket"}},"default_connection":"bb"}`)
	if _, err := upload(t, "v1", writeFiles(t, "a.rpm", "a")); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("got %+v", err)
	}
}

// The files took the place of a missing <REPO> (or <TAG>): the repository is reported missing, not the release
func TestUpload_NoSuchRepo(t *testing.T) {
	defaults(t)
	releaseServer(t, &forgeState{tags: []string{"v1.0.0"}})
	var err *cerr.CustomError
	testutil.Run(t, func() { err = UploadAssets("pkg_1.0.0.deb", "pkg-1.0.0.rpm", writeFiles(t, "a.apk", "a")) })
	if err == nil || err.Title != "No such repository" || !strings.Contains(err.Message, "acme/pkg_1.0.0.deb does not exist") {
		t.Fatalf("got %+v", err)
	}
}
