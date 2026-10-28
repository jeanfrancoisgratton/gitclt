// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/branch_mgmt/branches/rename.go
// Original timestamp : 2026.10.28 09:12:30

package branches

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os/exec"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// RenameBranch renames a branch of OwnerOrg/repo: in the local working copy (the default), on the server, or both
// On the server, the open pull requests and the protection rules follow the branch
func RenameBranch(repo, branch, newName string) *cerr.CustomError {
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
		return &cerr.CustomError{Title: "Missing owner", Message: "renaming on the server needs the owner (-o OWNER)"}
	}

	// Check the local side before touching anything: the working copy must be a clone of the repo, with git to write to it
	if withLocal {
		if lr, ce = localBranches(repo); ce != nil {
			return ce
		}
		if _, e := exec.LookPath("git"); e != nil {
			return &cerr.CustomError{Title: "git not found", Message: "renaming local branches needs the git client: " + e.Error()}
		}
		if _, ok := lr.branches[branch]; !ok {
			return &cerr.CustomError{Title: "No such branch", Message: "the local working copy has no branch " + branch}
		}
		if _, ok := lr.branches[newName]; ok {
			return &cerr.CustomError{Title: "Branch already exists", Message: "the local working copy already has a branch " + newName}
		}
	}

	if withServer {
		if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
			return ce
		}
	}

	// Local first: if git refuses, the server's branch is left alone
	if withLocal {
		if out, e := exec.Command("git", "branch", "-m", branch, newName).CombinedOutput(); e != nil {
			return &cerr.CustomError{Title: "Unable to rename the local branch " + branch, Message: gitMessage(out, e)}
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Renamed local branch " + hftx.Brown(branch) + " to " + hftx.Brown(newName)))
		}
	}
	if withServer {
		if _, e := c.Branches().Rename(ctx, OwnerOrg, repo, branch, newName); e != nil {
			return forge.Error("Unable to rename the branch "+branch+" on the server", e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Renamed server branch " + hftx.Brown(branch) + " to " + hftx.Brown(newName)))
		}
	}

	// Both sides renamed: the local branch must now track the server's new name, not the old one
	if withLocal && withServer {
		out, _ := exec.Command("git", "config", "--get", "branch."+newName+".merge").Output()
		if strings.TrimSpace(string(out)) == "refs/heads/"+branch {
			if out, e := exec.Command("git", "config", "branch."+newName+".merge", "refs/heads/"+newName).CombinedOutput(); e != nil {
				return &cerr.CustomError{Title: "Unable to update the upstream of " + newName, Message: gitMessage(out, e)}
			}
		}
	}
	return nil
}
