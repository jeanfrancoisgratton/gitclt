// gitclt
// Tests the issue commands against a fake Forgejo that behaves like the real one where that matters:
// pull requests answer as issues, labels and milestones can share names, and an unknown label filter lists everything

package issues

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const (
	issueTwo   = `{"number":2,"title":"First","body":"Body text","state":"open","user":{"login":"acme"},"labels":[{"name":"bug"}],"assignees":[{"login":"acme"}],"milestone":{"id":1,"title":"v1.5"},"comments":1,"html_url":"http://x/acme/web/issues/2"}`
	issueThree = `{"number":3,"title":"Second","state":"closed","user":{"login":"acme"},"labels":[],"comments":0}`
	prAsIssue  = `{"number":1,"title":"a PR","state":"open","user":{"login":"acme"},"labels":[],"comments":0}`
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &IssueTitle, &IssueBody, &IssueBodyFile, &IssueMilestone, &LockReason, &LabelName, &LabelColor,
		&LabelDescription, &MilestoneTitle, &MilestoneDescription, &MilestoneDue, &MilestoneState, &shared.ConnectionName} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &ListState, "open")
	testutil.Set(t, &MilestoneListState, "open")
	for _, l := range []*[]string{&ListLabels, &IssueLabels, &IssueAddLabels, &IssueRemoveLabels, &IssueAssignees} {
		testutil.Set(t, l, nil)
	}
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &WithComments, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
}

func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

// A fake server for acme/web: #1 is a pull request, #2 an open issue, #3 a closed one; what it changes is recorded
type fake struct {
	calls  []string
	bodies []map[string]any
}

func server(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	record := func(status int, reply string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/api/v1/repos/acme/web"))
			if r.Method == http.MethodDelete {
				w.WriteHeader(status)
				return
			}
			testutil.Record(t, &f.bodies, status, reply)(w, r)
		}
	}
	base := "/api/v1/repos/acme/web"
	label := func(name string) string { return `{"id":1,"name":"` + name + `","color":"d73a4a"}` }
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base:                            testutil.Reply(`{"name":"web","full_name":"acme/web"}`),
		"GET " + base + "/issues":                testutil.Reply(`[` + issueTwo + `,` + issueThree + `]`),
		"POST " + base + "/issues":               record(http.StatusCreated, issueTwo),
		"GET " + base + "/issues/1":              testutil.Reply(prAsIssue),
		"GET " + base + "/pulls/1":               testutil.Reply(`{"number":1,"title":"a PR","state":"open"}`),
		"GET " + base + "/issues/2":              testutil.Reply(issueTwo),
		"GET " + base + "/issues/3":              testutil.Reply(issueThree),
		"PATCH " + base + "/issues/2":            record(http.StatusOK, issueTwo),
		"PUT " + base + "/issues/2/labels":       testutil.Reply(`[]`),
		"PATCH " + base + "/issues/3":            record(http.StatusOK, issueThree),
		"DELETE " + base + "/issues/2":           record(http.StatusNoContent, ``),
		"GET " + base + "/issues/2/comments":     testutil.Reply(`[{"id":16,"user":{"login":"acme"},"body":"Hello"}]`),
		"POST " + base + "/issues/2/comments":    record(http.StatusCreated, `{"id":17,"user":{"login":"acme"},"body":"x"}`),
		"PATCH " + base + "/issues/comments/16":  record(http.StatusOK, `{"id":16,"body":"y"}`),
		"DELETE " + base + "/issues/comments/16": record(http.StatusNoContent, ``),
		"GET " + base + "/labels":                testutil.Reply(`[` + label("bug") + `,{"id":2,"name":"twin","color":"000000"},{"id":3,"name":"twin","color":"ffffff"}]`),
		"POST " + base + "/labels":               record(http.StatusCreated, label("docs")),
		"PATCH " + base + "/labels/1":            record(http.StatusOK, label("bug")),
		"DELETE " + base + "/labels/1":           record(http.StatusNoContent, ``),
		"GET " + base + "/milestones":            testutil.Reply(`[{"id":1,"title":"v1.5","state":"open"},{"id":2,"title":"2026","state":"open"},{"id":3,"title":"dup","state":"open"},{"id":4,"title":"dup","state":"closed"}]`),
		"POST " + base + "/milestones":           record(http.StatusCreated, `{"id":9,"title":"new","state":"open"}`),
		"PATCH " + base + "/milestones/1":        record(http.StatusOK, `{"id":1,"title":"v1.5","state":"closed"}`),
		"DELETE " + base + "/milestones/1":       record(http.StatusNoContent, ``),
	})
	return f
}

