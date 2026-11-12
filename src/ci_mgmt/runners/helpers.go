// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runners/helpers.go
// Original timestamp : 2026.11.11 18:00:00

package runners

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// target is a connection's runners interface, and the scope the command works on
type target struct {
	c       *forge.Client
	runners gol.Runners
	scope   gol.RunnerScope
	where   string // the scope, for the messages
}

// open connects, refuses a forge without runners (Bitbucket), and decides the scope: --org, --user or --instance,
// else the repository (see forge.Client.Target)
func open(ctx context.Context, repo string) (target, *cerr.CustomError) {
	if repo != "" && (OrgScope != "" || UserScope || InstanceScope) {
		return target{}, &cerr.CustomError{Title: "Too many scopes",
			Message: "a repository cannot be given with --org, --user or --instance"}
	}
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return target{}, ce
	}
	if ce = c.Require(gol.CapRunners, "runners are only available on Forgejo and GitHub"); ce != nil {
		return target{}, ce
	}

	t := target{c: c, runners: c.CI().Runners()}
	switch {
	case OrgScope != "":
		t.scope, t.where = gol.OrgRunners(OrgScope), "organization "+OrgScope
	case UserScope:
		t.scope, t.where = gol.UserRunners(), "your account"
	case InstanceScope:
		t.scope, t.where = gol.InstanceRunners(), "the whole instance"
	default:
		owner, name, ce := c.Target(ctx, OwnerOrg, repo)
		if ce != nil {
			return target{}, ce
		}
		t.scope, t.where = gol.RepoRunners(owner, name), owner+"/"+name
	}
	return t, nil
}

// runnerError names what a 404 was about: the runner, or the scope itself (the repository, the organization)
// A scope only reaches its own runners: an organization's runner is a 404 through one of its repositories
func runnerError(title string, e error) *cerr.CustomError {
	switch {
	case gol.IsNotFound(e):
		return &cerr.CustomError{Title: "Not found", Code: http.StatusNotFound,
			Message: title + ": " + e.Error() + " (a runner is only reached through its own scope: --org for an organization's)"}
	case errors.Is(e, gol.ErrNotSupported):
		return &cerr.CustomError{Title: "Unsupported operation", Message: title + ": " + e.Error()}
	}
	return forge.Error(title, e)
}

// parseIDs reads runner IDs; they are positive numbers
func parseIDs(args []string) ([]int64, *cerr.CustomError) {
	ids := make([]int64, len(args))
	for i, a := range args {
		id, e := strconv.ParseInt(a, 10, 64)
		if e != nil || id <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid runner ID", Message: fmt.Sprintf("%q is not a runner ID", a)}
		}
		ids[i] = id
	}
	return ids, nil
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}

// state is the same word on every forge, from Online and Busy; Status stays the forge's own
func state(r gol.Runner) string {
	switch {
	case !r.Online:
		return "offline"
	case r.Busy:
		return "busy"
	}
	return "idle"
}
