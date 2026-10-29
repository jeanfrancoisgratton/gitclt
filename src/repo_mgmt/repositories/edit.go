// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/repositories/edit.go
// Original timestamp : 2026.10.29 15:10:00

package repositories

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// EditRepo only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
// Bitbucket cannot archive a repository, nor make it a template: the library refuses those flags there
func EditRepo(reponame string, changed func(string) bool) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	ero := editOptionsFromFlags(changed)
	if ero == (gol.EditRepoOptions{}) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt repo edit --help"}
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return ce
	}

	r, e := c.Repos().Edit(ctx, owner, reponame, ero)
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to edit the repository %s/%s", owner, reponame), e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated repository " + hftx.Brown(owner+"/"+r.Name)))
	}
	return nil
}

func editOptionsFromFlags(changed func(string) bool) gol.EditRepoOptions {
	var ero gol.EditRepoOptions

	strFlags := []struct {
		name string
		val  string
		dst  **string
	}{
		{"name", EditRepoName, &ero.Name},
		{"description", EditRepoDescription, &ero.Description},
		{"website", EditRepoWebsite, &ero.Website},
		{"default-branch", EditRepoDefaultBranch, &ero.DefaultBranch},
	}
	for _, f := range strFlags {
		if changed(f.name) {
			*f.dst = gol.Ptr(f.val)
		}
	}

	if changed("public") {
		ero.Visibility = gol.Ptr(gol.VisibilityPrivate)
		if EditRepoPublic {
			ero.Visibility = gol.Ptr(gol.VisibilityPublic)
		}
	}
	if changed("archived") {
		ero.Archived = gol.Ptr(EditRepoArchived)
	}
	if changed("template") {
		ero.Template = gol.Ptr(EditRepoTemplate)
	}

	return ero
}