func TestList(t *testing.T) {
	defaults(t)
	server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListIssues("web", true) })
	for _, want := range []string{"First", "Second", "bug", "v1.5"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("missing %q: %v\n%s", want, err, out)
		}
	}
	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { _, err = ListIssues("web", true) }); err != nil || out != "2\n3\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	testutil.Set(t, &ListState, "nonsense")
	if _, err = ListIssues("web", false); err == nil || err.Title != "Invalid state" {
		t.Fatalf("state: %+v", err)
	}
}

// Forgejo ignores a label it does not know, and lists everything
func TestList_LabelFilterIsEnforced(t *testing.T) {
	defaults(t)
	server(t)
	testutil.Set(t, &ListLabels, []string{"BUG"})
	if l, err := ListIssues("web", false); err != nil || len(l) != 1 || l[0].Number != 2 {
		t.Fatalf("known label: %+v, %v", l, err)
	}
	testutil.Set(t, &ListLabels, []string{"nosuchlabel"})
	if l, err := ListIssues("web", false); err != nil || len(l) != 0 {
		t.Fatalf("unknown label: %+v, %v", l, err)
	}
}

func TestGet(t *testing.T) {
	defaults(t)
	server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = GetIssue("web", "#2", true) })
	if err != nil || !strings.Contains(out, "Body text") || !strings.Contains(out, "v1.5") || strings.Contains(out, "Hello") {
		t.Fatalf("%v\n%s", err, out)
	}
	testutil.Set(t, &WithComments, true)
	if out = testutil.Run(t, func() { _, err = GetIssue("web", "2", true) }); err != nil || !strings.Contains(out, "Hello") {
		t.Fatalf("comments: %v\n%s", err, out)
	}
	testutil.Set(t, &JSONoutput, true)
	if out = testutil.Run(t, func() { _, err = GetIssue("web", "2", true) }); err != nil || !strings.Contains(out, `"discussion"`) || !strings.Contains(out, `"comments": 1`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if _, err = GetIssue("web", "99", false); err == nil || err.Title != "No such issue" {
		t.Fatalf("unknown: %+v", err)
	}
	if _, err = GetIssue("web", "x", false); err == nil || err.Title != "Invalid issue number" {
		t.Fatalf("not a number: %+v", err)
	}
	if _, err = GetIssue("nope", "2", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("no repo: %+v", err)
	}
}

// A pull request answers as an issue: nothing may close or delete one by mistake
func TestPullRequestNumbersAreRefused(t *testing.T) {
	defaults(t)
	f := server(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &LockReason, "spam")
	_, getErr := GetIssue("web", "1", false)
	for name, err := range map[string]*cerr.CustomError{
		"get":    getErr,
		"close":  CloseIssues("web", []string{"2", "1"}, false),
		"reopen": CloseIssues("web", []string{"1"}, true),
		"delete": DeleteIssues("web", []string{"2", "1"}),
		"lock":   LockIssue("web", "1", false),
	} {
		if err == nil || err.Title != "Not an issue" {
			t.Errorf("%s: %+v", name, err)
		}
	}
	// Everything is checked first, so #2 was left alone as well
	if len(f.calls) != 0 {
		t.Fatalf("changed: %v", f.calls)
	}
}

