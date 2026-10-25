// gitclt
// Tests the connections file: missing / broken files, secrets encrypted on disk, hand-edited cleartext secrets

package conn_cfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/testutil"
)

// Writes registry as the connections file, and loads it back
func load(t *testing.T, registry string) ConnectionsFileStruct {
	t.Helper()
	testutil.WriteRegistry(t, registry)
	cf, err := LoadConnectionsFile()
	if err != nil {
		t.Fatal(err.Error())
	}
	return cf
}

func TestLoad_MissingFileIsAnEmptyRegistry(t *testing.T) {
	cf := load(t, "")
	if cf.Auths == nil || len(cf.Auths) != 0 || cf.DefaultConnection != "" {
		t.Fatalf("got %+v", cf)
	}
}

func TestLoad_NullAuths(t *testing.T) {
	if cf := load(t, `{"auths":null}`); cf.Auths == nil {
		t.Fatal("Auths should never be nil, so callers can add to it")
	}
}

func TestLoad_Errors(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":`)
	if _, err := LoadConnectionsFile(); err == nil || err.Title != "Error parsing the connections file" {
		t.Fatalf("got %+v", err)
	}

	// The path exists, but is not a readable file
	path := testutil.WriteRegistry(t, "")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConnectionsFile(); err == nil || err.Title != "Error reading the connections file" {
		t.Fatalf("got %+v", err)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	path := testutil.WriteRegistry(t, "")
	cf := ConnectionsFileStruct{DefaultConnection: "b", Auths: map[string]ConnectionStruct{
		"b": {URL: "https://b", Token: "s3cr3t-token", ServerType: ServerTypeForgejo},
		"a": {URL: "https://a", Username: "me", Password: "s3cr3t-pw", ServerType: ServerTypeBitBucket},
	}}
	if err := cf.SaveConnectionsFile(); err != nil {
		t.Fatal(err.Error())
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "s3cr3t") {
		t.Fatalf("secrets are stored in clear:\n%s", raw)
	}
	// Map keys are sorted: a comes before b, whatever the map order
	if strings.Index(string(raw), `"a"`) > strings.Index(string(raw), `"b"`) {
		t.Fatalf("connections are not sorted:\n%s", raw)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %04o", fi.Mode().Perm())
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("the temporary file was left behind")
	}

	got, cerr := LoadConnectionsFile()
	if cerr != nil {
		t.Fatal(cerr.Error())
	}
	if got.DefaultConnection != "b" || got.Auths["b"] != cf.Auths["b"] || got.Auths["a"] != cf.Auths["a"] {
		t.Fatalf("got %+v", got)
	}
}

// A user may hand-edit the file with cleartext secrets: they must come back unchanged, not as garbage
func TestLoad_CleartextSecrets(t *testing.T) {
	cf := load(t, `{"auths":{"x":{"url":"https://x","token":"0123456789abcdef0123456789abcdef","password":"not base64!","servertype":"forgejo"}}}`)
	if c := cf.Auths["x"]; c.Token != "0123456789abcdef0123456789abcdef" || c.Password != "not base64!" {
		t.Fatalf("got %+v", c)
	}
}

func TestSafeDecodeString(t *testing.T) {
	for in, want := range map[string]string{"": "", "   ": "", "plain text": "plain text", encodeString("secret"): "secret"} {
		if got := safeDecodeString(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
	if encodeString("") != "" {
		t.Error("an empty secret must stay empty")
	}
}

func TestSave_Error(t *testing.T) {
	testutil.WriteRegistry(t, "")
	// The config directory is gone: the temp file cannot be created
	if err := os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".config")); err != nil {
		t.Fatal(err)
	}
	cf := ConnectionsFileStruct{Auths: map[string]ConnectionStruct{}}
	if err := cf.SaveConnectionsFile(); err == nil || err.Title != "Error writing the connections file" {
		t.Fatalf("got %+v", err)
	}
}
