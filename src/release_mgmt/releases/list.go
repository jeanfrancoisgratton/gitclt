// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/list.go
// Original timestamp : 2026.10.29 13:00:46

package releases

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListReleases lists the releases of owner/reponame, drafts included, in the server's order (newest first)
func ListReleases(reponame string, displayoutput bool) ([]gol.Release, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, reponame); ce != nil {
		return nil, ce
	}

	rels, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Release], error) {
		return c.Releases().List(ctx, owner, reponame, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the releases of %s/%s", owner, reponame), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return rels, nil
	}
	if JSONoutput {
		return rels, shared.PrintJSON(rels, "release list")
	}
	if len(rels) == 0 {
		if !shared.QuietOutput {
			fmt.Printf("%s/%s has no release\n", owner, reponame)
		}
		return rels, nil
	}
	renderReleaseTable(rels)
	return rels, nil
}

// ListAssets lists the files attached to the release of owner/reponame's tag
func ListAssets(reponame, tag string, displayoutput bool) ([]gol.Asset, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, reponame); ce != nil {
		return nil, ce
	}
	rel, ce := findRelease(ctx, c, owner, reponame, tag)
	if ce != nil {
		return nil, ce
	}
	if rel == nil {
		return nil, &cerr.CustomError{Title: "No such release", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no release for tag %s", owner, reponame, tag)}
	}

	assets, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Asset], error) {
		return c.Releases().ListAssets(ctx, owner, reponame, rel.ID, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to list the assets of release "+tag, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return assets, nil
	}
	if JSONoutput {
		return assets, shared.PrintJSON(assets, "asset list")
	}
	if len(assets) == 0 {
		if !shared.QuietOutput {
			fmt.Printf("Release %s has no asset\n", tag)
		}
		return assets, nil
	}
	renderAssetTable(assets)
	return assets, nil
}

// connect opens the client on the selected connection, refuses Bitbucket (no releases there), and decides the
// repositories' owner (-o, or the connection's default)
func connect(ctx context.Context) (*forge.Client, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", ce
	}
	if c.Backend() == gol.BackendBitbucket {
		return nil, "", &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: Bitbucket Cloud has no releases", c.Connection.Name)}
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return nil, "", ce
	}
	return c, owner, nil
}

// checkRepo makes sure owner/repo exists, so that a 404 further down is about the release, not the repository
func checkRepo(ctx context.Context, c *forge.Client, owner, repo string) *cerr.CustomError {
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return nil
}
