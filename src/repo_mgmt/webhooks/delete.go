// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/delete.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// DeleteHooks deletes webhooks; every one of them is checked before asking
// The run stops at the first failure; the webhooks deleted before it stay deleted
func DeleteHooks(t Target, args []string) *cerr.CustomError {
	ctx := context.Background()

	ids, ce := parseIDs(args)
	if ce != nil {
		return ce
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return ce
	}

	described := make([]string, 0, len(ids))
	for _, id := range ids {
		hook, e := opened.store.Get(ctx, id)
		if e != nil {
			return forge.Error("Unable to fetch "+opened.name(id), e)
		}
		described = append(described, fmt.Sprintf("%d (%s)", id, hook.URL))
	}

	if !confirm(fmt.Sprintf("Delete %s(s) %s%s ? [y/N]: ", opened.noun, strings.Join(described, ", "), opened.where)) {
		fmt.Println("Aborted, no webhook is deleted")
		return nil
	}

	for _, id := range ids {
		if e := opened.store.Delete(ctx, id); e != nil {
			return forge.Error("Unable to delete "+opened.name(id), e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted " + hftx.Brown(opened.name(id))))
		}
	}
	return nil
}

// PingHook has the forge send a test delivery; only repository webhooks can, and organization ones on GitHub
func PingHook(t Target, arg string) *cerr.CustomError {
	ctx := context.Background()

	ids, ce := parseIDs([]string{arg})
	if ce != nil {
		return ce
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return ce
	}
	pinger, ok := opened.store.(hookPinger)
	if !ok {
		return &cerr.CustomError{Title: "Unsupported operation", Message: "the " + opened.noun + "s have no test delivery"}
	}
	if e := pinger.Ping(ctx, ids[0]); e != nil {
		return forge.Error("Unable to test "+opened.name(ids[0]), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Sent a test delivery to " + hftx.Brown(opened.name(ids[0]))))
	}
	return nil
}
