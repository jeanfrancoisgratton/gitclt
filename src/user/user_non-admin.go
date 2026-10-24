// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/user_non-admin.go
// Original timestamp : 2026.10.23 12:59:01

package user

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

const timeFormat = "2006.01.02 15:04"

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
		jStream, jerr := json.Marshal(cu, jsontext.WithIndent("\t"))
		if jerr != nil {
			return nil, &cerr.CustomError{Title: "Unable to encode the user information", Message: jerr.Error()}
		}
		if e := hfjson.Print(jStream); e != nil {
			return cu, &cerr.CustomError{Title: "Unable to print the user information", Message: e.Error()}
		}
		return cu, nil
	}

	// tabulated output
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), cu.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Login:"), cu.Login)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Full name:"), cu.FullName)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("E-mail:"), cu.Email)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), cu.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Web site:"), cu.Website)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Location:"), cu.Location)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Visibility:"), cu.Visibility)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Admin:"), cu.IsAdmin)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Active:"), cu.Active)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Restricted:"), cu.Restricted)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Login prohibited:"), cu.ProhibitLogin)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("HTML URL:"), cu.HTMLURL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Avatar URL:"), cu.AvatarURL)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Followers:"), cu.Followers)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Following:"), cu.Following)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), formatTime(cu.CreatedAt))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Last login:"), formatTime(cu.LastLogin))

	if ExtendedInfo {
		printExtendedInfo(w, cu)
	}
	_ = w.Flush()

	// some parts could not be read (most likely a token scope issue); say so instead of silently showing nothing
	if len(cu.Missing) > 0 {
		fmt.Println()
		fmt.Println(hftx.Yellow("Some information could not be fetched:"))
		for _, m := range cu.Missing {
			if m.Status != 0 {
				fmt.Printf("  - %s (HTTP %d): %s\n", m.Part, m.Status, m.Message)
			} else {
				fmt.Printf("  - %s: %s\n", m.Part, m.Message)
			}
		}
	}

	return cu, nil
}

// printExtendedInfo writes the forge-specific fields (only those the forge reported) and the user's collections
func printExtendedInfo(w *tabwriter.Writer, cu *gol.CurrentUser) {
	// forge-specific scalars: skip the zero values, as they mostly mean "not reported by this forge"
	printIfSet(w, "Company:", cu.Company)
	printIfSet(w, "Pronouns:", cu.Pronouns)
	printIfSet(w, "Language:", cu.Language)
	printIfSet(w, "Login name:", cu.LoginName)
	printIfSet(w, "UUID:", cu.UUID)
	printIfSet(w, "Account ID:", cu.AccountID)
	printIfSet(w, "Account type:", cu.Type)
	printIfSet(w, "Plan:", cu.Plan)
	printIfSet(w, "Twitter:", cu.TwitterUsername)
	if cu.AuthSourceID != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Auth source ID:"), cu.AuthSourceID)
	}
	if cu.TwoFactorAuth != nil {
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("2FA enabled:"), *cu.TwoFactorAuth)
	}
	if cu.Hireable {
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Hireable:"), cu.Hireable)
	}
	if cu.StarredRepos != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Starred repos:"), cu.StarredRepos)
	}
	if cu.PublicRepos != 0 || cu.TotalPrivateRepos != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d public, %d private (%d owned)\n", hftx.Blue("Repositories:"),
			cu.PublicRepos, cu.TotalPrivateRepos, cu.OwnedPrivateRepos)
	}
	if cu.PublicGists != 0 || cu.PrivateGists != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d public, %d private\n", hftx.Blue("Gists:"), cu.PublicGists, cu.PrivateGists)
	}
	if cu.DiskUsage != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d KiB\n", hftx.Blue("Disk usage:"), cu.DiskUsage)
	}
	if cu.Collaborators != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Collaborators:"), cu.Collaborators)
	}
	if !cu.UpdatedAt.IsZero() {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Updated:"), formatTime(cu.UpdatedAt))
	}

	if s := cu.Settings; s != nil {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("Settings"))
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Theme:"), s.Theme)
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Diff view style:"), s.DiffViewStyle)
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Hide e-mail:"), s.HideEmail)
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Hide activity:"), s.HideActivity)
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Hide pronouns:"), s.HidePronouns)
		_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Repo unit hints:"), s.EnableRepoUnitHints)
	}

	if len(cu.Emails) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("E-mail addresses"))
		for _, m := range cu.Emails {
			_, _ = fmt.Fprintf(w, "  %s\t%s\n", m.Address, flags(map[string]bool{
				"primary": m.Primary, "verified": m.Verified}, m.Visibility))
		}
	}

	if len(cu.Orgs) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("Organisations"))
		for _, o := range cu.Orgs {
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", o.Name, o.FullName, o.Visibility)
		}
	}

	if len(cu.Teams) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("Teams"))
		for _, t := range cu.Teams {
			_, _ = fmt.Fprintf(w, "  %s/%s\t%s\t%s\n", t.Org, t.Name, t.Permission, flags(map[string]bool{
				"all repos": t.IncludesAllRepos, "can create repos": t.CanCreateOrgRepo}))
		}
	}

	if len(cu.Repos) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s (%d)\n", hftx.Green("Repositories"), len(cu.Repos))
		for _, r := range cu.Repos {
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", r.FullName, r.Visibility, flags(map[string]bool{
				"fork": r.Fork, "mirror": r.Mirror, "archived": r.Archived, "template": r.Template}))
		}
	}

	if len(cu.SSHKeys) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("SSH keys"))
		for _, k := range cu.SSHKeys {
			_, _ = fmt.Fprintf(w, "  %s\t%s\tcreated %s\tlast used %s\t%s\n", k.Title, k.Fingerprint,
				formatTime(k.CreatedAt), formatTime(k.LastUsed), flags(map[string]bool{"read-only": k.ReadOnly}))
		}
	}

	if len(cu.GPGKeys) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", hftx.Green("GPG keys"))
		for _, k := range cu.GPGKeys {
			addrs := make([]string, 0, len(k.Emails))
			for _, m := range k.Emails {
				addrs = append(addrs, m.Address)
			}
			expires := "never"
			if !k.ExpiresAt.IsZero() {
				expires = formatTime(k.ExpiresAt)
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\texpires %s\t%s\n", k.KeyID, strings.Join(addrs, ", "), expires,
				flags(map[string]bool{"sign": k.CanSign, "certify": k.CanCertify,
					"encrypt": k.CanEncryptComms || k.CanEncryptStorage, "verified": k.Verified}))
		}
	}
}

func printIfSet(w *tabwriter.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(label), value)
	}
}

// formatTime returns "-" for a zero time (never happened / not reported) instead of 0001.01.01
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format(timeFormat)
}

// flags turns a set of booleans into a sorted, comma-separated list of the true ones, plus any non-empty extras
func flags(set map[string]bool, extras ...string) string {
	var out []string
	for k, v := range set {
		if v {
			out = append(out, k)
		}
	}
	for _, x := range extras {
		if x != "" {
			out = append(out, x)
		}
	}
	slices.Sort(out)
	return strings.Join(out, ", ")
}
