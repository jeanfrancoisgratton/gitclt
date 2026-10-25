// gitclt
// Tests the user renderers: base profile, extended info, missing parts, and the small formatting helpers

package users

import (
	"bytes"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"gitclt/testutil"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func render(f func(w *tabwriter.Writer)) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 1, 4, 2, ' ', 0)
	f(w)
	_ = w.Flush()
	return testutil.StripANSI(b.String())
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(time.Time{}); got != "-" {
		t.Errorf("zero: got %q", got)
	}
	d := time.Date(2026, 10, 24, 13, 5, 0, 0, time.Local)
	if got := formatTime(d); got != "2026.10.24 13:05" {
		t.Errorf("got %q", got)
	}
}

func TestFlags(t *testing.T) {
	if got := flags(map[string]bool{"b": true, "a": true, "off": false}, "", "c"); got != "a, b, c" {
		t.Errorf("got %q", got)
	}
	if got := flags(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestPrintUser(t *testing.T) {
	testutil.Set(t, &ExtendedInfo, false)
	u := &gol.User{ID: 4, Login: "jfg", FullName: "J.F. Gratton", IsAdmin: true, Company: "ACME", StarredRepos: 3}

	out := render(func(w *tabwriter.Writer) { printUser(w, u) })
	for _, want := range []string{"Login:", "jfg", "J.F. Gratton", "Admin:", "true", "Last login:", "-"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "ACME") {
		t.Errorf("forge-specific fields shown without --extended:\n%s", out)
	}

	// --extended: only the forge-specific fields that are set
	ExtendedInfo = true
	out = render(func(w *tabwriter.Writer) { printUser(w, u) })
	if !strings.Contains(out, "ACME") || !strings.Contains(out, "Starred repos:") || strings.Contains(out, "Pronouns:") {
		t.Errorf("extended:\n%s", out)
	}
}

func TestPrintExtendedInfo(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	cu := &gol.CurrentUser{
		LoginName:     "jfg-ldap",
		AuthSourceID:  2,
		TwoFactorAuth: gol.Ptr(true),
		PublicRepos:   4, TotalPrivateRepos: 2, OwnedPrivateRepos: 1,
		Settings: &gol.UserSettings{Theme: "forgejo-dark", HideEmail: true},
		Emails:   []gol.Email{{Address: "jfg@example.com", Primary: true, Verified: true}},
		Orgs:     []gol.Org{{Name: "acme", FullName: "ACME Inc."}},
		Teams:    []gol.Team{{Org: "acme", Name: "devs", IncludesAllRepos: true}},
		Repos:    []gol.Repo{{FullName: "jfg/gitclt", Fork: true}},
		SSHKeys:  []gol.SSHKey{{Title: "laptop", Fingerprint: "SHA256:abc", CreatedAt: day, ReadOnly: true}},
		GPGKeys:  []gol.GPGKey{{KeyID: "ABCD1234", Emails: []gol.GPGKeyEmail{{Address: "jfg@example.com"}}, CanSign: true}},
	}

	out := render(func(w *tabwriter.Writer) { printExtendedInfo(w, cu) })
	for _, want := range []string{"Login name:", "jfg-ldap", "Auth source ID:", "2FA enabled:", "4 public, 2 private (1 owned)",
		"Settings", "forgejo-dark", "E-mail addresses", "primary, verified", "Organisations", "ACME Inc.",
		"Teams", "acme/devs", "all repos", "Repositories (1)", "jfg/gitclt", "fork",
		"SSH keys", "SHA256:abc", "last used -", "read-only", "GPG keys", "ABCD1234", "expires never", "sign"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Unset counters are not shown
	for _, unwanted := range []string{"Gists:", "Disk usage:", "Hireable:", "Updated:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, out)
		}
	}

	if out = render(func(w *tabwriter.Writer) { printExtendedInfo(w, &gol.CurrentUser{}) }); out != "" {
		t.Errorf("an empty user prints %q", out)
	}
}

func TestPrintMissing(t *testing.T) {
	if out := testutil.Run(t, func() { printMissing(nil) }); out != "" {
		t.Fatalf("nothing missing prints %q", out)
	}
	out := testutil.Run(t, func() {
		printMissing([]gol.PartError{{Part: "SSH keys", Status: 403, Message: "forbidden"}, {Part: "teams", Message: "not supported"}})
	})
	for _, want := range []string{"could not be fetched", "SSH keys (HTTP 403): forbidden", "- teams: not supported"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
