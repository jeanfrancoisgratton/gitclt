// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/history/helpers.go
// Original timestamp : 2026.10.30 15:10:00

package history

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// connect opens the client on the selected connection, decides the repository and its owner (see forge.Client.Target),
// and finds the commit history interface
func connect(ctx context.Context, repo string) (*forge.Client, string, string, gol.History, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, "", "", nil, ce
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return nil, "", "", nil, ce
	}
	h, ce := c.History()
	if ce != nil {
		return nil, "", "", nil, ce
	}
	return c, owner, repo, h, nil
}

// checkRepo makes sure owner/repo exists, so that a 404 further down is about the reference, not the repository
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

// parseDate reads --since / --until: a day (local time) or an RFC 3339 timestamp; empty means no bound
// A day given as an upper bound (endOfDay) includes the whole day
func parseDate(s string, endOfDay bool) (time.Time, *cerr.CustomError) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return t, nil
	}
	t, e := time.ParseInLocation(time.DateOnly, s, time.Local)
	if e != nil {
		return time.Time{}, &cerr.CustomError{Title: "Invalid date", Message: s + " is neither YYYY-MM-DD nor an RFC 3339 timestamp"}
	}
	if endOfDay {
		t = t.AddDate(0, 0, 1).Add(-time.Second)
	}
	return t, nil
}
