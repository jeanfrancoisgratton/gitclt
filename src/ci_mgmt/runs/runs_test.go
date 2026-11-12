// gitclt
// Tests the run, job and artifact commands against a fake Forgejo

package runs

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func defaults(t *testing.T) {
	t.Helper()
	for _, s := range []*string{&OwnerOrg, &ListWorkflow, &ListStatus, &ListEvent, &ListBranch, &ListSHA, &ArtifactName, &OutputFile, &shared.ConnectionName} {
		testutil.Set(t, s, "")
	}
	for _, b := range []*bool{&JSONoutput, &AssumeYes, &Overwrite, &FollowLog, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	testutil.Set(t, &ListLimit, 30)
	testutil.Set(t, &ArtifactRun, 0)
}

const (
	runDone    = `{"id":2,"index_in_repo":2,"title":"Fix","workflow_id":"build.yml","trigger_event":"push","prettyref":"main","commit_sha":"6c1a4d9abcdef","status":"success","trigger_user":{"login":"acme"}}`
	runGoing   = `{"id":5,"index_in_repo":5,"title":"Slow","workflow_id":"slow.yml","trigger_event":"push","prettyref":"main","status":"running","trigger_user":{"login":"acme"}}`
	jobs       = `[{"id":3,"run_id":2,"attempt":1,"name":"build","status":"success","runs_on":["self"]},{"id":4,"run_id":2,"attempt":1,"name":"after","status":"success"}]`
	artifactOK = `{"id":1,"name":"result","size_in_bytes":157,"run_id":2}`
)

type fake struct{ calls []string }

// A fake Forgejo of this version, with runs 2 (finished) and 5 (still going)
func server(t *testing.T, version string) *fake {
	t.Helper()
	f := &fake{}
	write := func(status int, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/api/v1/repos/acme/web/actions/"))
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}
	}
	base := "/api/v1/repos/acme/web/actions/"
	testutil.ForgejoAt(t, version, map[string]http.HandlerFunc{
		"GET /api/v1/repos/acme/web": testutil.Reply(`{"name":"web"}`),
		"GET " + base + "runs": func(w http.ResponseWriter, r *http.Request) {
			f.calls = append(f.calls, "LIST "+r.URL.RawQuery)
			testutil.Reply(`{"workflow_runs":[`+runGoing+`,`+runDone+`],"total_count":2}`)(w, r)
		},
		"GET " + base + "runs/2":            testutil.Reply(runDone),
		"GET " + base + "runs/5":            testutil.Reply(runGoing),
		"GET " + base + "runs/9":            testutil.Status(http.StatusNotFound),
		"GET " + base + "runs/2/jobs":       testutil.Reply(jobs),
		"GET " + base + "runs/2/logs":       write(http.StatusOK, "PK-zip-bytes"),
		"POST " + base + "runs/{id}/cancel": write(http.StatusOK, ``),
		"POST " + base + "runs/{id}/rerun":  write(http.StatusCreated, ``),
		"DELETE " + base + "runs/{id}":      write(http.StatusNoContent, ``),
		"GET " + base + "jobs/3":            testutil.Reply(`{"id":3,"run_id":2,"attempt":1,"name":"build","status":"success","steps":[{"number":0,"name":"Set up job","status":"success"}]}`),
		"GET " + base + "jobs/3/logs":       write(http.StatusOK, "hello from the build job\n"),
		"POST " + base + "jobs/{id}/rerun":  write(http.StatusCreated, ``),
		"GET " + base + "artifacts":         testutil.Reply(`[` + artifactOK + `]`),
		"GET " + base + "artifacts/1":       testutil.Reply(artifactOK),
		"GET " + base + "artifacts/7":       testutil.Reply(`{"id":7,"name":"old","expired":true,"run_id":2}`),
		"GET " + base + "artifacts/1/zip":   write(http.StatusOK, "PK-artifact"),
		"DELETE " + base + "artifacts/{id}": write(http.StatusNoContent, ``),
	})
	return f
}

