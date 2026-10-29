// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/releases/upload.go
// Original timestamp : 2026.10.29 17:20:00

package releases

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"path/filepath"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// A local file to upload, already checked; name is what the asset will be called
type localFile struct {
	path, name string
	size       int64
}

// UploadAssets attaches files to the release of owner/repo's tag (assets belong to releases, never to tags)
// Everything that can be checked is checked before the first upload: the files, the release (created with
// --create, from a tag that must already be on the server), and the asset names already taken (--replace)
// The run stops at the first failed upload; the files uploaded before it stay
func UploadAssets(repo, tag string, paths []string) *cerr.CustomError {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if (DraftRelease || Prerelease) && !CreateRelease {
		return &cerr.CustomError{Title: "Invalid flags", Message: "--draft and --prerelease only apply to a release created with --create"}
	}
	files, ce := checkFiles(paths)
	if ce != nil {
		return ce
	}

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return ce
	}
	if c.Backend() == gol.BackendBitbucket {
		return &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: Bitbucket Cloud has no releases", c.Connection.Name)}
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return ce
	}
	rels := c.Releases()

	rel, ce := findRelease(ctx, c, owner, repo, tag)
	if ce != nil {
		return ce
	}
	if rel == nil {
		if rel, ce = createRelease(ctx, c, owner, repo, tag); ce != nil {
			return ce
		}
	}

	assets, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Asset], error) {
		return rels.ListAssets(ctx, owner, repo, rel.ID, o)
	})
	if e != nil {
		return forge.Error("Unable to list the assets of release "+tag, e)
	}
	existing := map[string]gol.Asset{}
	for _, a := range assets {
		existing[a.Name] = a
	}
	if !ReplaceAssets {
		var taken []string
		for _, f := range files {
			if _, ok := existing[f.name]; ok {
				taken = append(taken, f.name)
			}
		}
		if len(taken) > 0 {
			return &cerr.CustomError{Title: "Assets already exist",
				Message: fmt.Sprintf("release %s already has %s; pass --replace to overwrite", tag, strings.Join(taken, ", "))}
		}
	}

	for _, f := range files {
		if a, ok := existing[f.name]; ok {
			if e = rels.DeleteAsset(ctx, owner, repo, rel.ID, a.ID); e != nil {
				return forge.Error("Unable to replace the asset "+f.name, e)
			}
		}
		if ce = uploadFile(ctx, c, owner, repo, rel.ID, f); ce != nil {
			return ce
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Uploaded %s (%s)", hftx.Brown(f.name), humanSize(f.size))))
		}
	}

	if !shared.QuietOutput && rel.HTMLURL != "" {
		fmt.Printf("Release %s: %s\n", hftx.Green(tag), rel.HTMLURL)
	}
	return nil
}

// checkFiles makes sure every path is a readable regular file, and that no two of them would get the same asset name
func checkFiles(paths []string) ([]localFile, *cerr.CustomError) {
	files := make([]localFile, 0, len(paths))
	seen := map[string]string{}
	for _, p := range paths {
		fi, e := os.Stat(p)
		if e != nil {
			return nil, &cerr.CustomError{Title: "Unable to read " + p, Message: e.Error()}
		}
		if !fi.Mode().IsRegular() {
			return nil, &cerr.CustomError{Title: "Not a file", Message: p + " is not a regular file"}
		}
		name := filepath.Base(p)
		if other, ok := seen[name]; ok {
			return nil, &cerr.CustomError{Title: "Duplicate asset name", Message: fmt.Sprintf("%s and %s would both be uploaded as %s", other, p, name)}
		}
		seen[name] = p
		files = append(files, localFile{path: p, name: name, size: fi.Size()})
	}
	return files, nil
}

// findRelease returns the tag's release, or nil when there is none
// GetByTag never returns drafts (GitHub, Forgejo), so the release list is searched as well: a re-run of
// upload --create --draft must find the draft it created, not create another one
func findRelease(ctx context.Context, c *forge.Client, owner, repo, tag string) (*gol.Release, *cerr.CustomError) {
	rel, e := c.Releases().GetByTag(ctx, owner, repo, tag)
	if e == nil {
		return rel, nil
	}
	if !gol.IsNotFound(e) {
		return nil, forge.Error("Unable to get the release of tag "+tag, e)
	}
	all, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Release], error) {
		return c.Releases().List(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the releases of %s/%s", owner, repo), e)
	}
	for i := range all {
		if all[i].TagName == tag {
			return &all[i], nil
		}
	}
	return nil, nil
}

// createRelease creates the tag's release (--create), titled after the tag; the tag must already be on the server,
// as the library would otherwise create it from the default branch's head
func createRelease(ctx context.Context, c *forge.Client, owner, repo, tag string) (*gol.Release, *cerr.CustomError) {
	if !CreateRelease {
		return nil, &cerr.CustomError{Title: "No such release",
			Message: fmt.Sprintf("%s/%s has no release for tag %s; pass --create to create it", owner, repo, tag)}
	}
	if _, e := c.Tags().Get(ctx, owner, repo, tag); e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such tag",
				Message: fmt.Sprintf("tag %s is not on %s/%s; push it first (git push origin %s)", tag, owner, repo, tag)}
		}
		return nil, forge.Error("Unable to get the tag "+tag, e)
	}

	rel, e := c.Releases().Create(ctx, owner, repo, gol.CreateReleaseOptions{TagName: tag, Name: tag, Draft: DraftRelease, Prerelease: Prerelease})
	if e != nil {
		return nil, forge.Error("Unable to create the release of tag "+tag, e)
	}
	if !shared.QuietOutput {
		kind := "release"
		if Prerelease {
			kind = "prerelease"
		}
		if DraftRelease {
			kind = "draft " + kind
		}
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created %s %s", kind, hftx.Brown(tag))))
	}
	return rel, nil
}

func uploadFile(ctx context.Context, c *forge.Client, owner, repo string, releaseID int64, f localFile) *cerr.CustomError {
	fh, e := os.Open(f.path)
	if e != nil {
		return &cerr.CustomError{Title: "Unable to read " + f.path, Message: e.Error()}
	}
	defer fh.Close()
	// An empty content type: the library guesses it from the extension
	if _, e = c.Releases().UploadAsset(ctx, owner, repo, releaseID, f.name, fh, f.size, ""); e != nil {
		return forge.Error("Unable to upload "+f.name, e)
	}
	return nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
