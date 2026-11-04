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

// createBranch decides the default branch to ask for: --default-branch when given (a forge that cannot set it refuses it),
// else develop, unless the forge cannot set it: GitHub never can at creation (it uses the account's or organization's
// setting), and an empty Bitbucket repository has no branch yet
func createBranch(backend gitoperationslib.Backend) string {
	switch {
	case RepoDefaultBranch != "":
		return RepoDefaultBranch
	case backend == gitoperationslib.BackendGitHub:
		return ""
	case backend == gitoperationslib.BackendBitbucket && !InitializeRepo:
		return ""
	}
	return "develop"
}

// Createrepo : most of the values can be overridden from the CLI, otherwise sensible values are provided
func CreateRepo(reponame string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()
	cro := gitoperationslib.CreateRepoOptions{
		Name:        reponame,
		Description: RepoDescription,
		Template:    TemplateRepo,
		AutoInit:    InitializeRepo,
		Visibility:  "public",
	}
	if !PublicRepo {
		cro.Visibility = "private"
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if cro.Owner, ce = c.Owner(ctx, OwnerOrg); ce != nil {
		return ce
	}
	cro.DefaultBranch = createBranch(c.Backend())

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
			fmt.Println(hftx.WarningSign("NOTICE: The repo is not initialized, you will need to locally set it up (git init/commit/remote add/push)"))
		}
	}
	return nil
}
