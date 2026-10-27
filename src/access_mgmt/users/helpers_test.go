// gitclt
// Tests readPasswordFile, and holds what the other user tests share (fixtures, defaults)

package users

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitclt/shared"
	"gitclt/testutil"
)

const jfgUser = `{"id":4,"login":"jfg","full_name":"J.F. Gratton","email":"jfg@example.com","visibility":"limited",
	"is_admin":true,"active":true,"location":"Montreal","pronouns":"they/them","starred_repos_count":3,
	"created":"2026-01-02T03:04:05Z"}`

// Resets the package flags to their defaults for the duration of the test
func defaults(t *testing.T) {
	t.Helper()
	for _, b := range []*bool{&ExtendedInfo, &JSONoutput, &PurgeUser, &AssumeYes, &WipeDryRun, &CreateUserRestricted,
		&CreateUserSendNotification, &CreateUserAdmin, &shared.QuietOutput} {
		testutil.Set(t, b, false)
	}
	for _, s := range []*string{&CreateUserEmail, &CreateUserPasswdFile, &CreateUserFullName, &CreateUserDescription,
		&CreateUserWebsite, &CreateUserLocation, &EditUserEmail, &EditUserFullName, &EditUserDescription,
		&EditUserWebsite, &EditUserLocation, &EditUserPasswdFile} {
		testutil.Set(t, s, "")
	}
	testutil.Set(t, &CreateUserVisibility, "")
	testutil.Set(t, &EditUserVisibility, "")
	testutil.Set(t, &shared.DebugMode, true) // GetPassword needs a terminal otherwise
}

func writeFile(t *testing.T, content string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pw")
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil { // WriteFile's perm goes through the umask
		t.Fatal(err)
	}
	return path
}

func TestReadPasswordFile(t *testing.T) {
	// Only the trailing line ending is stripped
	if pw, err := readPasswordFile(writeFile(t, " pass word \r\n", 0600)); err != nil || pw != " pass word " {
		t.Fatalf("got %q, %v", pw, err)
	}

	for name, tc := range map[string]struct {
		path string
		want string
	}{
		"readable by others": {writeFile(t, "pw", 0644), "chmod 600"},
		"empty":              {writeFile(t, "\n", 0600), "is empty"},
		"directory":          {t.TempDir(), "not a regular file"},
		"missing":            {filepath.Join(t.TempDir(), "nope"), "no such file"},
	} {
		if _, err := readPasswordFile(tc.path); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", name, err, tc.want)
		}
	}
}
