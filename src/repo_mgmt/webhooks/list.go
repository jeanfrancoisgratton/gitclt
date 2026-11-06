// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/list.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListHooks lists a target's webhooks
func ListHooks(t Target, displayoutput bool) ([]gol.Webhook, *cerr.CustomError) {
	ctx := context.Background()

	if t.kind == kindDefault {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: "Forgejo's API cannot list default webhooks: the web interface can (Site administration, Webhooks), and gitclt webhook default get takes an ID"}
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return nil, ce
	}
	lister, ok := opened.store.(hookLister)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation", Message: "the " + opened.noun + "s cannot be listed"}
	}
	list, e := shared.FetchAll(ctx, lister.List)
	if e != nil {
		return nil, forge.Error("Unable to fetch the "+opened.noun+" list"+opened.where, e)
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}

	if JSONoutput {
		return list, shared.PrintJSON(list, "webhook list")
	}
	if shared.QuietOutput {
		for _, h := range list {
			fmt.Println(h.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Println("There is no " + opened.noun + opened.where)
		return list, nil
	}
	renderHookTable(list)
	return list, nil
}

// GetHook shows a webhook
func GetHook(t Target, arg string, displayoutput bool) (*gol.Webhook, *cerr.CustomError) {
	ctx := context.Background()

	ids, ce := parseIDs([]string{arg})
	if ce != nil {
		return nil, ce
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return nil, ce
	}
	hook, e := opened.store.Get(ctx, ids[0])
	if e != nil {
		return nil, forge.Error("Unable to fetch "+opened.name(ids[0]), e)
	}

	if !displayoutput {
		return hook, nil
	}
	if JSONoutput {
		return hook, shared.PrintJSON(hook, "webhook")
	}
	renderHook(hook)
	return hook, nil
}
