// gitclt
// Tests branch rename: local (through git), server and both sides, against a real git working copy and a fake Forgejo

package branches

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
)

// renameServer records the bodies of the server's renames; Forgejo answers a rename with the renamed branch, fetched again
func renameServer(t *testing.T, status int) (bodies *[]map[string]any) {
	t.Helper()
	bodies = new([]map[string]any)
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/repos/acme/web/branches/dev":   testutil.Record(t, bodies, status, `{}`),
		"GET /api/v1/repos/acme/web/branches/develop": testutil.Reply(`{"name":"develop","commit":{"id":"` + shaMain + `"}}`),
		"GET /api/v1/repos/acme/web":                  testutil.Reply(`{"name":"web","default_branch":"main"}`),
	})
	return bodies
}

func TestRenameBranch_Local(t *testing.T) {
	defaults(t)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	// No prompt: a rename can be undone
	out := testutil.Run(t, func() {
		if ce := RenameBranch("web", "dev", "develop"); ce != nil {
			t.Error(ce.Error())
		}
	})
	names := localNames(t, root)
	if !strings.Contains(out, "Renamed local branch dev to develop") || slices.Contains(names, "dev") || !slices.Contains(names, "develop") {
		t.Fatalf("%v\n%s", names, out)
	}
	// Only the server side moves the upstream
	if m := git(t, root, "config", "--get", "branch.develop.merge"); m != "refs/heads/dev" {
		t.Errorf("upstream: %s", m)
	}

	// The checked out branch can be renamed too
	if ce := RenameBranch("web", "main", "trunk"); ce != nil || git(t, root, "branch", "--show-current") != "trunk" {
		t.Fatalf("checked out: %+v", ce)
	}
}

func TestRenameBranch_LocalRefusals(t *testing.T) {
	defaults(t)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	if ce := RenameBranch("web", "nope", "x"); ce == nil || ce.Title != "No such branch" {
		t.Fatalf("unknown branch: %+v", ce)
	}
	if ce := RenameBranch("web", "dev", "merged"); ce == nil || ce.Title != "Branch already exists" {
		t.Fatalf("existing name: %+v", ce)
	}
	// What only git knows about is still git's to refuse
	if ce := RenameBranch("web", "dev", "bad..name"); ce == nil || !strings.HasPrefix(ce.Title, "Unable to rename the local branch dev") {
		t.Fatalf("invalid name: %+v", ce)
	}
	if ce := RenameBranch("other", "dev", "develop"); ce == nil || ce.Title != "Not the same repository" {
		t.Fatalf("other repo: %+v", ce)
	}
	if !slices.Contains(localNames(t, root), "dev") {
		t.Fatal("dev was renamed anyway")
	}
}

func TestRenameBranch_Remote(t *testing.T) {
	defaults(t)
	testutil.Set(t, &RemoteOnly, true)
	bodies := renameServer(t, http.StatusOK)
	t.Chdir(t.TempDir()) // the local side is never read

	out := testutil.Run(t, func() {
		if ce := RenameBranch("web", "dev", "develop"); ce != nil {
			t.Error(ce.Error())
		}
	})
	if len(*bodies) != 1 || (*bodies)[0]["name"] != "develop" || !strings.Contains(out, "Renamed server branch dev to develop") {
		t.Fatalf("bodies %v\n%s", *bodies, out)
	}

	testutil.Set(t, &OwnerOrg, "")
	if ce := RenameBranch("web", "dev", "develop"); ce == nil || ce.Title != "Missing owner" {
		t.Fatalf("no owner: %+v", ce)
	}
}

func TestRenameBranch_All(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AllBranches, true)
	bodies := renameServer(t, http.StatusOK)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	out := testutil.Run(t, func() {
		if ce := RenameBranch("web", "dev", "develop"); ce != nil {
			t.Error(ce.Error())
		}
	})
	if len(*bodies) != 1 || !slices.Contains(localNames(t, root), "develop") ||
		strings.Index(out, "Renamed local branch") > strings.Index(out, "Renamed server branch") {
		t.Fatalf("bodies %v\n%s", *bodies, out)
	}
	// The local branch now tracks the server's new name
	if m := git(t, root, "config", "--get", "branch.develop.merge"); m != "refs/heads/develop" {
		t.Errorf("upstream: %s", m)
	}

	// A local refusal leaves the server's branch alone
	*bodies = nil
	if ce := RenameBranch("web", "develop", "merged"); ce == nil || len(*bodies) != 0 {
		t.Fatalf("existing name: %+v, bodies %v", ce, *bodies)
	}
}

// A server failure comes after the local rename, which stays done
func TestRenameBranch_RemoteFailure(t *testing.T) {
	defaults(t)
	testutil.Set(t, &AllBranches, true)
	renameServer(t, http.StatusConflict)
	root := gitWorkingCopy(t)
	t.Chdir(root)

	var ce error
	testutil.Run(t, func() {
		if e := RenameBranch("web", "dev", "develop"); e != nil {
			ce = e
			if e.Code != http.StatusConflict {
				t.Errorf("code: %+v", e)
			}
		}
	})
	if ce == nil || !slices.Contains(localNames(t, root), "develop") {
		t.Fatalf("got %v, branches %v", ce, localNames(t, root))
	}
}
