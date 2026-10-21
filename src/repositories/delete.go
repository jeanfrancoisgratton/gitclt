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

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"

	//gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func DeleteRepo(reponame string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError

	ctx := context.Background()
	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	repos := c.Repos()
	if e := repos.Delete(ctx, OwnerOrg, reponame); e != nil {
		return &cerr.CustomError{Title: fmt.Sprintf("Unable to delete the repository %s/%s", OwnerOrg, reponame),
			Message: e.Error()}
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted repository %s", hftx.Blue(OwnerOrg+"/"+reponame))))
	}
	return nil
}
