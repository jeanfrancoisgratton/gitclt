// gitclt
// Tests the self-service commands (e-mail addresses, SSH keys, avatar, 2FA status) against fake forges

package users

import (
	"bytes"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const emailList = `[{"email":"jfg@example.com","primary":true,"verified":true},{"email":"Old@example.com","verified":false}]`

const sshKeyList = `[{"id":1,"title":"laptop","key":"ssh-ed25519 AAAA laptop","fingerprint":"SHA256:one"},
	{"id":2,"title":"ci","key":"ssh-ed25519 BBBB ci","fingerprint":"SHA256:two"},
	{"id":3,"title":"ci","key":"ssh-ed25519 CCCC ci","fingerprint":"SHA256:three"}]`

const validKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 jfg@laptop"

// Points the default connection to a fake server of this type
func otherForge(t *testing.T, serverType string, routes map[string]http.HandlerFunc) {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range routes {
		mux.HandleFunc(pattern, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	testutil.WriteRegistry(t, fmt.Sprintf(`{"auths":{"test":{"url":%q,"token":"abc","username":"acme","servertype":%q}},"default_connection":"test"}`, srv.URL, serverType))
}

// Records the addresses of a DELETE, whose JSON body holds them
func recordEmails(t *testing.T, got *[]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body struct {
			Emails []string `json:"emails"`
		}
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("body %s: %v", b, err)
		}
		*got = append(*got, body.Emails...)
		w.WriteHeader(http.StatusNoContent)
	}
}

func tempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func pngOf(t *testing.T, width, height int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// ---- 2FA status

func TestWhoAmI_TwoFactor(t *testing.T) {
	defaults(t)
	var err *cerr.CustomError

	// Forgejo's API never tells: this must not read as "disabled"
	whoamiServer(t)
	out := testutil.Run(t, func() { _, err = WhoAmI(true) })
	if err != nil || !strings.Contains(out, "2FA enabled:") || !strings.Contains(out, "not reported") {
		t.Fatalf("forgejo: %v\n%s", err, out)
	}

	for _, tc := range []struct {
		reply string
		want  string
	}{
		{`{"username":"jfg","has_2fa_enabled":true}`, "✅️"},
		{`{"username":"jfg","has_2fa_enabled":false}`, "❌️"},
		{`{"username":"jfg"}`, "not reported"},
	} {
		otherForge(t, "bitbucket", map[string]http.HandlerFunc{"GET /user": testutil.Reply(tc.reply)})
		out = testutil.Run(t, func() { _, err = WhoAmI(true) })
		line := ""
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "2FA enabled:") {
				line = l
			}
		}
		if err != nil || !strings.HasSuffix(strings.TrimSpace(line), tc.want) {
			t.Errorf("%s: %v, line %q\n%s", tc.reply, err, line, out)
		}
	}

	// Shown once with -x as well
	testutil.Set(t, &ExtendedInfo, true)
	out = testutil.Run(t, func() { _, err = WhoAmI(true) })
	if err != nil || strings.Count(out, "2FA enabled:") != 1 {
		t.Fatalf("extended: %v\n%s", err, out)
	}
}

// ---- e-mail addresses

func TestListEmails(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"GET /api/v1/user/emails": testutil.Reply(emailList)})
	var err *cerr.CustomError

	out := testutil.Run(t, func() { _, err = ListEmails(true) })
	if err != nil || !strings.Contains(out, "jfg@example.com") || !strings.Contains(out, "Old@example.com") || strings.Contains(out, "Visibility") {
		t.Fatalf("table: %v\n%s", err, out)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	out = testutil.Run(t, func() { _, err = ListEmails(true) })
	if err != nil || out != "jfg@example.com\nOld@example.com\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
	testutil.Set(t, &shared.QuietOutput, false)

	testutil.Set(t, &JSONoutput, true)
	out = testutil.Run(t, func() { _, err = ListEmails(true) })
	if err != nil || !strings.Contains(out, `"address"`) {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
}

func TestAddEmails(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/emails": testutil.Record(t, &bodies, http.StatusCreated,
		`[{"email":"jfg@example.com","primary":true,"verified":true},{"email":"New@example.com","verified":false}]`)})

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = AddEmails([]string{"new@example.com"}) })
	if err != nil || !strings.Contains(out, "Added e-mail address New@example.com (not verified yet)") {
		t.Fatalf("%v\n%s", err, out)
	}
	// Forgejo answers with the whole list: the library keeps what was added
	if strings.Contains(out, "jfg@example.com") {
		t.Fatalf("reported an address that was not added:\n%s", out)
	}
	if len(bodies) != 1 || fmt.Sprint(bodies[0]["emails"]) != "[new@example.com]" {
		t.Fatalf("sent %v", bodies)
	}
}