func TestListRuns(t *testing.T) {
	defaults(t)
	f := server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListRuns("web", true) })
	// The outcome shows once a run ended, its status before
	if err != nil || !strings.Contains(out, "build.yml") || !strings.Contains(out, "success") || !strings.Contains(out, "running") {
		t.Fatalf("%v\n%s", err, out)
	}
	testutil.Set(t, &ListWorkflow, "build.yml")
	testutil.Set(t, &ListBranch, "main")
	testutil.Set(t, &shared.QuietOutput, true)
	f.calls = nil
	if out = testutil.Run(t, func() { _, err = ListRuns("web", true) }); err != nil || out != "5\n2\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	if !strings.Contains(f.calls[0], "workflow_id=build.yml") || !strings.Contains(f.calls[0], "ref=refs%2Fheads%2Fmain") {
		t.Fatalf("filters not sent: %v", f.calls)
	}
	testutil.Set(t, &ListLimit, 1)
	if l, err := ListRuns("web", false); err != nil || len(l) != 1 {
		t.Fatalf("limit: %d, %v", len(l), err)
	}
	testutil.Set(t, &ListLimit, -1)
	if _, err = ListRuns("web", false); err == nil || err.Title != "Invalid limit" {
		t.Fatalf("negative limit: %+v", err)
	}
	testutil.Set(t, &ListLimit, 30)
	if _, err = ListRuns("nope", false); err == nil || err.Title != "No such repository" {
		t.Fatalf("no repo: %+v", err)
	}
}

func TestGetRun(t *testing.T) {
	defaults(t)
	server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = GetRun("web", "2", true) })
	if err != nil || !strings.Contains(out, "6c1a4d9") || !strings.Contains(out, "after") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err = GetRun("web", "9", false); err == nil || err.Title != "No such run" {
		t.Fatalf("unknown: %+v", err)
	}
	if _, err = GetRun("web", "x", false); err == nil || err.Title != "Invalid run ID" {
		t.Fatalf("not an ID: %+v", err)
	}
}

// Forgejo 15 lists runs but not their jobs: the run is still shown
func TestGetRun_OldForgejo(t *testing.T) {
	defaults(t)
	server(t, "15.0.2")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = GetRun("web", "2", true) })
	if err != nil || !strings.Contains(out, "Fix") || !strings.Contains(out, "No jobs") || !strings.Contains(out, "16") {
		t.Fatalf("%v\n%s", err, out)
	}
	if err = RerunRun("web", "2"); err == nil || err.Title != "Unsupported operation" || !strings.Contains(err.Error(), "17") {
		t.Fatalf("rerun: %+v", err)
	}
}

func TestCancelRuns(t *testing.T) {
	defaults(t)
	f := server(t, "17.0.1")
	var err *cerr.CustomError
	// A finished run is skipped: the forges would take it for a success
	out := testutil.Run(t, func() { err = CancelRuns("web", []string{"2", "5", "5"}) })
	if err != nil || !strings.Contains(out, "Run 2 of acme/web already finished: success") || !slices.Equal(f.calls, []string{"POST runs/5/cancel"}) {
		t.Fatalf("%v, %v\n%s", err, f.calls, out)
	}
	f.calls = nil
	if err = CancelRuns("web", []string{"5", "9"}); err == nil || err.Title != "No such run" || len(f.calls) != 0 {
		t.Fatalf("unknown: %+v, %v", err, f.calls)
	}
}

