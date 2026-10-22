// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/create.go
// Original timestamp : 2026.10.21 00:18:45

package repositories

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// Createrepo : most of the values can be overridden from the CLI, otherwise sensible values are provided
func CreateRepo(reponame string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()
	cro := gitoperationslib.CreateRepoOptions{
		Owner:         OwnerOrg,
		Name:          reponame,
		Description:   RepoDescription,
		DefaultBranch: RepoDefaultBranch,
		Template:      TemplateRepo,
		AutoInit:      InitializeRepo,
		Visibility:    "public",
	}
	if !PublicRepo {
		cro.Visibility = "private"
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}

	// gitignore templates are only applied on an initialized repo; names are case-sensitive
	// Bitbucket has no gitignore templates
	if InitializeRepo && c.Backend() != gitoperationslib.BackendBitbucket {
		cro.Gitignores = []string{"Go"}
	}

	repos := c.Repos()
	if _, e := repos.Create(ctx, cro); e != nil {
		return &cerr.CustomError{Title: "Could not create repository", Message: e.Error()}
	}

	if !shared.QuietOutput {
		fmt.Printf("%s %s has been %s\n", hftx.EnabledSign("Repository"), hftx.Brown(reponame),
			hftx.Green("created"))
		if !InitializeRepo {
			hftx.WarningSign("NOTICE: The repo is not initialized, you will need to locally set it up (git init/commit/remote add/push)")
		}
	}
	return nil
}
