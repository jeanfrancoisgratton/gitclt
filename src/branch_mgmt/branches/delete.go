// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/delete.go
// Original timestamp : 2026.10.27 19:22:41

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os/exec"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteBranch deletes branches of OwnerOrg/repo: in the local working copy (the default), on the server, or both
// The first failure stops the run; the branches before it stay deleted
func DeleteBranch(repo string, branches []string) *cerr.CustomError {
	var c *forge.Client
	var lr *localRepo
	var ce *cerr.CustomError
	ctx := context.Background()

	// -a, or -l and -r together, mean both sides; neither flag means the local side only
	withLocal, withServer := !RemoteOnly, RemoteOnly
	if AllBranches || (LocalOnly && RemoteOnly) {
		withLocal, withServer = true, true
	}

	if withServer && OwnerOrg == "" {
		return &cerr.CustomError{Title: "Missing owner", Message: "deleting on the server needs the owner (-o OWNER)"}
	}

	// Check everything before asking: the working copy must be a clone of the repo, and git must be there to write to it
	if withLocal {
		if lr, ce = localBranches(repo); ce != nil {
			return ce
		}
		if _, e := exec.LookPath("git"); e != nil {
			return &cerr.CustomError{Title: "git not found", Message: "deleting local branches needs the git client: " + e.Error()}
		}
		for _, branch := range branches {
			if _, ok := lr.branches[branch]; !ok {
				return &cerr.CustomError{Title: "No such branch", Message: "the local working copy has no branch " + branch}
			}
		}
	}

	// Deletion cannot be undone; ask unless --yes was passed
	if !AssumeYes {
		sides := "local"
		switch {
		case withLocal && withServer:
			sides = "local + server"
		case withServer:
			sides = "server"
		}
		answer := hf.GetStringValFromPrompt(hftx.WarningSign(fmt.Sprintf("Delete branch(es) %s (%s) ? [y/N]: ", strings.Join(branches, ", "), sides)))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, no branch is deleted")
			return nil
		}
	}

	if withServer {
		if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
			return ce
		}
	}

	for _, branch := range branches {
		// Local first: git refuses an unmerged or checked out branch, and the server's copy is then left alone
		if withLocal {
			if ce = deleteLocalBranch(branch); ce != nil {
				return ce
			}
			if !shared.QuietOutput {
				fmt.Println(hftx.EnabledSign("Deleted local branch " + hftx.Brown(branch)))
			}
		}
		if withServer {
			if e := c.Branches().Delete(ctx, OwnerOrg, repo, branch); e != nil {
				return forge.Error("Unable to delete the branch "+branch+" on the server", e)
			}
			if !shared.QuietOutput {
				fmt.Println(hftx.EnabledSign("Deleted server branch " + hftx.Brown(branch)))
			}
		}
	}
	return nil
}

// deleteLocalBranch has git do the deletion: it handles the refs, packed-refs, reflog and config, and its safety rules
// (no unmerged branch without --force, no branch checked out here or in another worktree)
func deleteLocalBranch(branch string) *cerr.CustomError {
	flag := "-d"
	if ForceDelete {
		flag = "-D"
	}
	if out, e := exec.Command("git", "branch", flag, branch).CombinedOutput(); e != nil {
		// git's hints point to its own flags (git branch -D); ours is --force
		var lines []string
		for l := range strings.Lines(string(out)) {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "hint:") {
				lines = append(lines, l)
			}
		}
		msg := strings.Join(lines, "; ")
		if msg == "" {
			msg = e.Error()
		}
		if strings.Contains(msg, "not fully merged") {
			msg += " (use --force to delete it anyway)"
		}
		return &cerr.CustomError{Title: "Unable to delete the local branch " + branch, Message: msg}
	}
	return nil
}
