// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/collaborators/collaborators.go
// Original timestamp : 2026.11.06 18:00:00

package collaborators

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Owner / organization / workspace; empty means the connection's default (see forge.Client.Target)
var OwnerOrg = ""

var JSONoutput = false
var AssumeYes = false

// collaborator add
var Permission = "write"

func connect(ctx context.Context, repo string) (*forge.Client, gol.Collaborators, string, string, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, nil, "", "", ce
	}
	collabs, ce := c.Collaborators()
	if ce != nil {
		return nil, nil, "", "", ce
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return nil, nil, "", "", ce
	}
	// A 404 further down must be about the user, not the repository
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return nil, nil, "", "", &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound, Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return nil, nil, "", "", forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return c, collabs, owner, repo, nil
}

func newTable(header table.Row) table.Writer {
	t := table.NewWriter()
	t.AppendHeader(header)
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	return t
}

// List lists those with their own access to a repository, not those who have it through a team
func List(repo string, displayoutput bool) ([]gol.Collaborator, *cerr.CustomError) {
	ctx := context.Background()
	_, collabs, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Collaborator], error) {
		return collabs.ListCollaborators(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the collaborators of %s/%s", owner, repo), e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "collaborator list")
	}
	if shared.QuietOutput {
		for _, m := range list {
			fmt.Println(m.Login)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no collaborator\n", owner, repo)
		return list, nil
	}
	// The account ID only exists on Bitbucket
	hasAccount := slices.ContainsFunc(list, func(m gol.Collaborator) bool { return m.AccountID != "" })
	header := table.Row{"Login", "Permission"}
	if hasAccount {
		header = append(header, "Account ID")
	}
	t := newTable(header)
	for _, m := range list {
		row := table.Row{m.Login, string(m.Permission)}
		if hasAccount {
			row = append(row, m.AccountID)
		}
		t.AppendRow(row)
	}
	shared.Page(t.Render(), shared.TableHeader)
	return list, nil
}

// Get shows a collaborator's access; anybody else is an error
func Get(repo, user string, displayoutput bool) (*gol.Collaborator, *cerr.CustomError) {
	ctx := context.Background()
	_, collabs, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	m, e := collabs.GetCollaborator(ctx, owner, repo, user)
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such collaborator", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s is not a collaborator of %s/%s", user, owner, repo)}
		}
		return nil, forge.Error(fmt.Sprintf("Unable to get the collaborator %s", user), e)
	}

	if !displayoutput {
		return m, nil
	}
	if JSONoutput {
		return m, shared.PrintJSON(m, "collaborator")
	}
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Login:"), m.Login)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Permission:"), m.Permission)
	if m.AccountID != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Account ID:"), m.AccountID)
	}
	_ = w.Flush()
	return m, nil
}

// Add gives a user access to a repository, or changes that of an existing collaborator
// GitHub may invite them instead: they then have no access until they accept
func Add(repo, user string) *cerr.CustomError {
	ctx := context.Background()
	perm := gol.CollaboratorPermission(strings.ToLower(Permission))
	if !perm.Valid() {
		return &cerr.CustomError{Title: "Invalid permission", Message: Permission + " is not one of read, write, admin, triage, maintain"}
	}
	_, collabs, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	invited, e := collabs.AddCollaborator(ctx, owner, repo, user, perm)
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to give %s access to %s/%s", user, owner, repo), e)
	}
	if shared.QuietOutput {
		return nil
	}
	if invited {
		fmt.Println(hftx.InfoSign(fmt.Sprintf("Invited %s to %s/%s: they have no access until they accept", hftx.Brown(user), owner, repo)))
		return nil
	}
	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Gave %s %s access to %s/%s", hftx.Brown(user), perm, owner, repo)))
	return nil
}

// Remove takes users' access away, after confirmation; on GitHub, it also withdraws a pending invitation
// Every user is checked before asking, as the forges take removing a stranger for a success
// The run stops at the first failure; the users removed before it stay removed
func Remove(repo string, users []string) *cerr.CustomError {
	ctx := context.Background()
	c, collabs, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}

	var unique []string
	for _, user := range users {
		if slices.Contains(unique, user) {
			continue
		}
		unique = append(unique, user)
		_, e := collabs.GetCollaborator(ctx, owner, repo, user)
		switch {
		case e == nil:
		case !gol.IsNotFound(e):
			return forge.Error(fmt.Sprintf("Unable to get the collaborator %s", user), e)
		default:
			pending, ce := isInvited(ctx, c, owner, repo, user)
			if ce != nil {
				return ce
			}
			if !pending {
				return &cerr.CustomError{Title: "No such collaborator", Code: http.StatusNotFound,
					Message: fmt.Sprintf("%s is not a collaborator of %s/%s", user, owner, repo)}
			}
		}
	}

	answer := "y"
	if !AssumeYes {
		answer = hf.GetStringValFromPrompt(hftx.WarningSign(fmt.Sprintf("Remove the access of %s to %s/%s ? [y/N]: ", strings.Join(unique, ", "), owner, repo)))
	}
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		fmt.Println("Aborted, nobody is removed")
		return nil
	}

	for _, user := range unique {
		if e := collabs.RemoveCollaborator(ctx, owner, repo, user); e != nil {
			return forge.Error(fmt.Sprintf("Unable to remove %s from %s/%s", user, owner, repo), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Removed %s from %s/%s", hftx.Brown(user), owner, repo)))
		}
	}
	return nil
}

// Only GitHub invites: the other forges give access at once
func isInvited(ctx context.Context, c *forge.Client, owner, repo, user string) (bool, *cerr.CustomError) {
	inv, ok := c.Client.(gol.CollaboratorInvitations)
	if !ok {
		return false, nil
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.CollaboratorInvitation], error) {
		return inv.ListCollaboratorInvitations(ctx, owner, repo, o)
	})
	if e != nil {
		return false, forge.Error(fmt.Sprintf("Unable to list the invitations of %s/%s", owner, repo), e)
	}
	return slices.ContainsFunc(list, func(i gol.CollaboratorInvitation) bool { return strings.EqualFold(i.Login, user) }), nil
}

// Invitations lists who was invited and has not accepted yet (GitHub only)
func Invitations(repo string, displayoutput bool) ([]gol.CollaboratorInvitation, *cerr.CustomError) {
	ctx := context.Background()
	c, _, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	inv, ok := c.Client.(gol.CollaboratorInvitations)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: only GitHub has invitations; this forge gives access at once", c.Connection.Name)}
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.CollaboratorInvitation], error) {
		return inv.ListCollaboratorInvitations(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the invitations of %s/%s", owner, repo), e)
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "invitation list")
	}
	if shared.QuietOutput {
		for _, i := range list {
			fmt.Println(i.Login)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no pending invitation\n", owner, repo)
		return list, nil
	}
	t := newTable(table.Row{"Login", "Permission", "Expired", "Created", "URL"})
	for _, i := range list {
		t.AppendRow(table.Row{i.Login, string(i.Permission), i.Expired, i.CreatedAt.Local().Format("2006.01.02 15:04"), i.HTMLURL})
	}
	shared.Page(t.Render(), shared.TableHeader)
	return list, nil
}
