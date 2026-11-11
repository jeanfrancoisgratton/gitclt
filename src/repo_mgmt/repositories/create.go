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

// createFeatures turns Actions on unless --actions=false; Bitbucket has no features, so it only gets the flag when it
// was explicitly passed (changed is cmd.Flags().Changed), and then the library refuses it
func createFeatures(backend gitoperationslib.Backend, changed func(string) bool) gitoperationslib.RepoFeatures {
	if backend == gitoperationslib.BackendBitbucket && !changed("actions") {
		return gitoperationslib.RepoFeatures{}
	}
	return gitoperationslib.RepoFeatures{Actions: gitoperationslib.Ptr(CreateActions)}
}

// Createrepo : most of the values can be overridden from the CLI, otherwise sensible values are provided
func CreateRepo(reponame string, changed func(string) bool) *cerr.CustomError {
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
	cro.Features = createFeatures(c.Backend(), changed)

	// gitignore templates are only applied on an initialized repo; names are case-sensitive
	// Bitbucket has no gitignore templates
	if InitializeRepo && c.Backend() != gitoperationslib.BackendBitbucket {
		cro.Gitignores = []string{"Go"}
	}

	repos := c.Repos()
	r, e := repos.Create(ctx, cro)
	switch {
	case e != nil && r == nil:
		return &cerr.CustomError{Title: "Could not create repository", Message: e.Error()}
	case e != nil:
		// The repository exists, but its features could not be set, or the server kept its own value
		// (Forgejo with Actions turned off for the whole instance)
		fmt.Println(hftx.WarningSign("Repository " + hftx.Brown(reponame) + " was created, but: " + e.Error()))
	}

	if !shared.QuietOutput {
		if e == nil {
			fmt.Printf("%s %s has been %s\n", hftx.EnabledSign("Repository"), hftx.Brown(reponame),
				hftx.Green("created"))
		}
		if !InitializeRepo {
			fmt.Println(hftx.WarningSign("NOTICE: The repo is not initialized, you will need to locally set it up (git init/commit/remote add/push)"))
		}
	}
	return nil
}
