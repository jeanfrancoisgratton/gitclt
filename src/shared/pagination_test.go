// gitclt
// Tests that FetchAll walks every page, and stops on the first error

package shared

import (
	"context"
	"errors"
	"slices"
	"testing"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// A fake list call serving pages; NextPage = 0 marks the last one
func pager(pages [][]int, failOn int) func(context.Context, gol.ListOptions) (gol.Page[int], error) {
	return func(_ context.Context, o gol.ListOptions) (gol.Page[int], error) {
		page := max(o.Page, 1)
		if page == failOn {
			return gol.Page[int]{}, errors.New("boom")
		}
		p := gol.Page[int]{Items: pages[page-1]}
		if page < len(pages) {
			p.NextPage = page + 1
		}
		return p, nil
	}
}

func TestFetchAll(t *testing.T) {
	got, err := FetchAll(context.Background(), pager([][]int{{1, 2}, {3, 4}, {5}}, 0))
	if err != nil || !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestFetchAll_AsksForListPageSize(t *testing.T) {
	_, _ = FetchAll(context.Background(), func(_ context.Context, o gol.ListOptions) (gol.Page[int], error) {
		if o.PerPage != ListPageSize {
			t.Errorf("PerPage = %d, want %d", o.PerPage, ListPageSize)
		}
		return gol.Page[int]{}, nil
	})
}

// A failure on a later page must not return a partial list as if it were complete
func TestFetchAll_ErrorOnSecondPage(t *testing.T) {
	got, err := FetchAll(context.Background(), pager([][]int{{1, 2}, {3}}, 2))
	if err == nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}
