// gitclt
// Tests user edit: only the flags that were passed are sent, and the visibility quirk of Forgejo

package users

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Stands in for cmd.Flags().Changed: only the named flags were passed
func passed(names ...string) func(string) bool {
	return func(name string) bool { return slices.Contains(names, name) }
}

func keys(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
}

func TestEditUser_OnlyPassedFlagsAreSent(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditUserEmail, "new@example.com")
	testutil.Set(t, &EditUserLocation, "Quebec") // set, but not passed: must not be sent
	testutil.Set(t, &EditUserAdmin, false)
	testutil.Set(t, &EditUserMaxRepos, 5)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/admin/users/jfg": testutil.Record(t, &bodies, 200, jfgUser)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditUser("jfg", passed("email", "admin", "max-repos")) })
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(bodies) != 1 {
		t.Fatalf("%d requests", len(bodies))
	}
	b := bodies[0]
	if got := keys(b); !slices.Equal(got, []string{"admin", "email", "max_repo_creation"}) {
		t.Fatalf("sent %v", got)
	}
	// --admin=false must be sent as false, not dropped
	if b["email"] != "new@example.com" || b["admin"] != false || b["max_repo_creation"] != 5.0 {
		t.Fatalf("sent %v", b)
	}
	if !strings.Contains(out, "Updated user jfg") {
		t.Fatalf("output = %q", out)
	}
}

func TestEditOptionsFromFlags(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditUserFullName, "")
	testutil.Set(t, &EditUserVisibility, "private")

	// An empty string clears the field: it is sent, as ""
	euo, err := editOptionsFromFlags(passed("fullname", "visibility", "active", "prohibit-login", "restricted", "allow-create-org",
		"must-change-passwd", "description", "website"))
	if err != nil {
		t.Fatal(err.Error())
	}
	if euo.FullName == nil || *euo.FullName != "" || euo.Visibility == nil || *euo.Visibility != "private" {
		t.Fatalf("got %+v", euo)
	}
	for name, p := range map[string]*bool{"active": euo.Active, "prohibit-login": euo.ProhibitLogin, "restricted": euo.Restricted,
		"allow-create-org": euo.AllowCreateOrganization, "must-change-passwd": euo.MustChangePassword} {
		if p == nil {
			t.Errorf("%s was not set", name)
		}
	}
	if euo.Email != nil || euo.Admin != nil || euo.Password != nil || euo.MaxRepoCreation != nil {
		t.Fatalf("flags not passed were set: %+v", euo)
	}
}

func TestEditOptionsFromFlags_Passwords(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditUserPasswdFile, writeFile(t, "from-file\n", 0600))
	if euo, err := editOptionsFromFlags(passed("password-file")); err != nil || *euo.Password != "from-file" {
		t.Fatalf("file: got %+v, %v", euo, err)
	}

	testutil.Set(t, &EditUserPasswdFile, writeFile(t, "pw", 0644))
	if _, err := editOptionsFromFlags(passed("password-file")); err == nil || err.Title != "Unable to read password file" {
		t.Fatalf("bad file: got %+v", err)
	}

	var euo gol.EditUserOptions
	out := testutil.Run(t, func() { euo, _ = editOptionsFromFlags(passed("passwd")) }, "typed-in")
	if euo.Password == nil || *euo.Password != "typed-in" || !strings.Contains(out, "new password") {
		t.Fatalf("prompt: got %+v, %q", euo, out)
	}
}

func TestEditUser_Errors(t *testing.T) {
	defaults(t)

	// Checked before connecting: no registry needed
	testutil.WriteRegistry(t, "")
	if err := EditUser("jfg", passed()); err == nil || err.Title != "Nothing to edit" {
		t.Fatalf("nothing: got %+v", err)
	}
	testutil.Set(t, &EditUserMaxRepos, -2)
	if err := EditUser("jfg", passed("max-repos")); err == nil || err.Title != "Invalid value" {
		t.Fatalf("max-repos: got %+v", err)
	}
	testutil.Set(t, &EditUserMaxRepos, -1)
	if err := EditUser("jfg", passed("max-repos")); err == nil || err.Title != "No connection specified" {
		t.Fatalf("no connection: got %+v", err)
	}

	testutil.Forgejo(t, nil)
	if err := EditUser("nobody", passed("max-repos")); err == nil || err.Code != http.StatusNotFound || err.Title != "Unable to edit the user" {
		t.Fatalf("404: got %+v", err)
	}
}

// Forgejo may keep the old visibility while applying everything else: a warning, not an error
func TestEditUser_VisibilityIgnoredByServer(t *testing.T) {
	defaults(t)
	testutil.Set(t, &EditUserVisibility, "public")
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"PATCH /api/v1/admin/users/jfg": testutil.Record(t, &bodies, 200, jfgUser)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditUser("jfg", passed("visibility")) })
	if err != nil {
		t.Fatalf("got %+v", err)
	}
	if !strings.Contains(out, "User jfg was updated, but") || strings.Contains(out, "Updated user") {
		t.Fatalf("output = %q", out)
	}
}
