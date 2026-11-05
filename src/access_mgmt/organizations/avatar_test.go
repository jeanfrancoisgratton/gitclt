// gitclt
// Tests the org avatar commands against a fake Forgejo; the image checks themselves are tested with user avatar

package organizations

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func avatarFile(t *testing.T) (string, []byte) {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "acme.png")
	if err := os.WriteFile(path, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path, b.Bytes()
}

func TestSetOrgAvatar(t *testing.T) {
	defaults(t)
	var bodies []map[string]any
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/orgs/acme/avatar": testutil.Record(t, &bodies, http.StatusNoContent, ``)})

	path, img := avatarFile(t)
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = SetOrgAvatar("acme", path) })
	if err != nil || !strings.Contains(out, "Avatar of organization acme set") {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(bodies) != 1 || bodies[0]["image"] != base64.StdEncoding.EncodeToString(img) {
		t.Fatalf("sent %v", bodies)
	}

	// Refused locally: nothing more reaches the server
	bad := filepath.Join(t.TempDir(), "acme.txt")
	if e := os.WriteFile(bad, []byte("this is not an image"), 0600); e != nil {
		t.Fatal(e)
	}
	if err = SetOrgAvatar("acme", bad); err == nil || err.Title != "Invalid avatar" || len(bodies) != 1 {
		t.Fatalf("not an image: %+v, %d requests", err, len(bodies))
	}
}

func TestSetOrgAvatar_ServerRefusals(t *testing.T) {
	defaults(t)
	path, _ := avatarFile(t)

	// Not an owner of the organization
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/orgs/acme/avatar": testutil.Status(http.StatusForbidden)})
	if err := SetOrgAvatar("acme", path); err == nil || err.Code != http.StatusForbidden {
		t.Fatalf("got %+v", err)
	}
	// No such organization
	if err := SetOrgAvatar("nope", path); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("got %+v", err)
	}
	// Forgejo's bare HTTP 500 gets an explanation
	testutil.Forgejo(t, map[string]http.HandlerFunc{"POST /api/v1/orgs/acme/avatar": testutil.Status(http.StatusInternalServerError)})
	if err := SetOrgAvatar("acme", path); err == nil || !strings.Contains(err.Error(), "an image it cannot read") {
		t.Fatalf("got %+v", err)
	}
}

func TestDeleteOrgAvatar(t *testing.T) {
	defaults(t)
	calls := 0
	testutil.Forgejo(t, map[string]http.HandlerFunc{"DELETE /api/v1/orgs/acme/avatar": func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}})
	var err *cerr.CustomError

	out := testutil.Run(t, func() { err = DeleteOrgAvatar("acme") }, "n")
	if err != nil || !strings.Contains(out, "Aborted") || calls != 0 {
		t.Fatalf("no: %v, %d\n%s", err, calls, out)
	}
	out = testutil.Run(t, func() { err = DeleteOrgAvatar("acme") }, "y")
	if err != nil || !strings.Contains(out, "Avatar of organization acme deleted") || calls != 1 {
		t.Fatalf("yes: %v, %d\n%s", err, calls, out)
	}
	testutil.Set(t, &AssumeYes, true)
	if err = DeleteOrgAvatar("nope"); err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("no such org: %+v", err)
	}
}

// Refused before the question: nothing to answer
func TestOrgAvatar_ForgejoOnly(t *testing.T) {
	defaults(t)
	path, _ := avatarFile(t)
	testutil.WriteRegistry(t, githubRegistry)
	if err := SetOrgAvatar("acme", path); err == nil || err.Title != "Unsupported operation" {
		t.Errorf("set: %+v", err)
	}
	if err := DeleteOrgAvatar("acme"); err == nil || err.Title != "Unsupported operation" {
		t.Errorf("delete: %+v", err)
	}
}
