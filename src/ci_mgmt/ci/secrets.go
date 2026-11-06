// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/ci/secrets.go
// Original timestamp : 2026.11.06 18:00:00

package ci

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListSecrets lists the names of the secrets of a scope: their values can never be read back
func ListSecrets(repo string, displayoutput bool) ([]gol.Secret, *cerr.CustomError) {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Secret], error) {
		return t.ci.Secrets().List(ctx, t.scope, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to list the secrets of "+t.where, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "secret list")
	}
	if shared.QuietOutput {
		for _, s := range list {
			fmt.Println(s.Name)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Println("There is no secret on " + t.where)
		return list, nil
	}
	renderSecretTable(list)
	return list, nil
}

// SetSecret creates a secret, or replaces its value
// The value is never an argument, where any local user could read it: it comes from --from-file, --stdin or a prompt
func SetSecret(repo, name string) *cerr.CustomError {
	ctx := context.Background()
	value, ok, ce := flagValue(true)
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	// Asked for last: nobody should type a secret only to learn that the repository was wrong
	if !ok {
		if value, ce = promptValue(name); ce != nil {
			return ce
		}
	}
	if e := t.ci.Secrets().Set(ctx, t.scope, name, value); e != nil {
		return forge.Error(fmt.Sprintf("Unable to set the secret %s on %s", name, t.where), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Set secret %s on %s", hftx.Brown(name), t.where)))
	}
	return nil
}

// DeleteSecrets deletes secrets, after confirmation; every name is checked before asking, where the forge can list them
// The run stops at the first failure; the secrets deleted before it stay deleted
func DeleteSecrets(repo string, names []string) *cerr.CustomError {
	ctx := context.Background()
	names = unique(names)
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}

	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Secret], error) {
		return t.ci.Secrets().List(ctx, t.scope, o)
	})
	switch {
	// Forgejo before 17 cannot list a user's own secrets: the server answers for those
	case errors.Is(e, gol.ErrNotSupported):
	case e != nil:
		return forge.Error("Unable to list the secrets of "+t.where, e)
	default:
		existing := make([]string, len(list))
		for i, s := range list {
			existing[i] = s.Name
		}
		if missing := missingName(names, existing); missing != "" {
			return &cerr.CustomError{Title: "No such secret", Code: http.StatusNotFound, Message: fmt.Sprintf("There is no secret %s on %s", missing, t.where)}
		}
	}

	if !confirm(fmt.Sprintf("Delete secret(s) %s of %s ? Their values cannot be read back [y/N]: ", strings.Join(names, ", "), t.where)) {
		fmt.Println("Aborted, no secret is deleted")
		return nil
	}
	for _, name := range names {
		if e := t.ci.Secrets().Delete(ctx, t.scope, name); e != nil {
			return forge.Error("Unable to delete the secret "+name, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted secret " + hftx.Brown(name)))
		}
	}
	return nil
}
