// gitclt
// Tests the repo ls sort order (-s descending, -S ascending)

package repositories

import (
	"slices"
	"testing"
	"time"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func TestSortRepos(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	repos := []gol.Repo{
		{ID: 10, Owner: "b", Name: "Zeta", FullName: "b/Zeta", CreatedAt: day(2)},
		{ID: 2, Owner: "a", Name: "alpha", FullName: "a/alpha", CreatedAt: day(3)},
		{ID: 7, Owner: "b", Name: "beta", FullName: "b/beta", CreatedAt: day(1)},
	}
	for _, tc := range []struct {
		name, desc, asc string
		hasIDs          bool
		want            []string
	}{
		{"default: ID, ascending", "", "", true, []string{"a/alpha", "b/beta", "b/Zeta"}},
		{"ID, descending (numeric)", "id", "", true, []string{"b/Zeta", "b/beta", "a/alpha"}},
		{"ID, ascending (numeric)", "", "id", true, []string{"a/alpha", "b/beta", "b/Zeta"}},
		{"no IDs: repository, ascending", "", "", false, []string{"a/alpha", "b/beta", "b/Zeta"}},
		{"repository, ascending, case-insensitive", "", "r", true, []string{"a/alpha", "b/beta", "b/Zeta"}},
		{"created, descending", "c", "", true, []string{"a/alpha", "b/Zeta", "b/beta"}},
		{"owner, descending, ties on full name", "o", "", true, []string{"b/beta", "b/Zeta", "a/alpha"}},
	} {
		SortBy, SortByAsc = tc.desc, tc.asc
		got := slices.Clone(repos)
		sortRepos(got, tc.hasIDs)
		var names []string
		for _, r := range got {
			names = append(names, r.FullName)
		}
		if !slices.Equal(names, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, names, tc.want)
		}
	}
	SortBy, SortByAsc = "", ""
}

func TestYesNo(t *testing.T) {
	if yesNo(true) != "yes" || yesNo(false) != "no" {
		t.Fatal("wrong yesNo")
	}
}
