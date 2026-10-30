// gitclt
// Tests the pull request commands against a fake Forgejo that keeps its pull requests in memory

package pullrequests

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &PRTitle, &PRBody, &PRBodyFile, &PRBase, &MergeTitle, &MergeMessage, &ReviewBody} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &ListState, "open")
	testutil.Set(t, &MergeMethod, "merge")
	testutil.Set(t, &PRReviewers, nil)
	for _, b := range []*bool{&JSONoutput, &PRDraft, &DeleteBranch, &AutoMerge, &CancelAutoMerge, &ReviewApprove,
		&ReviewRequestChanges, &ReviewComment, &RemoveReviewers, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
}

// none stands in for cmd.Flags().Changed when no flag is passed
func none(string) bool { return false }

func changedFlags(names ...string) func(string) bool {
	return func(n string) bool { return slices.Contains(names, n) }
}

type fakePR struct {
	number              int
	title, body, state  string
	head, base, headSHA string
	merged, conflicts   bool
}

// What the fake forge holds for acme/web (the fake connection's username is the owner)
// Like Forgejo, a draft is a title with a WIP prefix; calls records the writes ("merge 1 squash"...)
type forgeState struct {
	mu       sync.Mutex
	branches []string
	prs      []*fakePR
	reviews  map[int][]string // number -> "reviewer:STATE"
	calls    []string
}

func (s *forgeState) pr(n string) *fakePR {
	for _, p := range s.prs {
		if strconv.Itoa(p.number) == n {
			return p
		}
	}
	return nil
}

func prJSON(p *fakePR) string {
	draft := strings.HasPrefix(p.title, "WIP:")
	return fmt.Sprintf(`{"number":%d,"title":%q,"body":%q,"state":%q,"draft":%t,"user":{"login":"acme"},`+
		`"head":{"ref":%q,"sha":%q,"repo":{"full_name":"acme/web"}},"base":{"ref":%q,"sha":"b","repo":{"full_name":"acme/web"}},`+
		`"mergeable":%t,"merged":%t,"html_url":"https://forge/acme/web/pulls/%d"}`,
		p.number, p.title, p.body, p.state, draft, p.head, p.headSHA, p.base, !p.conflicts, p.merged, p.number)
}

