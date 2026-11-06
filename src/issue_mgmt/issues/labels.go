// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/labels.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"regexp"
	"slices"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

var hexColor = regexp.MustCompile(`^[0-9a-f]{6}$`)

// parseColor takes a color with or without its #, in any case; the library wants six lower case digits without it
func parseColor(s string) (string, *cerr.CustomError) {
	color := strings.ToLower(strings.TrimPrefix(s, "#"))
	if !hexColor.MatchString(color) {
		return "", &cerr.CustomError{Title: "Invalid color", Message: s + " is not a color: six hexadecimal digits, such as d73a4a"}
	}
	return color, nil
}

func allLabels(ctx context.Context, c *forge.Client, owner, repo string) ([]gol.Label, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Label], error) {
		return c.Labels().List(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the labels of %s/%s", owner, repo), e)
	}
	return list, nil
}

// getLabel finds a label by its name
// Forgejo lets several labels share a name, while the API addresses them by name: such a label is refused, as it could hit the wrong one
func getLabel(ctx context.Context, c *forge.Client, owner, repo, name string) (*gol.Label, *cerr.CustomError) {
	list, ce := allLabels(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}
	var found *gol.Label
	count := 0
	for i := range list {
		if list[i].Name == name {
			found = &list[i]
			count++
		}
	}
	switch count {
	case 0:
		return nil, &cerr.CustomError{Title: "No such label", Code: http.StatusNotFound, Message: fmt.Sprintf("%s/%s has no label %s", owner, repo, name)}
	case 1:
		return found, nil
	}
	return nil, &cerr.CustomError{Title: "Ambiguous label",
		Message: fmt.Sprintf("%d labels of %s/%s are named %s, and the API can only tell labels by their name: keep one in the web interface", count, owner, repo, name)}
}

// labelNameFree refuses a name already taken: GitHub does too, but Forgejo would make a second label with it
func labelNameFree(ctx context.Context, c *forge.Client, owner, repo, name string) *cerr.CustomError {
	list, ce := allLabels(ctx, c, owner, repo)
	if ce != nil {
		return ce
	}
	if slices.ContainsFunc(list, func(l gol.Label) bool { return l.Name == name }) {
		return &cerr.CustomError{Title: "Label already exists", Message: fmt.Sprintf("%s/%s already has a label %s", owner, repo, name)}
	}
	return nil
}

// ListLabelsOf lists the labels of owner/repo
func ListLabelsOf(repo string, displayoutput bool) ([]gol.Label, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	list, ce := allLabels(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "label list")
	}
	if shared.QuietOutput {
		for _, l := range list {
			fmt.Println(l.Name)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("%s/%s has no label\n", owner, repo)
		return list, nil
	}
	renderLabelTable(list)
	return list, nil
}

// GetLabel shows a label of owner/repo
func GetLabel(repo, name string, displayoutput bool) (*gol.Label, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	label, ce := getLabel(ctx, c, owner, repo, name)
	if ce != nil {
		return nil, ce
	}
	if !displayoutput {
		return label, nil
	}
	if JSONoutput {
		return label, shared.PrintJSON(label, "label")
	}
	renderLabel(label)
	return label, nil
}

// CreateLabel creates a label in owner/repo
func CreateLabel(repo, name string) *cerr.CustomError {
	ctx := context.Background()
	color, ce := parseColor(LabelColor)
	if ce != nil {
		return ce
	}
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if ce = labelNameFree(ctx, c, owner, repo, name); ce != nil {
		return ce
	}
	label, e := c.Labels().Create(ctx, owner, repo, gol.Label{Name: name, Color: color, Description: LabelDescription})
	if e != nil {
		return forge.Error(fmt.Sprintf("Unable to create the label %s in %s/%s", name, owner, repo), e)
	}
	if JSONoutput {
		return shared.PrintJSON(label, "label")
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created label %s in %s/%s", hftx.Brown(label.Name), owner, repo)))
	}
	return nil
}

var labelEditFlags = []string{"name", "color", "description"}

// EditLabel only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditLabel(repo, name string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	if !slices.ContainsFunc(labelEditFlags, changed) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt issue label edit --help"}
	}
	var opts gol.EditLabelOptions
	if changed("name") {
		opts.Name = gol.Ptr(LabelName)
	}
	if changed("color") {
		color, ce := parseColor(LabelColor)
		if ce != nil {
			return ce
		}
		opts.Color = gol.Ptr(color)
	}
	if changed("description") {
		opts.Description = gol.Ptr(LabelDescription)
	}

	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	if _, ce = getLabel(ctx, c, owner, repo, name); ce != nil {
		return ce
	}
	if changed("name") && LabelName != name {
		if ce = labelNameFree(ctx, c, owner, repo, LabelName); ce != nil {
			return ce
		}
	}
	if _, e := c.Labels().Edit(ctx, owner, repo, name, opts); e != nil {
		return forge.Error("Unable to edit the label "+name, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated label " + hftx.Brown(name)))
	}
	return nil
}

// DeleteLabels deletes labels, after confirmation; every name is checked before asking
// The run stops at the first failure; the labels deleted before it stay deleted
func DeleteLabels(repo string, names []string) *cerr.CustomError {
	ctx := context.Background()
	names = unique(names)
	c, owner, repo, ce := connect(ctx, repo)
	if ce != nil {
		return ce
	}
	for _, name := range names {
		if _, ce = getLabel(ctx, c, owner, repo, name); ce != nil {
			return ce
		}
	}
	if !confirm(fmt.Sprintf("Delete label(s) %s of %s/%s ? They leave every issue carrying them [y/N]: ", strings.Join(names, ", "), owner, repo)) {
		fmt.Println("Aborted, no label is deleted")
		return nil
	}
	for _, name := range names {
		if e := c.Labels().Delete(ctx, owner, repo, name); e != nil {
			return forge.Error("Unable to delete the label "+name, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted label " + hftx.Brown(name)))
		}
	}
	return nil
}
