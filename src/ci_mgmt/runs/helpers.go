// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/helpers.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	"github.com/jeanfrancoisgratton/helperFunctions/v5/pager"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"golang.org/x/term"
)

// target is an opened repository, with the interfaces to its runs and artifacts
type target struct {
	c           *forge.Client
	runs        gol.WorkflowRuns
	owner, repo string
}

func (t target) where() string { return t.owner + "/" + t.repo }

// open connects, refuses a forge without workflow runs, decides the repository (see forge.Client.Target) and makes sure it
// exists, so that a 404 further down is about the run, job or artifact
func open(ctx context.Context, repo string) (target, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return target{}, ce
	}
	runs, ok := c.Client.(gol.WorkflowRuns)
	if !ok {
		return target{}, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: workflow runs are only available on Forgejo and GitHub", c.Connection.Name)}
	}
	owner, repo, ce := c.Target(ctx, OwnerOrg, repo)
	if ce != nil {
		return target{}, ce
	}
	if _, e := c.Repos().Get(ctx, owner, repo); e != nil {
		if gol.IsNotFound(e) {
			return target{}, &cerr.CustomError{Title: "No such repository", Code: http.StatusNotFound, Message: fmt.Sprintf("%s/%s does not exist", owner, repo)}
		}
		return target{}, forge.Error(fmt.Sprintf("Unable to get the repository %s/%s", owner, repo), e)
	}
	return target{c: c, runs: runs, owner: owner, repo: repo}, nil
}

// artifacts returns the interface to the repository's artifacts (Forgejo 16 and later, GitHub)
func (t target) artifacts() (gol.WorkflowArtifacts, *cerr.CustomError) {
	a, ok := t.c.Client.(gol.WorkflowArtifacts)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: workflow artifacts are only available on Forgejo and GitHub", t.c.Connection.Name)}
	}
	return a, nil
}

// runError names what a 404 was about; the library's ErrNotSupported says which Forgejo version is needed
func runError(title, what string, e error) *cerr.CustomError {
	switch {
	case gol.IsNotFound(e):
		return &cerr.CustomError{Title: "No such " + what, Code: http.StatusNotFound, Message: title + ": " + e.Error()}
	case errors.Is(e, gol.ErrNotSupported):
		return &cerr.CustomError{Title: "Unsupported operation", Message: title + ": " + e.Error()}
	}
	return forge.Error(title, e)
}

// parseIDs reads run, job or artifact IDs, dropping repeated ones
func parseIDs(args []string, what string) ([]int64, *cerr.CustomError) {
	var out []int64
	for _, a := range args {
		id, e := strconv.ParseInt(strings.TrimPrefix(a, "#"), 10, 64)
		if e != nil || id <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid " + what + " ID", Message: a + " is not " + article(what) + " ID"}
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

func article(what string) string {
	if strings.ContainsAny(what[:1], "aeiou") {
		return "an " + what
	}
	return "a " + what
}

// save writes what r holds to path, or to the standard output for "-"; an existing file is only replaced with --force,
// and what was written of a file is removed if the copy fails
func save(r io.ReadCloser, path string) (int64, *cerr.CustomError) {
	defer r.Close()
	if path == "-" {
		n, e := io.Copy(os.Stdout, r)
		if e != nil {
			return n, &cerr.CustomError{Title: "Unable to write the file", Message: e.Error()}
		}
		return n, nil
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if Overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, e := os.OpenFile(path, flags, 0o644)
	if errors.Is(e, os.ErrExist) {
		return 0, &cerr.CustomError{Title: "File already exists", Message: path + " already exists: pass --force to replace it, or -O to name another file"}
	}
	if e != nil {
		return 0, &cerr.CustomError{Title: "Unable to write the file", Message: e.Error()}
	}
	n, e := io.Copy(f, r)
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		_ = os.Remove(path)
		return n, &cerr.CustomError{Title: "Unable to write the file", Message: e.Error()}
	}
	return n, nil
}

// size makes a byte count readable
func size(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}

// interactive tells if the output can be paged: the pager needs a terminal on both sides, for its keys and its screen
func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// pageLog shows a log through the pager, with title as its banner; Windows line endings are dropped, as the pager
// draws its lines itself
func pageLog(r io.ReadCloser, title string) *cerr.CustomError {
	defer r.Close()
	b, e := io.ReadAll(r)
	if e != nil {
		return &cerr.CustomError{Title: "Unable to read the log", Message: e.Error()}
	}
	text := strings.TrimRight(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if text == "" {
		fmt.Println("The log is empty")
		return nil
	}
	lines := append([]string{hftx.Blue(title)}, strings.Split(text, "\n")...)
	if e = pager.Page(lines, 1, 0); e != nil {
		return &cerr.CustomError{Title: "Unable to show the log", Message: e.Error()}
	}
	return nil
}
