// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/pagination.go
// Original timestamp : 2026.10.24 02:43:40

package shared

import (
	"context"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Forgejo's maximum page size is a server setting (50 by default); GitHub's and Bitbucket's are 100
const ListPageSize = 50

// FetchAll collects every item of a paginated list call. The forge only returns one page per call;
// gol.All keeps requesting the next page until the forge reports there is none.
// fetch wraps the list call, e.g.:
//
//	func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.User], error) { return adm.Users().List(ctx, o) }
func FetchAll[T any](ctx context.Context, fetch func(context.Context, gol.ListOptions) (gol.Page[T], error)) ([]T, error) {
	var items []T
	for item, err := range gol.All(ctx, gol.ListOptions{PerPage: ListPageSize}, fetch) {
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
