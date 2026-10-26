// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repositories/helpers.go
// Original timestamp : 2026.10.20 10:03:57

package repositories

import (
	"cmp"
	"context"
	"gitclt/forge"
	"gitclt/shared"
	"slices"
	"strings"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// fetchRepos collects every page of repositories for owner.
// An empty owner lists every repository the authenticated user can access (including their orgs' repositories),
// otherwise the forge only returns that user's or organization's repositories (the filter is applied server-side)
func fetchRepos(ctx context.Context, c *forge.Client, owner string) ([]gol.Repo, error) {
	return shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Repo], error) {
		return c.Repos().List(ctx, owner, o)
	})
}

// sortRepos sorts repos in place, on the column given by -s (descending) or -S (ascending)
// Neither flag means -S ID; an unknown or empty column means ID, or Repository when the forge has no IDs (Bitbucket)
// Ties are broken on the full name (ascending), so that the order does not depend on the forge's
func sortRepos(repos []gol.Repo, hasIDs bool) {
	key, asc := SortBy, false
	if SortBy == "" {
		key, asc = SortByAsc, true
	}

	var compare func(a, b gol.Repo) int
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "owner", "organization", "org", "o":
		compare = func(a, b gol.Repo) int { return cmp.Compare(strings.ToLower(a.Owner), strings.ToLower(b.Owner)) }
	case "repository", "repo", "r":
		compare = func(a, b gol.Repo) int { return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }
	case "created", "creation", "c":
		compare = func(a, b gol.Repo) int { return a.CreatedAt.Compare(b.CreatedAt) }
	case "updated", "update", "u", "last", "l":
		compare = func(a, b gol.Repo) int { return a.UpdatedAt.Compare(b.UpdatedAt) }
	default:
		if hasIDs {
			compare = func(a, b gol.Repo) int { return cmp.Compare(a.ID, b.ID) }
		} else {
			compare = func(a, b gol.Repo) int { return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }
		}
	}

	slices.SortFunc(repos, func(a, b gol.Repo) int {
		c := compare(a, b)
		if !asc {
			c = -c
		}
		if c == 0 {
			c = cmp.Compare(strings.ToLower(a.FullName), strings.ToLower(b.FullName))
		}
		return c
	})
}