func TestDeleteRuns(t *testing.T) {
	defaults(t)
	f := server(t, "17.0.1")
	var err *cerr.CustomError
	// One still going stops everything, before the question
	if err = DeleteRuns("web", []string{"2", "5"}); err == nil || err.Title != "Run not finished" || len(f.calls) != 0 {
		t.Fatalf("running: %+v, %v", err, f.calls)
	}
	out := testutil.Run(t, func() { err = DeleteRuns("web", []string{"2"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || !strings.Contains(out, "acme/web") || len(f.calls) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, f.calls, out)
	}
	testutil.Run(t, func() { err = DeleteRuns("web", []string{"2"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE runs/2"}) {
		t.Fatalf("yes: %v, %v", err, f.calls)
	}
}

func TestRerun(t *testing.T) {
	defaults(t)
	f := server(t, "17.0.1")
	var err *cerr.CustomError
	testutil.Run(t, func() { err = RerunRun("web", "2") })
	testutil.Run(t, func() { err = RerunJob("web", "3") })
	if err != nil || !slices.Equal(f.calls, []string{"POST runs/2/rerun", "POST jobs/3/rerun"}) {
		t.Fatalf("%v, %v", err, f.calls)
	}
}

func TestJobs(t *testing.T) {
	defaults(t)
	server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListJobs("web", "2", true) })
	if err != nil || !strings.Contains(out, "build") || !strings.Contains(out, "after") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if out = testutil.Run(t, func() { _, err = GetJob("web", "3", true) }); err != nil || !strings.Contains(out, "Set up job") {
		t.Fatalf("get: %v\n%s", err, out)
	}
	if out = testutil.Run(t, func() { err = JobLog("web", "3") }); err != nil || out != "hello from the build job\n" {
		t.Fatalf("log: %v\n%q", err, out)
	}
}

// A file is never replaced unless asked, and a server name never chooses where it goes
func TestDownloads(t *testing.T) {
	defaults(t)
	server(t, "17.0.1")
	t.Chdir(t.TempDir())
	var err *cerr.CustomError

	testutil.Run(t, func() { err = RunLogs("web", "2") })
	if b, e := os.ReadFile("run-2-logs.zip"); err != nil || e != nil || string(b) != "PK-zip-bytes" {
		t.Fatalf("run logs: %v, %v, %q", err, e, b)
	}
	if err = RunLogs("web", "2"); err == nil || err.Title != "File already exists" {
		t.Fatalf("existing: %+v", err)
	}
	testutil.Set(t, &Overwrite, true)
	if testutil.Run(t, func() { err = RunLogs("web", "2") }); err != nil {
		t.Fatalf("--force: %v", err)
	}
	testutil.Set(t, &Overwrite, false)

	testutil.Run(t, func() { err = DownloadArtifact("web", "1") })
	if b, e := os.ReadFile("result.zip"); err != nil || e != nil || string(b) != "PK-artifact" {
		t.Fatalf("artifact: %v, %v, %q", err, e, b)
	}
	if err = DownloadArtifact("web", "7"); err == nil || err.Title != "Artifact expired" {
		t.Fatalf("expired: %+v", err)
	}
	testutil.Set(t, &OutputFile, "-")
	if out := testutil.Run(t, func() { err = DownloadArtifact("web", "1") }); err != nil || out != "PK-artifact" {
		t.Fatalf("stdout: %v, %q", err, out)
	}
	testutil.Set(t, &OutputFile, filepath.Join(t.TempDir(), "job.log"))
	if testutil.Run(t, func() { err = JobLog("web", "3") }); err != nil {
		t.Fatalf("job log to a file: %v", err)
	}
}

func TestSavedName_StaysHere(t *testing.T) {
	for name, want := range map[string]string{"result": "result.zip", "../../etc/passwd": "passwd.zip", "a/b": "b.zip"} {
		if got := filepath.Base(filepath.Clean("/"+name)) + ".zip"; got != want {
			t.Errorf("%q: got %q", name, got)
		}
	}
}

func TestArtifacts(t *testing.T) {
	defaults(t)
	f := server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { _, err = ListArtifacts("web", true) })
	if err != nil || !strings.Contains(out, "result") || !strings.Contains(out, "157 B") {
		t.Fatalf("list: %v\n%s", err, out)
	}
	testutil.Run(t, func() { err = DeleteArtifacts("web", []string{"1"}) }, "y")
	if err != nil || !slices.Equal(f.calls, []string{"DELETE artifacts/1"}) {
		t.Fatalf("delete: %v, %v", err, f.calls)
	}
}

func TestBitbucket(t *testing.T) {
	defaults(t)
	testutil.WriteRegistry(t, `{"auths":{"bb":{"url":"http://127.0.0.1:1","token":"x","username":"acme","owner":"acme","servertype":"bitbucket"}},"default_connection":"bb"}`)
	if _, err := ListRuns("web", false); err == nil || err.Title != "Unsupported operation" {
		t.Fatalf("got %+v", err)
	}
}

func TestSize(t *testing.T) {
	for n, want := range map[int64]string{17: "17 B", 2048: "2.0 KiB", 3 << 20: "3.0 MiB"} {
		if got := size(n); got != want {
			t.Errorf("%d: %s", n, got)
		}
	}
}

// -f writes the log until the job ends; job 3 has already ended, so its whole log comes in one read
// Forgejo needs 17 for it: the library says so on 16
func TestJobLog_Follow(t *testing.T) {
	defaults(t)
	testutil.Set(t, &FollowLog, true)
	server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = JobLog("web", "3") })
	if err != nil || out != "hello from the build job\n" {
		t.Fatalf("%v, output %q", err, out)
	}

	server(t, "16.0.7")
	if err = JobLog("web", "3"); err == nil || err.Title != "Unsupported operation" || !strings.Contains(err.Message, "17") {
		t.Fatalf("Forgejo 16: %+v", err)
	}
}

// Not on a terminal (as in the tests), the log is written as is: the pager is only for an interactive session
func TestJobLog_NotPagedWhenPiped(t *testing.T) {
	defaults(t)
	server(t, "17.0.1")
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = JobLog("web", "3") })
	if err != nil || out != "hello from the build job\n" {
		t.Fatalf("%v, output %q", err, out)
	}
}
