// gitclt
// Tests user create: the create request, and the follow-up edit for the fields the create endpoint lacks

package user

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"gitclt/testutil"
	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// A fake Forgejo recording the create (POST) and follow-up edit (PATCH) bodies
func createServer(t *testing.T, createStatus, editStatus int) (creates, edits *[]map[string]any) {
	t.Helper()
	creates, edits = &[]map[string]any{}, &[]map[string]any{}
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"POST /api/v1/admin/users":      record(t, creates, createStatus, jfgUser),
		"PATCH /api/v1/admin/users/jfg": record(t, edits, editStatus, jfgUser),
	})
	return creates, edits
}

func TestCreateUser_WithFollowUpEdit(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateUserEmail, "jfg@example.com")
	testutil.Set(t, &CreateUserFullname, "J.F. Gratton")
	testutil.Set(t, &CreateUserPasswdFile, writeFile(t, "s3cr3t\n", 0600))
	testutil.Set(t, &CreateUserVisibility, "limited")
	testutil.Set(t, &CreateUserRestrictedUser, true)
	testutil.Set(t, &CreateUserLocation, "Montreal")
	testutil.Set(t, &CreateUserAdmin, true)
	creates, edits := createServer(t, http.StatusCreated, http.StatusOK)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateUser("jfg", nil) })
	if err != nil {
		t.Fatal(err.Error())
	}

	c := (*creates)[0]
	if c["username"] != "jfg" || c["email"] != "jfg@example.com" || c["password"] != "s3cr3t" || c["full_name"] != "J.F. Gratton" ||
		c["visibility"] != "limited" || c["restricted"] != true {
		t.Fatalf("create sent %v", c)
	}
	// nil: the server default
	if _, ok := c["must_change_password"]; ok {
		t.Fatalf("must_change_password was sent: %v", c)
	}
	if len(*edits) != 1 || !slices.Equal(keys((*edits)[0]), []string{"admin", "location"}) {
		t.Fatalf("edit sent %v", *edits)
	}
	if !strings.Contains(out, "Created user jfg") || !strings.Contains(out, "Updated user jfg") {
		t.Fatalf("output = %q", out)
	}
}

// No profile field nor --admin: no follow-up edit; the password is prompted for
func TestCreateUser_PromptedPassword(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateUserEmail, "jfg@example.com")
	creates, edits := createServer(t, http.StatusCreated, http.StatusOK)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = CreateUser("jfg", gol.Ptr(true)) }, "typed-in")
	if err != nil {
		t.Fatal(err.Error())
	}
	if c := (*creates)[0]; c["password"] != "typed-in" || c["must_change_password"] != true {
		t.Fatalf("create sent %v", c)
	}
	if len(*edits) != 0 {
		t.Fatalf("unexpected edit: %v", *edits)
	}
	if !strings.Contains(out, "password is temporary") {
		t.Fatalf("output = %q", out)
	}
}

func TestCreateUser_Errors(t *testing.T) {
	defaults(t)
	testutil.Set(t, &CreateUserEmail, "jfg@example.com")

	// The password file is read before connecting
	testutil.WriteRegistry(t, "")
	testutil.Set(t, &CreateUserPasswdFile, writeFile(t, "pw", 0644))
	if err := CreateUser("jfg", nil); err == nil || err.Title != "Unable to read password file" {
		t.Fatalf("password file: got %+v", err)
	}
	testutil.Set(t, &CreateUserPasswdFile, writeFile(t, "pw", 0600))
	if err := CreateUser("jfg", nil); err == nil || err.Title != "No connection specified" {
		t.Fatalf("no connection: got %+v", err)
	}

	createServer(t, http.StatusUnprocessableEntity, http.StatusOK)
	if err := CreateUser("jfg", nil); err == nil || err.Title != "Unable to create the user" {
		t.Fatalf("create: got %+v", err)
	}

	// The user exists, but the follow-up edit failed: the message says not to re-run create
	testutil.Set(t, &CreateUserWebsite, "https://example.com")
	createServer(t, http.StatusCreated, http.StatusForbidden)
	var err *cerr.CustomError
	testutil.Run(t, func() { err = CreateUser("jfg", nil) })
	if err == nil || !strings.Contains(err.Title, "do not re-run create") {
		t.Fatalf("follow-up edit: got %+v", err)
	}
}
