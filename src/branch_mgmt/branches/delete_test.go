// gitclt
// Tests branch rm: local (through git), server and both sides, against a real git working copy and a fake Forgejo

package branches

import (
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

// git runs a git command in dir, failing the test if git does
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitWorkingCopy creates a real working copy of acme/web, as the commands that write to it go through the git client:
// main (checked out), merged (same commit as main), unmerged (one commit ahead of main), and dev, which tracks origin/dev
// The user's own git configuration is kept out of it
func gitWorkingCopy(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("the git client is not installed")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(v, "Tester")
	}
	for _, v := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(v, "tester@example.com")
	}

	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "remote", "add", "origin", "git@forge.example:acme/web.git")
	git(t, root, "commit", "-q", "--allow-empty", "-m", "init")
	git(t, root, "branch", "merged")
	git(t, root, "branch", "dev")
	git(t, root, "config", "branch.dev.remote", "origin")
	git(t, root, "config", "branch.dev.merge", "refs/heads/dev")
	git(t, root, "checkout", "-q", "-b", "unmerged")
	git(t, root, "commit", "-q", "--allow-empty", "-m", "work")
	git(t, root, "checkout", "-q", "main")
	return root
}

func localNames(t *testing.T, root string) []string {
	t.Helper()
	return strings.Fields(git(t, root, "branch", "--format=%(refname:short)"))
}

// deleteServer answers the server's branch deletions with status, and records the deleted names
func deleteServer(t *testing.T, status int) (deleted *[]string) {
	t.Helper()
	deleted = new([]string)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/repos/acme/web/branches/{name}": func(w http.ResponseWriter, r *http.Request) {
			*deleted = append(*deleted, r.PathValue("name"))
			testutil.Status(status)(w, r)
		},
	})
	return deleted
}

func TestDeleteBranch_Local(t *testing.T) {
	defaults(t)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	// The prompt names the side; anything but y/yes aborts
	out := testutil.Run(t, func() {
		if ce := DeleteBranch("web", []string{"merged"}); ce != nil {
			t.Error(ce.Error())
		}
	}, "n")
	if !strings.Contains(out, "merged (local) ? [y/N]") || !strings.Contains(out, "Aborted") || !slices.Contains(localNames(t, root), "merged") {
		t.Fatalf("abort:\n%s", out)
	}

	out = testutil.Run(t, func() {
		if ce := DeleteBranch("web", []string{"merged"}); ce != nil {
			t.Error(ce.Error())
		}
	}, "y")
	if !strings.Contains(out, "Deleted local branch merged") || slices.Contains(localNames(t, root), "merged") {
		t.Fatalf("delete:\n%s", out)
	}
}

func TestDeleteBranch_LocalRefusals(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	// git refuses an unmerged branch, and the message points to --force rather than to git's -D
	ce := DeleteBranch("web", []string{"unmerged"})
	if ce == nil || !strings.Contains(ce.Message, "not fully merged (use --force") || strings.Contains(ce.Message, "hint:") {
		t.Fatalf("unmerged: %+v", ce)
	}
	testutil.Set(t, &ForceDelete, true)
	testutil.Run(t, func() {
		if ce := DeleteBranch("web", []string{"unmerged"}); ce != nil {
			t.Error(ce.Error())
		}
	})
	if slices.Contains(localNames(t, root), "unmerged") {
		t.Fatal("unmerged: still there with --force")
	}

	// Even forced, git never deletes the checked out branch
	if ce = DeleteBranch("web", []string{"main"}); ce == nil || !strings.HasPrefix(ce.Title, "Unable to delete the local branch main") {
		t.Fatalf("checked out: %+v", ce)
	}

	// An unknown branch is caught before anything is deleted, even when it is not the first
	if ce = DeleteBranch("web", []string{"merged", "nope"}); ce == nil || ce.Title != "No such branch" || !slices.Contains(localNames(t, root), "merged") {
		t.Fatalf("unknown branch: %+v", ce)
	}

	// The working copy must be the repo's
	if ce = DeleteBranch("other", []string{"merged"}); ce == nil || ce.Title != "Not the same repository" {
		t.Fatalf("other repo: %+v", ce)
	}
	t.Chdir(t.TempDir())
	if ce = DeleteBranch("web", []string{"merged"}); ce == nil || ce.Title != "No local working copy" {
		t.Fatalf("no working copy: %+v", ce)
	}
}

func TestDeleteBranch_Remote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &RemoteOnly, true)
	deleted := deleteServer(t, http.StatusNoContent)
	t.Chdir(t.TempDir()) // the local side is never read

	out := testutil.Run(t, func() {
		if ce := DeleteBranch("web", []string{"old", "older"}); ce != nil {
			t.Error(ce.Error())
		}
	})
	if !slices.Equal(*deleted, []string{"old", "older"}) || !strings.Contains(out, "Deleted server branch older") || strings.Contains(out, "local") {
		t.Fatalf("deleted %v:\n%s", *deleted, out)
	}

	// Without -o, the connection's username is the owner
	testutil.Set(t, &OwnerOrg, "")
	if ce := DeleteBranch("web", []string{"old"}); ce != nil || len(*deleted) != 3 {
		t.Fatalf("no -o: %+v, deleted %v", ce, *deleted)
	}
}

// The first failure stops the run
func TestDeleteBranch_RemoteFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AssumeYes, true)
	testutil.Set(t, &RemoteOnly, true)
	deleted := deleteServer(t, http.StatusForbidden)

	ce := DeleteBranch("web", []string{"old", "older"})
	if ce == nil || ce.Code != http.StatusForbidden || !slices.Equal(*deleted, []string{"old"}) {
		t.Fatalf("got %+v, deleted %v", ce, *deleted)
	}
}

// -a, or -l with -r: local first, then the server
func TestDeleteBranch_All(t *testing.T) {
	for _, flags := range [][]*bool{{&AllBranches}, {&LocalOnly, &RemoteOnly}} {
		defaults(t)
		testutil.Set(t, &AssumeYes, true)
		for _, f := range flags {
			testutil.Set(t, f, true)
		}
		deleted := deleteServer(t, http.StatusNoContent)
		root := gitWorkingCopy(t)
		t.Chdir(root)

		out := testutil.Run(t, func() {
			if ce := DeleteBranch("web", []string{"merged"}); ce != nil {
				t.Error(ce.Error())
			}
		})
		if !slices.Equal(*deleted, []string{"merged"}) || slices.Contains(localNames(t, root), "merged") ||
			strings.Index(out, "Deleted local branch") > strings.Index(out, "Deleted server branch") {
			t.Fatalf("all=%t local=%t remote=%t: deleted %v\n%s", AllBranches, LocalOnly, RemoteOnly, *deleted, out)
		}

		// A local refusal leaves the server's branch alone
		*deleted = nil
		if ce := DeleteBranch("web", []string{"unmerged"}); ce == nil || len(*deleted) != 0 {
			t.Fatalf("unmerged: %+v, deleted %v", ce, *deleted)
		}
	}
}
