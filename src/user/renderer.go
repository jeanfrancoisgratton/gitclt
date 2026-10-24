// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/renderer.go
// Original timestamp : 2026.10.24 02:17:25

package user

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
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

// printJSON encodes and pretty-prints any of the user types; what is only used in the error titles
func printJSON(v any, what string) *cerr.CustomError {
	jStream, jerr := json.Marshal(v, jsontext.WithIndent("\t"))
	if jerr != nil {
		return &cerr.CustomError{Title: "Unable to encode the " + what, Message: jerr.Error()}
	}
	if e := hfjson.Print(jStream); e != nil {
		return &cerr.CustomError{Title: "Unable to print the " + what, Message: e.Error()}
	}
	return nil
}

// printUser writes the base profile shared by gol.User and gol.CurrentUser.
// With ExtendedInfo, the forge-specific fields of gol.User follow (only those the forge reported)
func printUser(w *tabwriter.Writer, u *gol.User) {
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("ID:"), u.ID)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Login:"), u.Login)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Full name:"), u.FullName)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("E-mail:"), u.Email)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Description:"), u.Description)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Web site:"), u.Website)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Location:"), u.Location)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Visibility:"), u.Visibility)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Admin:"), u.IsAdmin)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Active:"), u.Active)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Restricted:"), u.Restricted)
	_, _ = fmt.Fprintf(w, "%s\t%t\n", hftx.Blue("Login prohibited:"), u.ProhibitLogin)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("HTML URL:"), u.HTMLURL)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Avatar URL:"), u.AvatarURL)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Followers:"), u.Followers)
	_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Following:"), u.Following)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Created:"), formatTime(u.CreatedAt))
	_, _ = fmt.Fprintf(w, "%s\t%s\n", hftx.Blue("Last login:"), formatTime(u.LastLogin))

	if !ExtendedInfo {
		return
	}
	// forge-specific scalars: skip the zero values, as they mostly mean "not reported by this forge"
	printIfSet(w, "Company:", u.Company)
	printIfSet(w, "Pronouns:", u.Pronouns)
	printIfSet(w, "Language:", u.Language)
	if u.StarredRepos != 0 {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", hftx.Blue("Starred repos:"), u.StarredRepos)
	}
}

// printExtendedInfo writes what only gol.CurrentUser carries: the forge-specific fields (only those the forge
// reported), settings, e-mails, organisations, teams, repositories and keys
func printExtendedInfo(w *tabwriter.Writer, cu *gol.CurrentUser) {
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

// printMissing reports the parts of a gol.CurrentUser that could not be read (most likely a token scope issue),
// instead of silently showing nothing
func printMissing(missing []gol.PartError) {
	if len(missing) == 0 {
		return
	}
	fmt.Println()
	fmt.Println(hftx.Yellow("Some information could not be fetched:"))
	for _, m := range missing {
		if m.Status != 0 {
			fmt.Printf("  - %s (HTTP %d): %s\n", m.Part, m.Status, m.Message)
		} else {
			fmt.Printf("  - %s: %s\n", m.Part, m.Message)
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
