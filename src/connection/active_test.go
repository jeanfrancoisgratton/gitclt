// gitclt
// Tests ResolveConnection: how the authentication method is picked, and URL checks

package conn_cfg

import (
	"testing"

	"gitclt/testutil"
)

func TestResolveConnection(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{
		"both":{"url":"https://both/","token":"tok","username":"me","password":"pw","servertype":"forgejo"},
		"basic":{"url":"https://basic","username":"me","password":"pw","servertype":"bitbucket"},
		"gh":{"url":"https://github.com","username":"me","password":"pw","servertype":"github"},
		"badurl":{"url":"not a url","token":"tok","servertype":"forgejo"}},
		"default_connection":"both"}`)

	// The token wins over username/password; the username is kept anyway, the password is not
	ac, err := ResolveConnection("")
	if err != nil {
		t.Fatal(err.Error())
	}
	want := ActiveConnection{Name: "both", ServerType: ServerTypeForgejo, URL: "https://both", Auth: AuthToken, Token: "tok", Username: "me"}
	if ac != want {
		t.Fatalf("got %+v, want %+v", ac, want)
	}

	ac, err = ResolveConnection("basic")
	if err != nil {
		t.Fatal(err.Error())
	}
	if ac.Auth != AuthBasic || ac.Password != "pw" || ac.Token != "" {
		t.Fatalf("got %+v", ac)
	}

	// GitHub has no password authentication on its REST API
	if _, err = ResolveConnection("gh"); err == nil || err.Title != "Invalid connection" {
		t.Fatalf("gh: got %+v", err)
	}
	if _, err = ResolveConnection("badurl"); err == nil || err.Title != "Invalid connection" {
		t.Fatalf("badurl: got %+v", err)
	}
	if _, err = ResolveConnection("nosuch"); err == nil {
		t.Fatal("nosuch: expected an error")
	}
}

func TestResolveConnection_BrokenFile(t *testing.T) {
	testutil.WriteRegistry(t, `{`)
	if _, err := ResolveConnection(""); err == nil {
		t.Fatal("expected an error")
	}
}
