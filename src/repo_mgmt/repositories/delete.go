// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/delete.go
// Original timestamp : 2026.10.21 00:19:32

package repositories

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteRepo deletes a repository, after confirmation: this cannot be undone
// The repository is checked first, so that nothing is asked about a repository that does not exist
func DeleteRepo(reponame string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return ce
	}
	repos := c.Repos()
	fullname := owner + "/" + reponame
	if _, e := repos.Get(ctx, owner, reponame); e != nil {
		if gol.IsNotFound(e) {
			return &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s does not exist", fullname)}
		}
		return forge.Error("Unable to get the repository "+fullname, e)
	}

	if !AssumeYes {
		answer := hf.GetStringValFromPrompt(hftx.WarningSign("Delete repository " + hftx.Brown(fullname) +
			" on " + c.Connection.Name + ", with everything in it ? This cannot be undone [y/N]: "))
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Println("Aborted, repository " + hftx.Brown(fullname) + " was not deleted")
			return nil
		}
	}

	if e := repos.Delete(ctx, owner, reponame); e != nil {
		return &cerr.CustomError{Title: "Unable to delete the repository " + fullname, Message: e.Error()}
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted repository %s", hftx.Brown(fullname))))
	}
	return nil
}
