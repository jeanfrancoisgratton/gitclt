// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/whoami.go
// Original timestamp : 2026.10.23 12:59:01

package user

import (
	"context"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"text/tabwriter"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// WhoAmI provides information about the user tied to the current connection's credentials.
// The base profile is always shown; ExtendedInfo adds the forge-specific fields, settings, e-mails,
// organisations, teams, repositories and keys.
func WhoAmI(displayOutput bool) (*gol.CurrentUser, *cerr.CustomError) {
	var c *forge.Client
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}

	cu, e := c.Directory().Me(ctx)
	if e != nil {
		return nil, forge.Error("Unable to fetch user information", e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return cu, nil
	}

	if JSONoutput {
		return cu, printJSON(cu, "user information")
	}

	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	printUser(w, &cu.User)
	if ExtendedInfo {
		printExtendedInfo(w, cu)
	}
	_ = w.Flush()
	printMissing(cu.Missing)

	return cu, nil
}
