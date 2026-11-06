// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/target.go
// Original timestamp : 2026.11.05 12:00:00

package forge

import (
	"context"
	"gitclt/shared"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Decides which repository a command works on, and who owns it (flag being -o, when given)
// Without a repository name, it is the one the current directory is a clone of, on this connection's server;
// -o still wins over the clone's owner
// With a name, the owner follows Owner(), unless that name is the current clone's: its remote then tells the owner
func (c *Client) Target(ctx context.Context, flag, repo string) (string, string, *cerr.CustomError) {
	var here *shared.LocalRepo
	cwd, e := os.Getwd()
	if e == nil {
		here, e = shared.InferRepo(cwd, c.Connection.URL)
	}

	if repo == "" {
		if e != nil {
			return "", "", &cerr.CustomError{Title: "Missing repository",
				Message: "no repository given, and it cannot be told from the current directory: " + e.Error()}
		}
		if flag != "" {
			return flag, here.Name, nil
		}
		return here.Owner, here.Name, nil
	}

	if flag == "" && e == nil && strings.EqualFold(here.Name, repo) {
		return here.Owner, repo, nil
	}
	owner, ce := c.Owner(ctx, flag)
	return owner, repo, ce
}
