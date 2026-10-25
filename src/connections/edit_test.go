// gitclt
// Tests conn edit (ENTER keeps, '-' clears a secret) and conn default

package connections

import (
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const editRegistry = `{"auths":{
	"x":{"url":"https://x","token":"x-secret-tok","username":"me","password":"x-secret-pw","servertype":"forgejo"},
	"y":{"url":"https://y","token":"tok","servertype":"forgejo"}},"default_connection":"x"}`

// Answers the five conn edit prompts: server type, URL, token, username, password
func editConnection(t *testing.T, name string, answers ...string) (string, *cerr.CustomError) {
	t.Helper()
	testutil.Set(t, &shared.DebugMode, true)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = EditConnection(name) }, answers...)
	return out, err
}

func TestEditConnection_EnterKeepsEverything(t *testing.T) {
	testutil.WriteRegistry(t, editRegistry)
	before, _ := LoadConnectionsFile()

	out, err := editConnection(t, "x", "", "", "", "", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	// The current values are offered, except for the secrets
	if !strings.Contains(out, "[forgejo]") || !strings.Contains(out, "[https://x]") || strings.Contains(out, "secret") {
		t.Fatalf("output = %q", out)
	}
	after, _ := LoadConnectionsFile()
	if after.Auths["x"] != before.Auths["x"] || len(after.Auths) != 2 {
		t.Fatalf("got %+v", after.Auths)
	}
}

func TestEditConnection_ChangeAndClear(t *testing.T) {
	testutil.WriteRegistry(t, editRegistry)

	if _, err := editConnection(t, "x", "github", "https://github.com", "newtok", "", "-"); err != nil {
		t.Fatal(err.Error())
	}
	cf, _ := LoadConnectionsFile()
	want := ConnectionStruct{URL: "https://github.com", Token: "newtok", Username: "me", ServerType: ServerTypeGithub}
	if cf.Auths["x"] != want {
		t.Fatalf("got %+v, want %+v", cf.Auths["x"], want)
	}
}

func TestEditConnection_Errors(t *testing.T) {
	testutil.WriteRegistry(t, editRegistry)

	if _, err := editConnection(t, "nosuch"); err == nil || err.Title != "Unknown connection" {
		t.Fatalf("unknown: got %+v", err)
	}
	if _, err := editConnection(t, "y", "gitlab"); err == nil || err.Title != "Invalid server type" {
		t.Fatalf("server type: got %+v", err)
	}
	// Clearing the only credential: refused, and the file is untouched
	if _, err := editConnection(t, "y", "", "", "-", "", ""); err == nil || err.Title != "Invalid connection" {
		t.Fatalf("validate: got %+v", err)
	}
	if cf, _ := LoadConnectionsFile(); cf.Auths["y"].Token != "tok" {
		t.Fatal("an invalid edit was saved")
	}
}

func TestDefaultConnection(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{"x":{"url":"https://x","token":"t","servertype":"forgejo"}}}`)

	var err *cerr.CustomError
	if out := testutil.Run(t, func() { err = ShowDefaultConnection() }); err != nil || !strings.Contains(out, "No default connection is set") {
		t.Fatalf("show: %q, %v", out, err)
	}
	if err = SetDefaultConnection("nosuch"); err == nil || err.Title != "Unknown connection" {
		t.Fatalf("set nosuch: got %+v", err)
	}
	if out := testutil.Run(t, func() { err = SetDefaultConnection("x") }); err != nil || !strings.Contains(out, "Default connection set to x") {
		t.Fatalf("set: %q, %v", out, err)
	}
	if out := testutil.Run(t, func() { err = ShowDefaultConnection() }); err != nil || !strings.Contains(out, "Default connection: x") {
		t.Fatalf("show: %q, %v", out, err)
	}
}

func TestEditDefault_BrokenFile(t *testing.T) {
	testutil.WriteRegistry(t, `{`)
	if err := EditConnection("x"); err == nil {
		t.Error("edit: expected an error")
	}
	if err := ShowDefaultConnection(); err == nil {
		t.Error("show default: expected an error")
	}
	if err := SetDefaultConnection("x"); err == nil {
		t.Error("set default: expected an error")
	}
}
