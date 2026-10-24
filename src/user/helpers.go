// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/user/helpers.go
// Original timestamp : 2026.10.24 13:43:57

package user

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// readPasswordFile reads a password from path, refusing files that are
// not regular files or that group/others can access.
func readPasswordFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", path)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		return "", fmt.Errorf("%s has permissions %04o; must not be accessible by group or others (chmod 600)", path, perm)
	}

	b, err := io.ReadAll(io.LimitReader(f, 4096)) // a password file has no business being larger
	if err != nil {
		return "", err
	}
	pw := strings.TrimRight(string(b), "\r\n")
	if pw == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return pw, nil
}
