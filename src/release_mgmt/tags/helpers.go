// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/helpers.go
// Original timestamp : 2026.10.29 22:30:00

package tags

import (
	"cmp"
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

// connect opens the client on the selected connection, and decides the repositories' owner (-o, or the connection's default)
func connect(ctx context.Context) (*forge.Client, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", ce
	}
	owner, ce := c.Owner(ctx, OwnerOrg)
	if ce != nil {
		return nil, "", ce
	}
	return c, owner, nil
}

// checkRepo makes sure owner/repo exists, so that a 404 further down is about the tag, not the repository
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

// releasesByTag lists owner/repo's releases, drafts included, by tag name; Bitbucket has none (an empty map)
func releasesByTag(ctx context.Context, c *forge.Client, owner, repo string) (map[string]gol.Release, *cerr.CustomError) {
	byTag := map[string]gol.Release{}
	if c.Backend() == gol.BackendBitbucket {
		return byTag, nil
	}
	all, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Release], error) {
		return c.Releases().List(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the releases of %s/%s", owner, repo), e)
	}
	for _, r := range all {
		byTag[r.TagName] = r
	}
	return byTag, nil
}

// compareVersions orders tag names with their digit runs compared as numbers: v0.10.0 comes after v0.9.0
func compareVersions(a, b string) int {
	for a != "" && b != "" {
		da, db := digits(a), digits(b)
		switch {
		case da > 0 && db > 0:
			// Leading zeros aside, the longer number is the larger
			na, nb := strings.TrimLeft(a[:da], "0"), strings.TrimLeft(b[:db], "0")
			if c := cmp.Compare(len(na), len(nb)); c != 0 {
				return c
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			a, b = a[da:], b[db:]
		case a[0] != b[0]:
			return cmp.Compare(a[0], b[0])
		default:
			a, b = a[1:], b[1:]
		}
	}
	return cmp.Compare(len(a), len(b))
}

func digits(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

// kind words what a release is: draft and prerelease can go together
func kind(r gol.Release) string {
	k := "release"
	if r.Prerelease {
		k = "prerelease"
	}
	if r.Draft {
		k = "draft " + k
	}
	return k
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