func TestCreate(t *testing.T) {
	defaults(t)
	f := server(t)
	path := filepath.Join(t.TempDir(), "body")
	if e := os.WriteFile(path, []byte("From a file"), 0600); e != nil {
		t.Fatal(e)
	}
	testutil.Set(t, &IssueTitle, "First")
	testutil.Set(t, &IssueBodyFile, path)
	testutil.Set(t, &IssueAssignees, []string{"acme"})
	testutil.Set(t, &IssueMilestone, "v1.5")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateIssue("web", passed("body-file")) })
	if err != nil || !strings.Contains(out, "Created issue #2 in acme/web") {
		t.Fatalf("%v\n%s", err, out)
	}
	b := f.bodies[0]
	if b["title"] != "First" || b["body"] != "From a file" || b["milestone"] != float64(1) || fmt.Sprint(b["assignees"]) != "[acme]" {
		t.Fatalf("sent %v", b)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	if out = testutil.Run(t, func() { err = CreateIssue("web", passed("body-file")) }); err != nil || out != "2\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}

	f.calls = nil
	testutil.Set(t, &IssueMilestone, "nosuch")
	if err = CreateIssue("web", passed()); err == nil || err.Title != "No such milestone" || len(f.calls) != 0 {
		t.Fatalf("milestone: %+v, %v", err, f.calls)
	}
	testutil.Set(t, &IssueTitle, "  ")
	if err = CreateIssue("web", passed()); err == nil || err.Title != "Missing title" {
		t.Fatalf("title: %+v", err)
	}
}

func TestEdit(t *testing.T) {
	defaults(t)
	f := server(t)
	if err := EditIssue("web", "2", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}

	// Only what was passed; an empty list clears, and an empty milestone removes it
	testutil.Set(t, &IssueTitle, "Renamed")
	var err *cerr.CustomError
	testutil.Run(t, func() { err = EditIssue("web", "2", passed("title", "label", "milestone")) })
	if err != nil || len(f.bodies) == 0 {
		t.Fatalf("%v, %v", err, f.bodies)
	}
	b := f.bodies[len(f.bodies)-1]
	if _, sent := b["body"]; sent || b["title"] != "Renamed" {
		t.Fatalf("sent %v", b)
	}

	// edit accepts a pull request's number: this is how one gets a milestone
	f.calls = nil
	if err = EditIssue("web", "99", passed("title")); err == nil || err.Title != "No such issue" || len(f.calls) != 0 {
		t.Fatalf("unknown: %+v, %v", err, f.calls)
	}
}

func TestClose(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	if err = CloseIssues("web", []string{"2", "99"}, false); err == nil || err.Title != "No such issue" || len(f.calls) != 0 {
		t.Fatalf("unknown: %+v, %v", err, f.calls)
	}
	out := testutil.Run(t, func() { err = CloseIssues("web", []string{"2", "3", "#2"}, false) })
	if err != nil || !strings.Contains(out, "Closed issue #2") || !strings.Contains(out, "#3 is already closed") || !slices.Equal(f.calls, []string{"PATCH /issues/2"}) {
		t.Fatalf("%v, %v\n%s", err, f.calls, out)
	}
	if f.bodies[0]["state"] != "closed" {
		t.Fatalf("sent %v", f.bodies[0])
	}
	f.calls = nil
	testutil.Run(t, func() { err = CloseIssues("web", []string{"3"}, true) })
	if err != nil || !slices.Equal(f.calls, []string{"PATCH /issues/3"}) {
		t.Fatalf("reopen: %v, %v", err, f.calls)
	}
}

