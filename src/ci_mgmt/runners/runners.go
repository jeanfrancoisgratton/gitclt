// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runners/runners.go
// Original timestamp : 2026.11.11 18:00:00

package runners

import (
	"context"
	"fmt"
	"gitclt/shared"
	"strings"
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListRunners lists the runners of the scope; --inherited adds those it can use from above it (Forgejo)
func ListRunners(repo string, displayoutput bool) ([]gol.Runner, *cerr.CustomError) {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Runner], error) {
		return t.runners.ListRunners(ctx, t.scope, gol.ListRunnersOptions{ListOptions: o, Inherited: Inherited})
	})
	if e != nil {
		return nil, runnerError("Unable to list the runners of "+t.where, e)
	}
	if list == nil {
		list = []gol.Runner{}
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "runner list")
	}
	if shared.QuietOutput {
		for _, r := range list {
			fmt.Println(r.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s has no runner\n", t.where)
		return list, nil
	}
	renderRunnerTable(list)
	return list, nil
}

// GetRunner shows one runner of the scope
func GetRunner(repo, arg string, displayoutput bool) (*gol.Runner, *cerr.CustomError) {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg})
	if ce != nil {
		return nil, ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	r, e := t.runners.GetRunner(ctx, t.scope, ids[0])
	if e != nil {
		return nil, runnerError(fmt.Sprintf("Unable to get runner %d of %s", ids[0], t.where), e)
	}
	if !displayoutput {
		return r, nil
	}
	if JSONoutput {
		return r, shared.PrintJSON(r, "runner")
	}
	renderRunner(r)
	return r, nil
}

// RegisterRunner creates a runner's record, and prints what its program needs to connect: on Forgejo, the ID, UUID
// and token of its .runner file; on GitHub, the just-in-time configuration of a single-use runner
// The secret is never stored by the forge in a readable form: it is printed this once, and -q prints it alone
func RegisterRunner(repo string) *cerr.CustomError {
	ctx := context.Background()
	if strings.TrimSpace(RegisterName) == "" {
		return &cerr.CustomError{Title: "Missing name", Message: "--name is required"}
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	opts := gol.RegisterRunnerOptions{Name: RegisterName, Description: RegisterDescription, Ephemeral: RegisterEphemeral, Labels: RegisterLabels}
	rr, e := t.runners.RegisterRunner(ctx, t.scope, opts)
	if e != nil {
		return runnerError("Unable to register a runner for "+t.where, e)
	}

	secret := rr.Token
	if rr.JITConfig != "" {
		secret = rr.JITConfig
	}
	if shared.QuietOutput {
		fmt.Println(secret)
		return nil
	}
	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Registered runner %s (ID %d) for %s", hftx.Brown(RegisterName), rr.ID, t.where)))
	if rr.JITConfig != "" {
		fmt.Println("Start the runner with: ./run.sh --jitconfig <the configuration below>; it takes one job, then goes away")
		fmt.Printf("\n%s\n\n", rr.JITConfig)
	} else {
		fmt.Println("These go in the runner's .runner file:")
		fmt.Printf("  ID:    %d\n  UUID:  %s\n  Token: %s\n", rr.ID, rr.UUID, rr.Token)
	}
	fmt.Println(hftx.WarningSign("This secret is shown only this once: it cannot be read back"))
	return nil
}

// RunnerToken prints a registration token, for a runner's program to register with; GitHub's expires within the hour
// -q prints the token alone
func RunnerToken(repo string) *cerr.CustomError {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	tk, e := t.runners.RunnerRegistrationToken(ctx, t.scope)
	if e != nil {
		return runnerError("Unable to get a runner registration token for "+t.where, e)
	}
	if shared.QuietOutput {
		fmt.Println(tk.Token)
		return nil
	}
	fmt.Printf("Runner registration token for %s:\n\n%s\n\n", t.where, tk.Token)
	if !tk.ExpiresAt.IsZero() {
		fmt.Println(hftx.WarningSign("It expires on " + tk.ExpiresAt.Local().Format("2006.01.02 15:04") +
			fmt.Sprintf(" (in %s)", time.Until(tk.ExpiresAt).Round(time.Minute))))
	}
	return nil
}

// DeleteRunners deletes runners of the scope, after confirmation; every runner is checked before asking
// The run stops at the first failure; the runners deleted before it stay deleted
func DeleteRunners(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs(args)
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	shown := make([]string, len(ids))
	for i, id := range ids {
		r, e := t.runners.GetRunner(ctx, t.scope, id)
		if e != nil {
			return runnerError(fmt.Sprintf("Unable to get runner %d of %s", id, t.where), e)
		}
		shown[i] = fmt.Sprintf("%d (%s, %s)", id, r.Name, state(*r))
	}
	if !confirm(fmt.Sprintf("Delete runner(s) %s of %s ? [y/N]: ", strings.Join(shown, ", "), t.where)) {
		fmt.Println("Aborted, no runner is deleted")
		return nil
	}
	for _, id := range ids {
		if e := t.runners.DeleteRunner(ctx, t.scope, id); e != nil {
			return runnerError(fmt.Sprintf("Unable to delete runner %d of %s", id, t.where), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted runner %d of %s", id, t.where)))
		}
	}
	return nil
}
