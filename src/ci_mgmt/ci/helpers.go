// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/ci/helpers.go
// Original timestamp : 2026.11.06 18:00:00

package ci

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"golang.org/x/term"
)

// Far above what the forges take; it is there so that a wrong file is refused rather than read whole
const maxValueSize = 1 << 20

// target is an opened scope: where the secrets and variables live, and how to name it in messages
type target struct {
	ci    gol.CI
	scope gol.CIScope
	where string
}

// HasOwnScope tells whether a flag says where to look: there is then no repository among the arguments
func HasOwnScope() bool { return ScopeOrg != "" || ScopeUser }

// open connects, and decides the scope from the flags; by default, the repository (see forge.Client.Target)
func open(ctx context.Context, repo string) (target, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return target{}, ce
	}
	ci, ce := c.CI()
	if ce != nil {
		return target{}, ce
	}
	switch {
	case ScopeUser:
		return target{ci, gol.UserCI(), "your account"}, nil
	case ScopeOrg != "":
		return target{ci, gol.OrgCI(ScopeOrg), ScopeOrg}, nil
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return target{}, ce
	}
	// A 404 further down must be about the secret or variable, not the repository
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return target{}, &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound, Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return target{}, forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	if ScopeEnv != "" {
		return target{ci, gol.EnvCI(owner, repo, ScopeEnv), fmt.Sprintf("%s/%s (environment %s)", owner, repo, ScopeEnv)}, nil
	}
	return target{ci, gol.RepoCI(owner, repo), owner + "/" + repo}, nil
}

// readValueFile reads a value from a file; a secret's file must be one that only its owner can access
func readValueFile(path string, secret bool) (string, *cerr.CustomError) {
	fail := func(msg string) (string, *cerr.CustomError) {
		return "", &cerr.CustomError{Title: "Unable to read the value", Message: msg}
	}
	f, e := os.Open(path)
	if e != nil {
		return fail(e.Error())
	}
	defer f.Close()
	fi, e := f.Stat()
	if e != nil {
		return fail(e.Error())
	}
	if !fi.Mode().IsRegular() {
		return fail(path + " is not a regular file")
	}
	if perm := fi.Mode().Perm(); secret && perm&0o077 != 0 {
		return fail(fmt.Sprintf("%s has permissions %04o; must not be accessible by group or others (chmod 600)", path, perm))
	}
	return readValue(f, path)
}

// readValue reads a whole value, without its trailing line ending, and refuses rather than cut it short
func readValue(r io.Reader, from string) (string, *cerr.CustomError) {
	b, e := io.ReadAll(io.LimitReader(r, maxValueSize+1))
	if e != nil {
		return "", &cerr.CustomError{Title: "Unable to read the value", Message: e.Error()}
	}
	if len(b) > maxValueSize {
		return "", &cerr.CustomError{Title: "Unable to read the value", Message: fmt.Sprintf("%s holds more than %d bytes", from, maxValueSize)}
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "" {
		return "", &cerr.CustomError{Title: "Unable to read the value", Message: from + " is empty"}
	}
	return value, nil
}

// flagValue reads a value from --from-file or --stdin; ok tells whether either flag was passed
func flagValue(secret bool) (value string, ok bool, ce *cerr.CustomError) {
	switch {
	case ValueFile != "":
		value, ce = readValueFile(ValueFile, secret)
		return value, true, ce
	case ValueStdin:
		value, ce = readValue(os.Stdin, "the standard input")
		return value, true, ce
	}
	return "", false, nil
}

// promptValue asks for a value, without showing it; GetPassword panics without a terminal, unless in debug mode
func promptValue(name string) (string, *cerr.CustomError) {
	if !shared.DebugMode && !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", &cerr.CustomError{Title: "Unable to read the value",
			Message: "it cannot be asked for without a terminal; use --from-file or --stdin"}
	}
	value := hf.GetPassword("Enter the value of "+name+": ", shared.DebugMode)
	if value == "" {
		return "", &cerr.CustomError{Title: "Unable to read the value", Message: "nothing was entered"}
	}
	return value, nil
}

// missingName finds the first name that is not among those listed; the forges compare names ignoring case, Forgejo and
// GitHub storing them upper-cased
func missingName(names, existing []string) string {
	for _, name := range names {
		if !slices.ContainsFunc(existing, func(e string) bool { return strings.EqualFold(e, name) }) {
			return name
		}
	}
	return ""
}

// unique drops repeated arguments, keeping the first occurrence's order
func unique(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}