func TestDelete(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = DeleteIssues("web", []string{"2"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || !strings.Contains(out, "First") || !strings.Contains(out, "acme/web") || len(f.calls) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, f.calls, out)
	}
	testutil.Run(t, func() { err = DeleteIssues("web", []string{"2"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE /issues/2"}) {
		t.Fatalf("yes: %v, %v", err, f.calls)
	}
}

func TestLock_ReasonOnForgejo(t *testing.T) {
	defaults(t)
	server(t)
	if err := LockIssue("web", "2", false); err == nil || err.Title != "Missing reason" {
		t.Fatalf("got %+v", err)
	}
}

func TestComments(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListComments("web", "2", true) })
	if err != nil || !strings.Contains(out, "Comment 16") || !strings.Contains(out, "Hello") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if err = AddComment("web", "2", passed()); err == nil || err.Title != "Missing text" {
		t.Fatalf("no text: %+v", err)
	}
	testutil.Set(t, &IssueBody, "Hi")
	out = testutil.Run(t, func() { err = AddComment("web", "2", passed("body")) })
	if err != nil || !strings.Contains(out, "Added comment 17") || f.bodies[0]["body"] != "Hi" {
		t.Fatalf("add: %v, %v\n%s", err, f.bodies, out)
	}
	if testutil.Run(t, func() { err = EditComment("web", "16", passed("body")) }); err != nil {
		t.Fatalf("edit: %v", err)
	}
	out = testutil.Run(t, func() { err = DeleteComments("web", []string{"16"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") {
		t.Fatalf("no: %v\n%s", err, out)
	}
	f.calls = nil
	testutil.Run(t, func() { err = DeleteComments("web", []string{"16"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE /issues/comments/16"}) {
		t.Fatalf("yes: %v, %v", err, f.calls)
	}
}

// GitHub keeps a due date as midnight UTC: it is that day, whatever the time zone
func TestDay(t *testing.T) {
	west := time.FixedZone("west", -5*3600)
	testutil.Set(t, &time.Local, west)
	if got := day(time.Date(2030, 6, 15, 0, 0, 0, 0, time.UTC)); got != "2030.06.15" {
		t.Errorf("midnight UTC: %s", got)
	}
	if got := day(time.Date(2030, 6, 15, 23, 59, 59, 0, west)); got != "2030.06.15" {
		t.Errorf("end of a local day: %s", got)
	}
	if got := day(time.Time{}); got != "" {
		t.Errorf("none: %s", got)
	}
}

// GitHub creates a label that does not exist, without a word; a typo must not leave one behind
func TestGitHub_UnknownLabels(t *testing.T) {
	defaults(t)
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v3")
		switch {
		case r.Method == http.MethodGet && path == "/repos/acme/web":
			fmt.Fprint(w, `{"name":"web"}`)
		case r.Method == http.MethodGet && path == "/repos/acme/web/labels":
			fmt.Fprint(w, `[{"id":1,"name":"Bug","color":"d73a4a"}]`)
		case r.Method == http.MethodGet && path == "/repos/acme/web/issues/2":
			fmt.Fprint(w, `{"number":2,"title":"First","state":"open"}`)
		case r.Method == http.MethodGet && path == "/repos/acme/web/issues/9":
			w.WriteHeader(http.StatusGone)
			fmt.Fprint(w, `{"message":"This issue was deleted"}`)
		case r.Method == http.MethodPost && path == "/repos/acme/web/issues":
			calls = append(calls, "create")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"number":3,"title":"x","state":"open"}`)
		case r.Method == http.MethodPost && path == "/repos/acme/web/issues/2/labels":
			calls = append(calls, "add")
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	testutil.WriteRegistry(t, fmt.Sprintf(`{"auths":{"gh":{"url":%q,"token":"abc","username":"acme","servertype":"github"}},"default_connection":"gh"}`, srv.URL))

	testutil.Set(t, &IssueTitle, "x")
	testutil.Set(t, &IssueLabels, []string{"typo"})
	if err := CreateIssue("web", passed()); err == nil || err.Title != "No such label" {
		t.Errorf("create: %+v", err)
	}
	testutil.Set(t, &IssueLabels, nil)
	testutil.Set(t, &IssueAddLabels, []string{"typo"})
	if err := EditIssue("web", "2", passed("add-label")); err == nil || err.Title != "No such label" {
		t.Errorf("edit: %+v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("sent: %v", calls)
	}

	// The label's case does not matter there
	testutil.Set(t, &IssueAddLabels, []string{"bug"})
	var err *cerr.CustomError
	if testutil.Run(t, func() { err = EditIssue("web", "2", passed("add-label")) }); err != nil || !slices.Equal(calls, []string{"add"}) {
		t.Errorf("known label: %v, %v", err, calls)
	}

	// A deleted issue answers 410
	if _, err = GetIssue("web", "9", false); err == nil || err.Title != "No such issue" {
		t.Errorf("deleted: %+v", err)
	}
}

func TestParseColor(t *testing.T) {
	for in, want := range map[string]string{"#D73A4A": "d73a4a", "0075ca": "0075ca"} {
		if got, err := parseColor(in); err != nil || got != want {
			t.Errorf("%s: %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"red", "", "#12345", "1234567", "gggggg"} {
		if _, err := parseColor(in); err == nil {
			t.Errorf("%q: expected a refusal", in)
		}
	}
}

func TestLabels(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListLabelsOf("web", true) })
	if err != nil || !strings.Contains(out, "#d73a4a") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if out = testutil.Run(t, func() { _, err = GetLabel("web", "bug", true) }); err != nil || !strings.Contains(out, "Color:") {
		t.Fatalf("get: %v\n%s", err, out)
	}
	if _, err = GetLabel("web", "nosuch", false); err == nil || err.Title != "No such label" {
		t.Fatalf("unknown: %+v", err)
	}

	testutil.Set(t, &LabelColor, "#0075CA")
	testutil.Run(t, func() { err = CreateLabel("web", "docs") })
	if err != nil || !strings.HasSuffix(fmt.Sprint(f.bodies[0]["color"]), "0075ca") || f.bodies[0]["name"] != "docs" {
		t.Fatalf("create: %v, %v", err, f.bodies)
	}
	if err = EditLabel("web", "bug", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}
	testutil.Set(t, &LabelDescription, "")
	f.calls = nil
	testutil.Run(t, func() { err = EditLabel("web", "bug", passed("description")) })
	if err != nil || !slices.Equal(f.calls, []string{"PATCH /labels/1"}) {
		t.Fatalf("edit: %v, %v", err, f.calls)
	}
	f.calls = nil
	testutil.Run(t, func() { err = DeleteLabels("web", []string{"bug", "bug"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE /labels/1"}) {
		t.Fatalf("delete: %v, %v", err, f.calls)
	}
}

// Forgejo accepts two labels of the same name, and addresses them by name: which one would be hit is anyone's guess
func TestLabels_Duplicates(t *testing.T) {
	defaults(t)
	f := server(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &LabelColor, "000000")
	testutil.Set(t, &LabelName, "bug")
	_, getErr := GetLabel("web", "twin", false)
	for name, tc := range map[string]struct {
		err  *cerr.CustomError
		want string
	}{
		"get":    {getErr, "Ambiguous label"},
		"edit":   {EditLabel("web", "twin", passed("color")), "Ambiguous label"},
		"delete": {DeleteLabels("web", []string{"twin"}), "Ambiguous label"},
		"create": {CreateLabel("web", "bug"), "Label already exists"},
	} {
		if tc.err == nil || tc.err.Title != tc.want {
			t.Errorf("%s: %+v", name, tc.err)
		}
	}
	testutil.Set(t, &LabelName, "twin")
	if err := EditLabel("web", "bug", passed("name")); err == nil || err.Title != "Label already exists" {
		t.Errorf("rename onto a taken name: %+v", err)
	}
	for _, call := range f.calls {
		if strings.Contains(call, "/labels/2") || strings.Contains(call, "/labels/3") || strings.HasPrefix(call, "POST") {
			t.Fatalf("touched a duplicate: %v", f.calls)
		}
	}
}

func TestMilestones(t *testing.T) {
	defaults(t)
	f := server(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListMilestones("web", true) })
	if err != nil || !strings.Contains(out, "v1.5") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	// By title or by ID; a title that looks like an ID is a title
	for arg, id := range map[string]int64{"v1.5": 1, "1": 1, "2026": 2, "2": 2, "4": 4} {
		if m, err := GetMilestone("web", arg, false); err != nil || m.ID != id {
			t.Errorf("%s: %+v, %+v", arg, m, err)
		}
	}
	if _, err = GetMilestone("web", "nosuch", false); err == nil || err.Title != "No such milestone" {
		t.Fatalf("unknown: %+v", err)
	}

	testutil.Set(t, &MilestoneDue, "2026-12-31")
	testutil.Set(t, &MilestoneDescription, "first")
	testutil.Run(t, func() { err = CreateMilestone("web", "new") })
	if err != nil || f.bodies[0]["title"] != "new" || !strings.HasPrefix(fmt.Sprint(f.bodies[0]["due_on"]), "2026-12-31T23:59:59") {
		t.Fatalf("create: %v, %v", err, f.bodies)
	}
	testutil.Set(t, &MilestoneDue, "soon")
	if err = CreateMilestone("web", "new2"); err == nil || err.Title != "Invalid date" {
		t.Fatalf("date: %+v", err)
	}

	if err = EditMilestone("web", "v1.5", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}
	testutil.Set(t, &MilestoneDue, "")
	if err = EditMilestone("web", "v1.5", passed("due")); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("no due date: %+v", err)
	}
	testutil.Set(t, &MilestoneState, "all")
	if err = EditMilestone("web", "v1.5", passed("state")); err == nil || err.Title != "Invalid state" {
		t.Fatalf("state: %+v", err)
	}
	testutil.Set(t, &MilestoneState, "closed")
	f.calls = nil
	testutil.Run(t, func() { err = EditMilestone("web", "v1.5", passed("state")) })
	if err != nil || !slices.Equal(f.calls, []string{"PATCH /milestones/1"}) {
		t.Fatalf("close: %v, %v", err, f.calls)
	}
	f.calls = nil
	testutil.Run(t, func() { err = DeleteMilestones("web", []string{"v1.5"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE /milestones/1"}) {
		t.Fatalf("delete: %v, %v", err, f.calls)
	}
}

func TestMilestones_Duplicates(t *testing.T) {
	defaults(t)
	f := server(t)
	testutil.Set(t, &AssumeYes, true)
	if _, err := GetMilestone("web", "dup", false); err == nil || err.Title != "Ambiguous milestone" || !strings.Contains(err.Error(), "(3, 4)") {
		t.Fatalf("get: %+v", err)
	}
	if err := DeleteMilestones("web", []string{"dup"}); err == nil || err.Title != "Ambiguous milestone" {
		t.Fatalf("delete: %+v", err)
	}
	if err := CreateMilestone("web", "v1.5"); err == nil || err.Title != "Milestone already exists" {
		t.Fatalf("create: %+v", err)
	}
	testutil.Set(t, &MilestoneTitle, "dup")
	if err := EditMilestone("web", "v1.5", passed("title")); err == nil || err.Title != "Milestone already exists" {
		t.Fatalf("rename: %+v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("changed: %v", f.calls)
	}
}

// Bitbucket retired its issue tracker: a clean refusal, with no request sent
func TestBitbucket(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"test":{"url":"http://127.0.0.1:1","token":"abc","username":"acme","owner":"acme","servertype":"bitbucket"}},"default_connection":"test"}`)
	if _, err := ListIssues("web", false); err == nil || err.Title != "Unsupported operation" || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("got %+v", err)
	}
}