func prServer(t *testing.T, s *forgeState) {
	t.Helper()
	const base = "/api/v1/repos/acme/web"
	s.reviews = map[int][]string{}
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET " + base: testutil.Reply(`{"id":1,"owner":{"login":"acme"},"name":"web","full_name":"acme/web","default_branch":"develop"}`),
		"GET " + base + "/branches/{b}": func(w http.ResponseWriter, r *http.Request) {
			if !slices.Contains(s.branches, r.PathValue("b")) {
				testutil.Status(http.StatusNotFound)(w, r)
				return
			}
			testutil.Reply(fmt.Sprintf(`{"name":%q,"commit":{"id":"abc"}}`, r.PathValue("b")))(w, r)
		},
		"GET " + base + "/pulls": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var out []string
			for _, p := range s.prs {
				if st := r.URL.Query().Get("state"); st == "all" || st == p.state {
					out = append(out, prJSON(p))
				}
			}
			testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
		},
		"GET " + base + "/pulls/{n}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if p := s.pr(r.PathValue("n")); p != nil {
				testutil.Reply(prJSON(p))(w, r)
				return
			}
			testutil.Status(http.StatusNotFound)(w, r)
		},
		"POST " + base + "/pulls": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			var b struct {
				Title string `json:"title"`
				Body  string `json:"body"`
				Head  string `json:"head"`
				Base  string `json:"base"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			p := &fakePR{number: len(s.prs) + 1, title: b.Title, body: b.Body, state: "open", head: b.Head, base: b.Base, headSHA: "h1"}
			s.prs = append(s.prs, p)
			s.calls = append(s.calls, fmt.Sprintf("create %s->%s %q", b.Head, b.Base, b.Title))
			w.WriteHeader(http.StatusCreated)
			testutil.Reply(prJSON(p))(w, r)
		},
		"PATCH " + base + "/pulls/{n}": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			p := s.pr(r.PathValue("n"))
			var b struct {
				Title *string `json:"title"`
				Body  *string `json:"body"`
				Base  *string `json:"base"`
				State *string `json:"state"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			var sent []string
			if b.Title != nil {
				p.title, sent = *b.Title, append(sent, "title="+*b.Title)
			}
			if b.Body != nil {
				p.body, sent = *b.Body, append(sent, "body")
			}
			if b.Base != nil {
				p.base, sent = *b.Base, append(sent, "base="+*b.Base)
			}
			if b.State != nil {
				p.state, sent = *b.State, append(sent, "state="+*b.State)
			}
			s.calls = append(s.calls, fmt.Sprintf("edit %d %s", p.number, strings.Join(sent, " ")))
			testutil.Reply(prJSON(p))(w, r)
		},
		"POST " + base + "/pulls/{n}/merge": func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			p := s.pr(r.PathValue("n"))
			var b struct {
				Do      string `json:"Do"`
				HeadSHA string `json:"head_commit_id"`
				Delete  bool   `json:"delete_branch_after_merge"`
				Auto    bool   `json:"merge_when_checks_succeed"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			if b.HeadSHA != p.headSHA {
				testutil.Status(http.StatusConflict)(w, r)
				return
			}
			s.calls = append(s.calls, fmt.Sprintf("merge %d %s delete=%t auto=%t", p.number, b.Do, b.Delete, b.Auto))
			if !b.Auto {
				p.merged, p.state = true, "closed"
			}
			w.WriteHeader(http.StatusOK)
		},
		"DELETE " + base + "/pulls/{n}/merge": func(w http.ResponseWriter, r *http.Request) {
			s.calls = append(s.calls, "cancel-auto "+r.PathValue("n"))
			w.WriteHeader(http.StatusNoContent)
		},
		"POST " + base + "/pulls/{n}/requested_reviewers": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				Reviewers []string `json:"reviewers"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			s.calls = append(s.calls, fmt.Sprintf("request %s %s", r.PathValue("n"), strings.Join(b.Reviewers, ",")))
			w.WriteHeader(http.StatusCreated)
			testutil.Reply("[]")(w, r)
		},
		"DELETE " + base + "/pulls/{n}/requested_reviewers": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				Reviewers []string `json:"reviewers"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			s.calls = append(s.calls, fmt.Sprintf("unrequest %s %s", r.PathValue("n"), strings.Join(b.Reviewers, ",")))
			w.WriteHeader(http.StatusNoContent)
		},
		"GET " + base + "/pulls/{n}/reviews": func(w http.ResponseWriter, r *http.Request) {
			n, _ := strconv.Atoi(r.PathValue("n"))
			var out []string
			for i, rv := range s.reviews[n] {
				who, state, _ := strings.Cut(rv, ":")
				out = append(out, fmt.Sprintf(`{"id":%d,"user":{"login":%q},"state":%q}`, i+1, who, state))
			}
			testutil.Reply("["+strings.Join(out, ",")+"]")(w, r)
		},
		"POST " + base + "/pulls/{n}/reviews": func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				Event string `json:"event"`
				Body  string `json:"body"`
			}
			_ = json.UnmarshalRead(r.Body, &b)
			s.calls = append(s.calls, fmt.Sprintf("review %s %s %q", r.PathValue("n"), b.Event, b.Body))
			testutil.Reply(fmt.Sprintf(`{"id":9,"user":{"login":"acme"},"state":%q}`, b.Event))(w, r)
		},
	})
}

func TestListAndGet(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, title: "One", state: "open", head: "f1", base: "develop"},
		{number: 2, title: "Two", state: "closed", head: "f2", base: "develop", merged: true}}}
	prServer(t, s)
	s.reviews[1] = []string{"bob:APPROVED", "carol:REQUEST_REVIEW"}

	list, err := ListPRs("web", false)
	if err != nil || len(list) != 1 || list[0].Number != 1 {
		t.Fatalf("%v, %+v", err, list)
	}
	testutil.Set(t, &ListState, "ALL")
	out := testutil.StripANSI(testutil.Run(t, func() { list, err = ListPRs("web", true) }))
	if err != nil || len(list) != 2 || !strings.Contains(out, "f2 → develop") || !strings.Contains(out, "merged") {
		t.Fatalf("%v:\n%s", err, out)
	}
	testutil.Set(t, &ListState, "nope")
	if _, err = ListPRs("web", false); err == nil || err.Title != "Invalid state" {
		t.Fatalf("bad state: %+v", err)
	}

	var pr *PullRequest
	out = testutil.StripANSI(testutil.Run(t, func() { pr, err = GetPR("web", "#1", true) }))
	if err != nil || len(pr.Reviews) != 2 {
		t.Fatalf("%v, %+v", err, pr)
	}
	for _, want := range []string{"#1 One", "Mergeable:", "yes", "bob: approved", "carol: requested"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if _, err = GetPR("web", "9", false); err == nil || err.Title != "No such pull request" {
		t.Fatalf("unknown: %+v", err)
	}
	if _, err = GetPR("web", "x1", false); err == nil || err.Title != "Invalid pull request number" {
		t.Fatalf("bad number: %+v", err)
	}
}

func TestCreatePR(t *testing.T) {
	defaults(t)
	s := &forgeState{branches: []string{"develop", "feature"}}
	prServer(t, s)

	// The base defaults to the default branch, the title to the head's name; reviewers are requested after
	testutil.Set(t, &PRReviewers, []string{"bob", "carol"})
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreatePR("web", "feature", "", none) })
	if err != nil || !slices.Equal(s.calls, []string{`create feature->develop "feature"`, "request 1 bob,carol"}) ||
		!strings.Contains(out, "Created pull request #1 (feature → develop)") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}

	// A draft gets the WIP prefix (Forgejo), and is reported as one
	testutil.Set(t, &PRReviewers, nil)
	testutil.Set(t, &PRDraft, true)
	testutil.Set(t, &PRTitle, "Second")
	out = testutil.Run(t, func() { err = CreatePR("web", "feature", "develop", none) })
	if err != nil || s.calls[2] != `create feature->develop "WIP: Second"` || strings.Contains(out, "not a draft") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}

	for _, tc := range []struct{ head, base, want string }{{"nope", "", "No such branch"}, {"feature", "gone", "No such branch"}, {"develop", "", "Same branches"}} {
		if err = CreatePR("web", tc.head, tc.base, none); err == nil || err.Title != tc.want {
			t.Errorf("%s -> %s: %+v", tc.head, tc.base, err)
		}
	}
	if len(s.calls) != 3 {
		t.Fatalf("calls %v", s.calls)
	}
}

func TestEditPR(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, title: "One", state: "open", head: "f", base: "develop"}}}
	prServer(t, s)

	var err *cerr.CustomError
	if err = EditPR("web", "1", none); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("no flag: %+v", err)
	}
	// Only the flags passed; --draft adds the prefix, --draft=false removes it
	testutil.Set(t, &PRBase, "main")
	testutil.Set(t, &PRDraft, true)
	out := testutil.Run(t, func() { err = EditPR("web", "1", changedFlags("base", "draft")) })
	if err != nil || s.calls[0] != "edit 1 title=WIP: One base=main" || !strings.Contains(out, "(draft)") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
	testutil.Set(t, &PRDraft, false)
	testutil.Set(t, &PRBody, "")
	testutil.Run(t, func() { err = EditPR("web", "1", changedFlags("draft", "body")) })
	if err != nil || s.calls[1] != "edit 1 title=One body" {
		t.Fatalf("%v, calls %v", err, s.calls)
	}
	if err = EditPR("web", "7", changedFlags("body")); err == nil || err.Title != "No such pull request" {
		t.Fatalf("unknown: %+v", err)
	}
}

func TestClosePRs(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, state: "open"}, {number: 2, state: "closed"}, {number: 3, state: "closed", merged: true}}}
	prServer(t, s)

	// Every number is checked first: an unknown one stops everything
	var err *cerr.CustomError
	if err = ClosePRs("web", []string{"1", "8"}, false); err == nil || err.Title != "No such pull request" || len(s.calls) != 0 {
		t.Fatalf("unknown: %+v, calls %v", err, s.calls)
	}
	out := testutil.Run(t, func() { err = ClosePRs("web", []string{"1", "2", "3", "#1"}, false) })
	if err != nil || !slices.Equal(s.calls, []string{"edit 1 state=closed"}) ||
		!strings.Contains(out, "#2 is already closed") || !strings.Contains(out, "#3 is already merged") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}

	if err = ClosePRs("web", []string{"1", "3"}, true); err == nil || err.Title != "Pull request merged" || len(s.calls) != 1 {
		t.Fatalf("reopen merged: %+v, calls %v", err, s.calls)
	}
	testutil.Run(t, func() { err = ClosePRs("web", []string{"1", "2"}, true) })
	if err != nil || !slices.Equal(s.calls[1:], []string{"edit 1 state=open", "edit 2 state=open"}) {
		t.Fatalf("%v, calls %v", err, s.calls)
	}
}

func TestMergePR(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, title: "One", state: "open", head: "f", base: "develop", headSHA: "h1"},
		{number: 2, title: "WIP: Two", state: "open", headSHA: "h2"}, {number: 3, title: "Three", state: "open", headSHA: "h3", conflicts: true},
		{number: 4, state: "closed"}}}
	prServer(t, s)

	var err *cerr.CustomError
	for _, tc := range []struct{ n, want string }{{"2", "Draft pull request"}, {"3", "Conflicts"}, {"4", "Pull request closed"}} {
		if err = MergePR("web", tc.n); err == nil || err.Title != tc.want {
			t.Errorf("#%s: %+v", tc.n, err)
		}
	}
	testutil.Set(t, &MergeMethod, "octopus")
	if err = MergePR("web", "1"); err == nil || err.Title != "Invalid merge method" {
		t.Fatalf("bad method: %+v", err)
	}
	if len(s.calls) != 0 {
		t.Fatalf("calls %v", s.calls)
	}

	// The head SHA is passed, so that a head that moved is refused by the server
	testutil.Set(t, &MergeMethod, "Squash")
	testutil.Set(t, &DeleteBranch, true)
	out := testutil.Run(t, func() { err = MergePR("web", "1") })
	if err != nil || !slices.Equal(s.calls, []string{"merge 1 squash delete=true auto=false"}) ||
		!strings.Contains(out, "Merged pull request #1 (squash) into develop") || !strings.Contains(out, "Deleted branch") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
	if err = MergePR("web", "1"); err == nil || err.Title != "Already merged" {
		t.Fatalf("merged twice: %+v", err)
	}

	// --auto lets a conflicting pull request through (the checks decide), --cancel-auto cancels
	testutil.Set(t, &DeleteBranch, false)
	testutil.Set(t, &AutoMerge, true)
	out = testutil.Run(t, func() { err = MergePR("web", "3") })
	if err != nil || s.calls[1] != "merge 3 squash delete=false auto=true" || !strings.Contains(out, "once its checks pass") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
	testutil.Set(t, &AutoMerge, false)
	testutil.Set(t, &CancelAutoMerge, true)
	testutil.Run(t, func() { err = MergePR("web", "3") })
	if err != nil || s.calls[2] != "cancel-auto 3" {
		t.Fatalf("%v, calls %v", err, s.calls)
	}
}

func TestReviewPR(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, state: "open"}, {number: 2, state: "closed"}}}
	prServer(t, s)

	var err *cerr.CustomError
	if err = ReviewPR("web", "1"); err == nil || err.Title != "Invalid flags" {
		t.Fatalf("no verdict: %+v", err)
	}
	testutil.Set(t, &ReviewComment, true)
	if err = ReviewPR("web", "1"); err == nil || err.Title != "Missing text" {
		t.Fatalf("comment without text: %+v", err)
	}
	testutil.Set(t, &ReviewComment, false)
	testutil.Set(t, &ReviewApprove, true)
	if err = ReviewPR("web", "2"); err == nil || err.Title != "Pull request closed" {
		t.Fatalf("closed: %+v", err)
	}
	out := testutil.Run(t, func() { err = ReviewPR("web", "1") })
	if err != nil || !slices.Equal(s.calls, []string{`review 1 APPROVED ""`}) || !strings.Contains(out, "Reviewed pull request #1: approved") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
	testutil.Set(t, &ReviewApprove, false)
	testutil.Set(t, &ReviewRequestChanges, true)
	testutil.Set(t, &ReviewBody, "add tests")
	out = testutil.Run(t, func() { err = ReviewPR("web", "1") })
	if err != nil || s.calls[1] != `review 1 REQUEST_CHANGES "add tests"` || !strings.Contains(out, "changes requested") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
}

func TestSetReviewers(t *testing.T) {
	defaults(t)
	s := &forgeState{prs: []*fakePR{{number: 1, state: "open"}}}
	prServer(t, s)

	var err *cerr.CustomError
	testutil.Run(t, func() { err = SetReviewers("web", "1", []string{"bob", "carol"}) })
	if err != nil {
		t.Fatal(err.Error())
	}
	testutil.Set(t, &RemoveReviewers, true)
	out := testutil.Run(t, func() { err = SetReviewers("web", "1", []string{"bob"}) })
	if err != nil || !slices.Equal(s.calls, []string{"request 1 bob,carol", "unrequest 1 bob"}) || !strings.Contains(out, "Withdrew") {
		t.Fatalf("%v, calls %v:\n%s", err, s.calls, out)
	}
	if err = SetReviewers("web", "5", []string{"bob"}); err == nil || err.Title != "No such pull request" {
		t.Fatalf("unknown: %+v", err)
	}
}
