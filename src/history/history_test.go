// gitclt
// Tests the history commands against a fake Forgejo serving a fixed history

package history

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &ListRef, &ListPath, &ListSince, &ListUntil} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &ListLimit, 30)
	testutil.Set(t, &JSONoutput, false)
	testutil.Set(t, &shared.QuietOutput, false)
}

func commitJSON(sha, msg string, parents ...string) string {
	ps := make([]string, len(parents))
	for i, p := range parents {
		ps[i] = fmt.Sprintf(`{"sha":%q}`, p)
	}
	return fmt.Sprintf(`{"sha":%q,"html_url":"https://forge/acme/web/commit/%s","commit":{"message":%q,`+
		`"author":{"name":"Alice","email":"alice@x","date":"2026-10-30T12:00:00Z"},`+
		`"committer":{"name":"Bob","email":"bob@x","date":"2026-10-30T13:00:00Z"}},`+
		`"author":{"login":"alice"},"parents":[%s]}`, sha, sha, msg, strings.Join(ps, ","))
}

// historyServer serves acme/web: three commits on develop, the newest a merge; refs records the ?sha= asked for
func historyServer(t *testing.T, refs *[]string) {
	t.Helper()
	const base = "/api/v1/repos/acme/web"
	list := "[" + commitJSON("c3c3c3c3c3", "Merge feature\n\nbody", "b2b2b2b2b2", "a1a1a1a1a1") + "," +
		commitJSON("b2b2b2b2b2", "Second", "a1a1a1a1a1") + "," + commitJSON("a1a1a1a1a1", "First") + "]"
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base: testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","default_branch":"develop"}`),
		"GET " + base + "/commits": func(w http.ResponseWriter, r *http.Request) {
			*refs = append(*refs, r.URL.Query().Get("sha"))
			if r.URL.Query().Get("sha") == "nope" {
				http.NotFound(w, r)
				return
			}
			testutil.Reply(list)(w, r)
		},
		"GET " + base + "/git/commits/{ref}": func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix("b2b2b2b2b2", r.PathValue("ref")) {
				http.NotFound(w, r)
				return
			}
			c := strings.TrimSuffix(commitJSON("b2b2b2b2b2", "Second\n\nThe details", "a1a1a1a1a1"), "}") +
				`,"stats":{"additions":7,"deletions":2},"files":[{"filename":"go.mod","status":"modified"},{"filename":"history/get.go","status":"added"}]}`
			testutil.Reply(c)(w, r)
		},
	})
}

func TestList(t *testing.T) {
	defaults(t)
	var refs []string
	historyServer(t, &refs)

	list, err := List("web", false)
	if err != nil || len(list) != 3 || list[0].Author.Login != "alice" || len(list[0].Parents) != 2 {
		t.Fatalf("%v, %+v", err, list)
	}

	testutil.Set(t, &ListLimit, 2)
	testutil.Set(t, &ListRef, "develop")
	out := testutil.StripANSI(testutil.Run(t, func() { list, err = List("web", true) }))
	if err != nil || len(list) != 2 || refs[len(refs)-1] != "develop" {
		t.Fatalf("%v, %+v, %v", err, list, refs)
	}
	for _, want := range []string{"M c3c3c3c", "Merge feature", "Second", "Alice"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "First") || strings.Contains(out, "body") {
		t.Errorf("past the limit, or more than the title:\n%s", out)
	}

	testutil.Set(t, &ListRef, "nope")
	if _, err = List("web", false); err == nil || err.Title != "Unknown reference" {
		t.Fatalf("unknown ref: %+v", err)
	}
	testutil.Set(t, &ListRef, "")
	if _, err = List("nosuchrepo", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("unknown repo: %+v", err)
	}
	testutil.Set(t, &ListLimit, -1)
	if _, err = List("web", false); err == nil || err.Title != "Invalid limit" {
		t.Fatalf("bad limit: %+v", err)
	}
	testutil.Set(t, &ListLimit, 0)
	testutil.Set(t, &ListSince, "yesterday")
	if _, err = List("web", false); err == nil || err.Title != "Invalid date" {
		t.Fatalf("bad date: %+v", err)
	}
	// Forgejo cannot filter by date
	testutil.Set(t, &ListSince, "2026-10-01")
	if _, err = List("web", false); err == nil {
		t.Fatal("Forgejo filtered by date")
	}
}

func TestGet(t *testing.T) {
	defaults(t)
	var refs []string
	historyServer(t, &refs)

	var err *cerr.CustomError
	out := testutil.StripANSI(testutil.Run(t, func() { _, err = Get("web", "b2b2b2b", true) }))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"b2b2b2b2b2", "Alice <alice@x> (alice)", "Bob <bob@x>", "a1a1a1a", "2 file(s), +7 -2",
		"    The details", "added", "history/get.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, err = Get("web", "b2b2b2b", true) })
	if err != nil || !strings.Contains(out, `"additions": 7`) && !strings.Contains(out, `"additions":7`) {
		t.Fatalf("%v:\n%s", err, out)
	}

	if _, err = Get("web", "zzz", false); err == nil || err.Title != "No such commit" {
		t.Fatalf("unknown: %+v", err)
	}
}

func TestParseDate(t *testing.T) {
	until, err := parseDate("2026-10-30", true)
	if err != nil || until.Day() != 30 || until.Hour() != 23 {
		t.Fatalf("%v, %v", err, until)
	}
	ts, err := parseDate("2026-10-30T10:00:00Z", false)
	if err != nil || !ts.Equal(time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("%v, %v", err, ts)
	}
}

// GitHub-like changes: per-file line counts get their columns, a rename shows both paths; an identical committer is not repeated
func TestRenderCommit(t *testing.T) {
	when := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	me := gol.CommitAuthor{Name: "Alice", Email: "alice@x", Login: "Alice", Date: when}
	out := testutil.StripANSI(testutil.Run(t, func() {
		renderCommit(gol.Commit{SHA: "abc", Message: "Rename\n", Author: me, Committer: me,
			Stats: &gol.CommitStats{Additions: 3, Deletions: 1},
			Files: []gol.ChangedFile{{Path: "new.go", PreviousPath: "old.go", Status: gol.FileRenamed, Additions: 3, Deletions: 1}}})
	}))
	for _, want := range []string{"Alice <alice@x>\n", "old.go → new.go", "renamed", " + ", " 3 "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Committer:", "Parents:", "(Alice)"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q:\n%s", unwanted, out)
		}
	}

	// Forgejo: all per-file counts are 0, so no count columns
	out = testutil.StripANSI(testutil.Run(t, func() {
		renderFiles([]gol.ChangedFile{{Path: "a.go", Status: gol.FileAdded}})
	}))
	if strings.Contains(out, " + ") {
		t.Errorf("count columns on Forgejo:\n%s", out)
	}
}

func TestListJSONAndEmpty(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web":         testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web"}`),
		"GET /api/v1/repos/acme/web/commits": testutil.Reply(`[]`),
	})
	testutil.Set(t, &ListPath, "nothing/here")
	out := testutil.Run(t, func() { _, _ = List("web", true) })
	if !strings.Contains(out, "No commit of acme/web matches") {
		t.Errorf("empty list:\n%s", out)
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, _ = List("web", true) }); out != "" {
		t.Errorf("quiet:\n%s", out)
	}
	testutil.Set(t, &JSONoutput, true)
	if out = strings.TrimSpace(testutil.Run(t, func() { _, _ = List("web", true) })); out != "[]" {
		t.Errorf("JSON of an empty list = %q", out)
	}
}
