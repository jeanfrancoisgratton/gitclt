// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/users/emails.go
// Original timestamp : 2026.11.04 21:00:00

package users

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListEmails lists the e-mail addresses of the connection's user
func ListEmails(displayOutput bool) ([]gol.Email, *cerr.CustomError) {
	ctx := context.Background()

	_, acct, ce := connectAccount(ctx)
	if ce != nil {
		return nil, ce
	}
	list, e := acct.Emails().List(ctx)
	if e != nil {
		return nil, forge.Error("Unable to fetch the e-mail addresses", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return list, nil
	}

	if JSONoutput {
		return list, shared.PrintJSON(list, "e-mail addresses")
	}
	if shared.QuietOutput {
		for _, m := range list {
			fmt.Println(m.Address)
		}
		return list, nil
	}

	renderEmailTable(list)
	return list, nil
}

// AddEmails adds e-mail addresses to the connection's user; Bitbucket can only list them
func AddEmails(addresses []string) *cerr.CustomError {
	ctx := context.Background()

	c, acct, ce := connectAccount(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkEmailsWritable(c); ce != nil {
		return ce
	}

	added, e := acct.Emails().Add(ctx, addresses...)
	if e != nil {
		return forge.Error("Unable to add the e-mail addresses", e)
	}

	if JSONoutput {
		return shared.PrintJSON(added, "e-mail addresses")
	}
	if !shared.QuietOutput {
		for _, m := range added {
			msg := "Added e-mail address " + hftx.Brown(m.Address)
			if !m.Verified {
				msg += " (not verified yet)"
			}
			fmt.Println(hftx.EnabledSign(msg))
		}
	}
	return nil
}

// DeleteEmails removes e-mail addresses from the connection's user
// Everything is checked before anything is removed: an unknown address, or the primary one, stops the whole run
func DeleteEmails(addresses []string) *cerr.CustomError {
	ctx := context.Background()

	c, acct, ce := connectAccount(ctx)
	if ce != nil {
		return ce
	}
	if ce = checkEmailsWritable(c); ce != nil {
		return ce
	}

	list, e := acct.Emails().List(ctx)
	if e != nil {
		return forge.Error("Unable to fetch the e-mail addresses", e)
	}
	targets, ce := findEmails(list, addresses)
	if ce != nil {
		return ce
	}

	joined := strings.Join(targets, ", ")
	if !confirmed("Delete e-mail address " + hftx.Brown(joined) + " ? [y/N]: ") {
		fmt.Println("Aborted, nothing was deleted")
		return nil
	}

	if e = acct.Emails().Delete(ctx, targets...); e != nil {
		return forge.Error("Unable to delete the e-mail addresses", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted e-mail address " + hftx.Brown(joined)))
	}
	return nil
}

func checkEmailsWritable(c *forge.Client) *cerr.CustomError {
	if c.Backend() == gol.BackendBitbucket {
		return &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: Bitbucket can only list e-mail addresses", c.Connection.Name)}
	}
	return nil
}

// Returns the addresses as the server spells them, as forges compare them in different ways
func findEmails(list []gol.Email, addresses []string) ([]string, *cerr.CustomError) {
	targets := make([]string, 0, len(addresses))
	for _, address := range addresses {
		found := false
		for _, m := range list {
			if !strings.EqualFold(m.Address, address) {
				continue
			}
			if m.Primary {
				return nil, &cerr.CustomError{Title: "Unable to delete the e-mail addresses",
					Message: m.Address + " is the primary address, which cannot be removed; nothing was deleted"}
			}
			targets = append(targets, m.Address)
			found = true
			break
		}
		if !found {
			return nil, &cerr.CustomError{Title: "Unable to find the e-mail address", Code: http.StatusNotFound,
				Message: "There is no address " + address + " on this account; nothing was deleted"}
		}
	}
	return targets, nil
}
