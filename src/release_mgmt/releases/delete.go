// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/delete.go
// Original timestamp : 2026.10.29 19:10:00

package releases

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteReleases deletes the releases of owner/repo's tags, drafts included; the tags themselves stay
// Every tag is checked before asking; the first failure stops the run, and the releases before it stay deleted
func DeleteReleases(repo string, tags []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	tags = unique(tags)

	// One listing finds every release, drafts included (GetByTag does not see them)
	all, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Release], error) {
		return c.Releases().List(ctx, owner, repo, o)
	})
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to list the releases of %s/%s", owner, repo), e)
	}
	byTag := map[string]gol.Release{}
	for _, r := range all {
		byTag[r.TagName] = r
	}
	var missing []string
	for _, tag := range tags {
		if _, ok := byTag[tag]; !ok {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		return &cerr.CustomError{Title: "No such release", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no release for %s", owner, repo, strings.Join(missing, ", "))}
	}

	if !confirm(fmt.Sprintf("Delete the release(s) of %s, with their assets (the tags stay) ? [y/N]: ", strings.Join(tags, ", "))) {
		fmt.Println("Aborted, no release is deleted")
		return nil
	}

	for _, tag := range tags {
		if e = c.Releases().Delete(ctx, owner, repo, byTag[tag].ID); e != nil {
			return forge.Error("Unable to delete the release of tag "+tag, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted release " + hftx.Brown(tag)))
		}
	}
	return nil
}

// DeleteAssets deletes files attached to the release of owner/repo's tag
// Every name is checked before asking; the first failure stops the run, and the assets before it stay deleted
func DeleteAssets(repo, tag string, names []string) *cerr.CustomError {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return ce
	}
	names = unique(names)

	rel, ce := findRelease(ctx, c, owner, repo, tag)
	if ce != nil {
		return ce
	}
	if rel == nil {
		return &cerr.CustomError{Title: "No such release", Code: http.StatusNotFound,
			Message: fmt.Sprintf("%s/%s has no release for tag %s", owner, repo, tag)}
	}
	assets, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Asset], error) {
		return c.Releases().ListAssets(ctx, owner, repo, rel.ID, o)
	})
	if e != nil {
		return forge.Error("Unable to list the assets of release "+tag, e)
	}
	byName := map[string]gol.Asset{}
	for _, a := range assets {
		byName[a.Name] = a
	}
	var missing []string
	for _, name := range names {
		if _, ok := byName[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return &cerr.CustomError{Title: "No such asset", Code: http.StatusNotFound,
			Message: fmt.Sprintf("release %s has no %s", tag, strings.Join(missing, ", "))}
	}

	if !confirm(fmt.Sprintf("Delete %s from release %s ? [y/N]: ", strings.Join(names, ", "), tag)) {
		fmt.Println("Aborted, no asset is deleted")
		return nil
	}

	for _, name := range names {
		if e = c.Releases().DeleteAsset(ctx, owner, repo, rel.ID, byName[name].ID); e != nil {
			return forge.Error("Unable to delete the asset "+name, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted asset " + hftx.Brown(name)))
		}
	}
	return nil
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}

// unique drops repeated arguments, keeping the first occurrence's order
func unique(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}