func TestDeleteEmails(t *testing.T) {
	defaults(t)
	var deleted []string
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/emails":    testutil.Reply(emailList),
		"DELETE /api/v1/user/emails": recordEmails(t, &deleted),
	})
	var err *cerr.CustomError

	// Refused before anything is removed, even if another address of the list is fine
	if err = DeleteEmails([]string{"old@example.com", "jfg@example.com"}); err == nil || !strings.Contains(err.Error(), "primary") {
		t.Fatalf("primary: %v", err)
	}
	if err = DeleteEmails([]string{"old@example.com", "nobody@example.com"}); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("unknown: %+v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted %v", deleted)
	}

	out := testutil.Run(t, func() { err = DeleteEmails([]string{"old@example.com"}) }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || len(deleted) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, deleted, out)
	}

	// The address is sent the way the server spells it
	out = testutil.Run(t, func() { err = DeleteEmails([]string{"old@example.com"}) }, "y")
	if err != nil || !strings.Contains(out, "Deleted e-mail address Old@example.com") || !slices.Equal(deleted, []string{"Old@example.com"}) {
		t.Fatalf("yes: %v, %v\n%s", err, deleted, out)
	}
}

func TestEmails_BitbucketReadOnly(t *testing.T) {
	defaults(t)
	otherForge(t, "bitbucket", nil)
	for name, err := range map[string]*cerr.CustomError{
		"add":    AddEmails([]string{"a@b"}),
		"delete": DeleteEmails([]string{"a@b"}),
	} {
		if err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: %+v", name, err)
		}
	}
}

// ---- SSH keys

func sshKeyServer(t *testing.T, deleted *[]string) {
	t.Helper()
	testutil.Forgejo(t, map[string]http.HandlerFunc{
		"GET /api/v1/user/keys": testutil.Reply(sshKeyList),
		"DELETE /api/v1/user/keys/{id}": func(w http.ResponseWriter, r *http.Request) {
			*deleted = append(*deleted, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		},
	})
}

func TestListSSHKeys(t *testing.T) {
	defaults(t)
	var deleted []string
	sshKeyServer(t, &deleted)
	var err *cerr.CustomError

	out := testutil.Run(t, func() { _, err = ListSSHKeys(true) })
	if err != nil || !strings.Contains(out, "laptop") || !strings.Contains(out, "SHA256:two") {
		t.Fatalf("table: %v\n%s", err, out)
	}

	testutil.Set(t, &shared.QuietOutput, true)
	out = testutil.Run(t, func() { _, err = ListSSHKeys(true) })
	if err != nil || out != "laptop\nci\nci\n" {
		t.Fatalf("quiet: %v\n%q", err, out)
	}
}

func TestGetSSHKey(t *testing.T) {
	defaults(t)
	var deleted []string
	sshKeyServer(t, &deleted)
	var err *cerr.CustomError

	out := testutil.Run(t, func() { _, err = GetSSHKey("laptop", true) })
	if err != nil || !strings.Contains(out, "SHA256:one") || !strings.Contains(out, "ssh-ed25519 AAAA laptop") {
		t.Fatalf("%v\n%s", err, out)
	}

	if _, err = GetSSHKey("nope", false); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("unknown: %+v", err)
	}
	// Two keys share this title: it names neither
	if _, err = GetSSHKey("ci", false); err == nil || !strings.Contains(err.Error(), "--id (2, 3)") {
		t.Fatalf("ambiguous: %+v", err)
	}

	testutil.Set(t, &SSHKeyID, 3)
	k, err := GetSSHKey("", false)
	if err != nil || k.Fingerprint != "SHA256:three" {
		t.Fatalf("by ID: %+v, %+v", k, err)
	}
	testutil.Set(t, &SSHKeyID, 99)
	if _, err = GetSSHKey("", false); err == nil || !strings.Contains(err.Error(), "ID 99") {
		t.Fatalf("unknown ID: %+v", err)
	}
}

func TestAddSSHKey(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/keys": testutil.Record(t, &bodies, http.StatusCreated,
		`{"id":9,"title":"laptop","key":"`+validKey+`","fingerprint":"SHA256:nine"}`)})

	path := tempFile(t, "id.pub", []byte(validKey+"\n"))
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = AddSSHKey("laptop", path) })
	if err != nil || !strings.Contains(out, "Added SSH key laptop") {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(bodies) != 1 || bodies[0]["key"] != validKey || bodies[0]["title"] != "laptop" {
		t.Fatalf("sent %v", bodies)
	}
}

