// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/organizations/get.go
// Original timestamp : 2026.10.24 23:07:23

package organizations

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"text/tabwriter"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func OrgGet(orgname string, displayoutput bool) (*gol.Org, *cerr.CustomError) {
	var c *forge.Client
	var adm gol.Admin
	var ce *cerr.CustomError
	ctx := context.Background()

	if c, ce = forge.Connect(ctx, shared.ConnectionName); ce != nil {
		return nil, ce
	}
	if adm, ce = c.Admin(); ce != nil {
		return nil, ce
	}

	oo, e := adm.Orgs().Get(ctx, orgname)
	if e != nil {
		return nil, forge.Error("Unable to fetch organization "+orgname, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return oo, nil
	}

	if JSONoutput {
		return oo, shared.PrintJSON(oo, "organization information")
	}

	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), oo.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Name:"), oo.Name)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Full name:"), oo.FullName)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), oo.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Website:"), oo.Website)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Location:"), oo.Location)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Email address:"), oo.Email)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Avatar URL:"), oo.AvatarURL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Visibility:"), oo.Visibility)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("HTML URL:"), oo.HTMLURL)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Repo owners can change team access:"), oo.RepoAdminChangeTeamAccess)
	_ = w.Flush()

	return oo, nil
}
