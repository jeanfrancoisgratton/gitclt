// gitclt
// Tests Validate, and the conn add / conn rm commands (prompts answered through testutil.Run)

package connections

import (
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    ConnectionStruct
		ok   bool
	}{
		{"token", ConnectionStruct{URL: "u", Token: "t", ServerType: ServerTypeForgejo}, true},
		{"userpass", ConnectionStruct{URL: "u", Username: "me", Password: "pw", ServerType: ServerTypeForgejo}, true},
		{"no server type", ConnectionStruct{URL: "u", Token: "t"}, false},
		{"no URL", ConnectionStruct{Token: "t", ServerType: ServerTypeForgejo}, false},
		{"username without password", ConnectionStruct{URL: "u", Username: "me", ServerType: ServerTypeForgejo}, false},
		{"nothing to authenticate with", ConnectionStruct{URL: "u", ServerType: ServerTypeForgejo}, false},
	} {
		if err := tc.c.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: got %v", tc.name, err)
		}
	}
}

// Answers the five conn add prompts: server type, URL, token, username, password
func addConnection(t *testing.T, name string, answers ...string) (string, *cerr.CustomError) {
	t.Helper()
	testutil.Set(t, &shared.DebugMode, true) // GetPassword needs a terminal otherwise
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = AddConnection(name) }, answers...)
	return out, err
}

func TestAddConnection(t *testing.T) {
	testutil.WriteRegistry(t, "")

	// An empty server type means the first supported one; the first connection becomes the default
	out, err := addConnection(t, "first", "", "https://first", "tok", "", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	if !strings.Contains(out, "Connection first added") {
		t.Fatalf("output = %q", out)
	}

	if _, err = addConnection(t, "second", "github", "https://github.com", "ghp_x", "me", ""); err != nil {
		t.Fatal(err.Error())
	}

	cf, _ := LoadConnectionsFile()
	if cf.DefaultConnection != "first" {
		t.Fatalf("default = %q", cf.DefaultConnection)
	}
	if c := cf.Auths["first"]; c.ServerType != ServerTypeForgejo || c.URL != "https://first" || c.Token != "tok" {
		t.Fatalf("first = %+v", c)
	}
	if c := cf.Auths["second"]; c.ServerType != ServerTypeGithub || c.Username != "me" {
		t.Fatalf("second = %+v", c)
	}
}

func TestAddConnection_Errors(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{"x":{"url":"https://x","token":"t","servertype":"forgejo"}}}`)

	// Refused before any prompt
	if _, err := addConnection(t, "x"); err == nil || err.Title != "Connection already exists" {
		t.Fatalf("existing: got %+v", err)
	}
	// Refused right after the first prompt
	if _, err := addConnection(t, "y", "gitlab"); err == nil || err.Title != "Invalid server type" {
		t.Fatalf("server type: got %+v", err)
	}
	// No way to authenticate
	if _, err := addConnection(t, "y", "forgejo", "https://y", "", "me", ""); err == nil || err.Title != "Invalid connection" {
		t.Fatalf("validate: got %+v", err)
	}

	cf, _ := LoadConnectionsFile()
	if _, ok := cf.Auths["y"]; ok {
		t.Fatal("an invalid connection was saved")
	}
}

func TestRemoveConnection(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{
		"a":{"url":"https://a","token":"t","servertype":"forgejo"},
		"b":{"url":"https://b","token":"t","servertype":"forgejo"}},"default_connection":"a"}`)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = RemoveConnection("b") })
	if err != nil || !strings.Contains(out, "Connection b removed") || strings.Contains(out, "default") {
		t.Fatalf("b: %q, %v", out, err)
	}

	// Removing the default connection leaves no default
	out = testutil.Run(t, func() { err = RemoveConnection("a") })
	if err != nil || !strings.Contains(out, "no default connection is now set") {
		t.Fatalf("a: %q, %v", out, err)
	}
	cf, _ := LoadConnectionsFile()
	if len(cf.Auths) != 0 || cf.DefaultConnection != "" {
		t.Fatalf("got %+v", cf)
	}

	if err = RemoveConnection("a"); err == nil || err.Title != "Unknown connection" {
		t.Fatalf("got %+v", err)
	}
}

func TestAddRemove_BrokenFile(t *testing.T) {
	testutil.WriteRegistry(t, `{`)
	if err := AddConnection("x"); err == nil {
		t.Error("add: expected an error")
	}
	if err := RemoveConnection("x"); err == nil {
		t.Error("rm: expected an error")
	}
}