// A private key must never reach the server, nor end up in an error message
func TestAddSSHKey_RefusesAnythingElse(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/keys": testutil.Record(t, &bodies, http.StatusCreated, `{}`)})

	const secret = "b3BlbnNzaC1rZXktdjEAAAAA"
	for name, content := range map[string]string{
		"openssh private key": "-----BEGIN OPENSSH PRIVATE KEY-----\n" + secret + "\n-----END OPENSSH PRIVATE KEY-----\n",
		"putty private key":   "PuTTY-User-Key-File-3: ssh-ed25519\nEncryption: none\nPrivate-Lines: 1\n" + secret + "\n",
		"two keys":            validKey + "\n" + validKey + "\n",
		"type only":           "ssh-ed25519\n",
		"empty":               "",
	} {
		err := AddSSHKey("laptop", tempFile(t, "id", []byte(content)))
		if err == nil || err.Title != "Invalid SSH key" || strings.Contains(err.Error(), secret) {
			t.Errorf("%s: %+v", name, err)
		}
	}
	if err := AddSSHKey("laptop", filepath.Join(t.TempDir(), "absent")); err == nil || err.Title != "Unable to read the SSH key" {
		t.Errorf("absent: %+v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("sent %v", bodies)
	}
}

func TestDeleteSSHKey(t *testing.T) {
	defaults(t)
	var deleted []string
	sshKeyServer(t, &deleted)
	var err *cerr.CustomError

	if err = DeleteSSHKey("ci"); err == nil || !strings.Contains(err.Error(), "--id") || len(deleted) != 0 {
		t.Fatalf("ambiguous: %+v, %v", err, deleted)
	}

	out := testutil.Run(t, func() { err = DeleteSSHKey("laptop") }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || len(deleted) != 0 {
		t.Fatalf("no: %v, %v\n%s", err, deleted, out)
	}
	out = testutil.Run(t, func() { err = DeleteSSHKey("laptop") }, "y")
	if err != nil || !strings.Contains(out, "Deleted SSH key laptop") || !slices.Equal(deleted, []string{"1"}) {
		t.Fatalf("yes: %v, %v\n%s", err, deleted, out)
	}

	// --id settles it, and --yes needs no answer
	testutil.Set(t, &SSHKeyID, 3)
	testutil.Set(t, &AssumeYes, true)
	deleted = nil
	if err = DeleteSSHKey(""); err != nil || !slices.Equal(deleted, []string{"3"}) {
		t.Fatalf("by ID: %v, %v", err, deleted)
	}
}

// ---- avatar

func TestSetAvatar(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/avatar": testutil.Record(t, &bodies, http.StatusNoContent, ``)})

	img := pngOf(t, 16, 16)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = SetAvatar(tempFile(t, "me.png", img)) })
	if err != nil || !strings.Contains(out, "Avatar set") {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(bodies) != 1 || bodies[0]["image"] != base64.StdEncoding.EncodeToString(img) {
		t.Fatalf("sent %v", bodies)
	}
}

// Forgejo would answer a bare HTTP 500 to these: they must not reach it
func TestSetAvatar_RefusedLocally(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/avatar": testutil.Record(t, &bodies, http.StatusNoContent, ``)})

	for name, content := range map[string][]byte{
		"empty":     {},
		"not image": []byte("this is not an image"),
		"too wide":  pngOf(t, 4097, 1),
		"too tall":  pngOf(t, 1, 4097),
		"too large": append(pngOf(t, 1, 1), make([]byte, 10<<20)...),
	} {
		if err := SetAvatar(tempFile(t, "me.png", content)); err == nil || err.Title != "Invalid avatar" {
			t.Errorf("%s: %+v", name, err)
		}
	}
	if err := SetAvatar(filepath.Join(t.TempDir(), "absent.png")); err == nil || err.Title != "Unable to read the avatar" {
		t.Errorf("absent: %+v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("sent %d bodies", len(bodies))
	}
}

// Forgejo's bare HTTP 500 gets an explanation
func TestSetAvatar_ServerRefusal(t *testing.T) {
	defaults(t)
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/user/avatar": testutil.Status(http.StatusInternalServerError)})
	err := SetAvatar(tempFile(t, "me.png", pngOf(t, 1, 1)))
	if err == nil || err.Code != http.StatusInternalServerError || !strings.Contains(err.Error(), "an image it cannot read") {
		t.Fatalf("got %+v", err)
	}
}

// The largest allowed dimensions are not refused
func TestSetAvatar_AtTheLimit(t *testing.T) {
	if _, err := ReadAvatar(tempFile(t, "me.png", pngOf(t, maxAvatarDimension, 1))); err != nil {
		t.Fatalf("got %+v", err)
	}
}

func TestDeleteAvatar(t *testing.T) {
	defaults(t)
	calls := 0
	testutil.Forgejo(t, map[string]http.HandlerFunc{"DELETE /api/v1/user/avatar": func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}})
	var err *cerr.CustomError

	out := testutil.Run(t, func() { err = DeleteAvatar() }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || calls != 0 {
		t.Fatalf("no: %v, %d\n%s", err, calls, out)
	}
	testutil.Set(t, &AssumeYes, true)
	out = testutil.Run(t, func() { err = DeleteAvatar() })
	if err != nil || !strings.Contains(out, "Avatar deleted") || calls != 1 {
		t.Fatalf("yes: %v, %d\n%s", err, calls, out)
	}
}

func TestAvatar_ForgejoOnly(t *testing.T) {
	defaults(t)
	for _, serverType := range []string{"github", "bitbucket"} {
		otherForge(t, serverType, nil)
		path := tempFile(t, "me.png", pngOf(t, 1, 1))
		if err := SetAvatar(path); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: set: %+v", serverType, err)
		}
		// Refused before the question: nothing to answer
		if err := DeleteAvatar(); err == nil || err.Title != "Unsupported operation" {
			t.Errorf("%s: delete: %+v", serverType, err)
		}
	}
}
