// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/users/helpers.go
// Original timestamp : 2026.10.24 13:43:57

package users

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"io"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// Connects, and returns the interface managing the connection user's own account
func connectAccount(ctx context.Context) (*forge.Client, gol.Account, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return nil, nil, ce
	}
	acct, ce := c.Account()
	if ce != nil {
		return nil, nil, ce
	}
	return c, acct, nil
}

// Asks before an operation that cannot be undone, unless --yes was passed
func confirmed(prompt string) bool {
	if AssumeYes {
		return true
	}
	answer := hf.GetStringValFromPrompt(hftx.WarningSign(prompt))
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes"
}

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
