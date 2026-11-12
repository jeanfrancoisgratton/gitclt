// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/target.go
// Original timestamp : 2026.11.05 12:00:00

package forge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/jeanfrancoisgratton/gitoperationslib/local"
)

// Decides which repository a command works on, and who owns it (flag being -o, when given)
// Without a repository name, it is the one the current directory is a clone of, on this connection's server;
// -o still wins over the clone's owner
// With a name, the owner follows Owner(), unless that name is the current clone's: its remote then tells the owner
func (c *Client) Target(ctx context.Context, flag, repo string) (string, string, *cerr.CustomError) {
	var here *local.RemoteRepo
	cwd, e := os.Getwd()
	if e == nil {
		here, e = inferRepo(cwd, c.Connection.URL)
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

// inferRepo finds the repository that the clone holding dir is of, on the server at serverURL (see local.Repo.RepoOn);
// with no remote on that server, the connection is likely the wrong one
func inferRepo(dir, serverURL string) (*local.RemoteRepo, error) {
	repo, err := local.Open(dir)
	if err != nil {
		return nil, err
	}
	defer repo.Close()
	r, err := repo.RepoOn(serverURL)
	if errors.Is(err, local.ErrNoRemoteOnHost) {
		err = fmt.Errorf("%w; choose the matching connection with -c", err)
	}
	return r, err
}
