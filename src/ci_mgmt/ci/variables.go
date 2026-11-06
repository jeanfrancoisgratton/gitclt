// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/ci/variables.go
// Original timestamp : 2026.11.06 18:00:00

package ci

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func listVariables(ctx context.Context, t target) ([]gol.Variable, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Variable], error) {
		return t.ci.Variables().List(ctx, t.scope, o)
	})
	if e != nil {
		return nil, forge.Error("Unable to list the variables of "+t.where, e)
	}
	return list, nil
}

// ListVariables lists the variables of a scope, with their values
func ListVariables(repo string, displayoutput bool) ([]gol.Variable, *cerr.CustomError) {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, ce := listVariables(ctx, t)
	if ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "variable list")
	}
	if shared.QuietOutput {
		for _, v := range list {
			fmt.Println(v.Name)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Println("There is no variable on " + t.where)
		return list, nil
	}
	renderVariableTable(list)
	return list, nil
}

// GetVariable shows a variable; -q outputs only its value, for scripts
func GetVariable(repo, name string, displayoutput bool) (*gol.Variable, *cerr.CustomError) {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	v, e := t.ci.Variables().Get(ctx, t.scope, name)
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such variable", Code: http.StatusNotFound, Message: fmt.Sprintf("There is no variable %s on %s", name, t.where)}
		}
		return nil, forge.Error("Unable to get the variable "+name, e)
	}

	if !displayoutput {
		return v, nil
	}
	if JSONoutput {
		return v, shared.PrintJSON(v, "variable")
	}
	if shared.QuietOutput {
		fmt.Println(v.Value)
		return v, nil
	}
	renderVariable(v)
	return v, nil
}

// SetVariable creates a variable, or changes its value; value is empty when it comes from --from-file or --stdin
func SetVariable(repo, name, value string) *cerr.CustomError {
	ctx := context.Background()
	fromFlag, ok, ce := flagValue(false)
	if ce != nil {
		return ce
	}
	if ok {
		value = fromFlag
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	if e := t.ci.Variables().Set(ctx, t.scope, name, value); e != nil {
		return forge.Error(fmt.Sprintf("Unable to set the variable %s on %s", name, t.where), e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Set variable %s on %s", hftx.Brown(name), t.where)))
	}
	return nil
}

// DeleteVariables deletes variables, after confirmation; every name is checked before asking
// The run stops at the first failure; the variables deleted before it stay deleted
func DeleteVariables(repo string, names []string) *cerr.CustomError {
	ctx := context.Background()
	names = unique(names)
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	list, ce := listVariables(ctx, t)
	if ce != nil {
		return ce
	}
	existing := make([]string, len(list))
	for i, v := range list {
		existing[i] = v.Name
	}
	if missing := missingName(names, existing); missing != "" {
		return &cerr.CustomError{Title: "No such variable", Code: http.StatusNotFound, Message: fmt.Sprintf("There is no variable %s on %s", missing, t.where)}
	}

	if !confirm(fmt.Sprintf("Delete variable(s) %s of %s ? [y/N]: ", strings.Join(names, ", "), t.where)) {
		fmt.Println("Aborted, no variable is deleted")
		return nil
	}
	for _, name := range names {
		if e := t.ci.Variables().Delete(ctx, t.scope, name); e != nil {
			return forge.Error("Unable to delete the variable "+name, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted variable " + hftx.Brown(name)))
		}
	}
	return nil
}
