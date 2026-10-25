// gitclt
// Tests the server type names, parsing, and their (un)marshalling as JSON strings

package conn_cfg

import (
	"encoding/json/v2"
	"slices"
	"testing"
)

func TestServerType_String(t *testing.T) {
	for st, want := range map[ServerType]string{ServerTypeUnknown: "", ServerTypeForgejo: "forgejo",
		ServerTypeGithub: "github", ServerTypeBitBucket: "bitbucket", -1: "", serverTypeCount: ""} {
		if got := st.String(); got != want {
			t.Errorf("%d: got %q, want %q", st, got, want)
		}
	}
}

// The first one is the default offered by conn add
func TestSupportedServerTypes(t *testing.T) {
	if got := SupportedServerTypes(); !slices.Equal(got, []string{"forgejo", "github", "bitbucket"}) {
		t.Fatalf("got %v", got)
	}
}

func TestParseServerType(t *testing.T) {
	for s, want := range map[string]ServerType{"forgejo": ServerTypeForgejo, "  GitHub ": ServerTypeGithub,
		"BITBUCKET": ServerTypeBitBucket} {
		if got, err := ParseServerType(s); err != nil || got != want {
			t.Errorf("%q: got %v, %v", s, got, err)
		}
	}
	for _, s := range []string{"", "gitlab"} {
		if got, err := ParseServerType(s); err == nil || got != ServerTypeUnknown {
			t.Errorf("%q: expected an error, got %v", s, got)
		}
	}
}

// Stored by name; an unknown name loads as ServerTypeUnknown instead of failing the whole file
func TestServerType_JSON(t *testing.T) {
	b, err := json.Marshal(ConnectionStruct{URL: "u", ServerType: ServerTypeGithub})
	if err != nil || string(b) != `{"url":"u","servertype":"github"}` {
		t.Fatalf("got %s, %v", b, err)
	}

	var c ConnectionStruct
	if err := json.Unmarshal([]byte(`{"url":"u","servertype":"Forgejo"}`), &c); err != nil || c.ServerType != ServerTypeForgejo {
		t.Fatalf("got %v, %v", c.ServerType, err)
	}
	if err := json.Unmarshal([]byte(`{"url":"u","servertype":"gitlab"}`), &c); err != nil || c.ServerType != ServerTypeUnknown {
		t.Fatalf("got %v, %v", c.ServerType, err)
	}
}
