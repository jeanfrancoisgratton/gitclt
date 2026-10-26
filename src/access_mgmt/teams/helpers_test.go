// gitclt
// Tests the --units parser, --permission, and how both combine into a team access

package teams

import (
	"testing"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

func TestParseUnits(t *testing.T) {
	units, err := parseUnits(" code=write, repo.Issues = READ ,,ext_wiki=none")
	if err != nil || len(units) != 3 || units[gol.TeamUnitCode] != gol.TeamPermissionWrite ||
		units[gol.TeamUnitIssues] != gol.TeamPermissionRead || units[gol.TeamUnitExternalWiki] != gol.TeamPermissionNone {
		t.Fatalf("got %v, %v", units, err)
	}
	if units, err = parseUnits(""); err != nil || len(units) != 0 {
		t.Fatalf("empty: got %v, %v", units, err)
	}
	for _, bad := range []string{"code", "code=admin", "code=owner", "kode=read", "repo.=read"} {
		if _, err = parseUnits(bad); err == nil || err.Title != "Invalid unit" {
			t.Errorf("%q: got %+v", bad, err)
		}
	}
}

func TestPermissionFlag(t *testing.T) {
	var p PermissionFlag
	if err := p.Set("WRITE"); err != nil || p != "write" {
		t.Fatalf("got %q, %v", p, err)
	}
	for _, bad := range []string{"", "owner", "none"} {
		if err := p.Set(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestTeamAccess(t *testing.T) {
	if ta, err := teamAccess("admin", " "); err != nil || ta.Permission != gol.TeamPermissionAdmin || ta.Units != nil {
		t.Fatalf("permission: got %+v, %v", ta, err)
	}
	if ta, err := teamAccess("", "code=read"); err != nil || ta.Permission != "" || len(ta.Units) != 1 {
		t.Fatalf("units: got %+v, %v", ta, err)
	}
	if ta, err := teamAccess("", ""); err != nil || ta.Permission != "" || ta.Units != nil {
		t.Fatalf("neither: got %+v, %v", ta, err)
	}
	if _, err := teamAccess("read", "code=read"); err == nil || err.Title != "Invalid team access" {
		t.Fatalf("both: got %+v", err)
	}
}
