// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/ci_mgmt/runs/artifacts.go
// Original timestamp : 2026.11.07 10:00:00

package runs

import (
	"context"
	"fmt"
	"gitclt/shared"
	"path/filepath"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListArtifacts lists a repository's artifacts, or with --run one run's (Forgejo 16 and later, GitHub)
func ListArtifacts(repo string, displayoutput bool) ([]gol.WorkflowArtifact, *cerr.CustomError) {
	ctx := context.Background()
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	a, ce := t.artifacts()
	if ce != nil {
		return nil, ce
	}
	opts := gol.ListWorkflowArtifactsOptions{Name: ArtifactName}
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.WorkflowArtifact], error) {
		opts.ListOptions = o
		if ArtifactRun != 0 {
			return a.ListForRun(ctx, t.owner, t.repo, ArtifactRun, opts)
		}
		return a.List(ctx, t.owner, t.repo, opts)
	})
	if e != nil {
		return nil, runError("Unable to list the artifacts of "+t.where(), "run", e)
	}

	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "artifact list")
	}
	if shared.QuietOutput {
		for _, x := range list {
			fmt.Println(x.ID)
		}
		return list, nil
	}
	if len(list) == 0 {
		fmt.Printf("No artifact of %s matches\n", t.where())
		return list, nil
	}
	shared.Page(renderArtifactTable(list), shared.TableHeader)
	return list, nil
}

// GetArtifact shows an artifact
func GetArtifact(repo, arg string, displayoutput bool) (*gol.WorkflowArtifact, *cerr.CustomError) {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "artifact")
	if ce != nil {
		return nil, ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	a, ce := t.artifacts()
	if ce != nil {
		return nil, ce
	}
	x, e := a.Get(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return nil, runError(fmt.Sprintf("Unable to get artifact %d of %s", ids[0], t.where()), "artifact", e)
	}
	if !displayoutput {
		return x, nil
	}
	if JSONoutput {
		return x, shared.PrintJSON(x, "artifact")
	}
	renderArtifact(x)
	return x, nil
}

// DownloadArtifact saves an artifact, as the ZIP archive the forge makes; by default as <name>.zip, here
func DownloadArtifact(repo, arg string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs([]string{arg}, "artifact")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	a, ce := t.artifacts()
	if ce != nil {
		return ce
	}
	x, e := a.Get(ctx, t.owner, t.repo, ids[0])
	if e != nil {
		return runError(fmt.Sprintf("Unable to get artifact %d of %s", ids[0], t.where()), "artifact", e)
	}
	if x.Expired {
		return &cerr.CustomError{Title: "Artifact expired", Message: fmt.Sprintf("artifact %d (%s) of %s has expired, and can no longer be downloaded", x.ID, x.Name, t.where())}
	}
	path := OutputFile
	if path == "" {
		// The name comes from the server: only its last element is kept, so that it cannot point elsewhere
		path = filepath.Base(filepath.Clean("/"+x.Name)) + ".zip"
	}
	r, e := a.Download(ctx, t.owner, t.repo, x.ID)
	if e != nil {
		return runError(fmt.Sprintf("Unable to download artifact %d of %s", x.ID, t.where()), "artifact", e)
	}
	n, ce := save(r, path)
	if ce != nil {
		return ce
	}
	if path != "-" && !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Saved artifact %s to %s (%s)", hftx.Brown(x.Name), hftx.Brown(path), size(n))))
	}
	return nil
}

// DeleteArtifacts deletes artifacts, after confirmation; every one is checked before asking
// The run stops at the first failure; the artifacts deleted before it stay deleted
func DeleteArtifacts(repo string, args []string) *cerr.CustomError {
	ctx := context.Background()
	ids, ce := parseIDs(args, "artifact")
	if ce != nil {
		return ce
	}
	t, ce := open(ctx, repo)
	if ce != nil {
		return ce
	}
	a, ce := t.artifacts()
	if ce != nil {
		return ce
	}
	shown := make([]string, len(ids))
	for i, id := range ids {
		x, e := a.Get(ctx, t.owner, t.repo, id)
		if e != nil {
			return runError(fmt.Sprintf("Unable to get artifact %d of %s", id, t.where()), "artifact", e)
		}
		shown[i] = fmt.Sprintf("%d (%s)", id, x.Name)
	}
	if !confirm(fmt.Sprintf("Delete artifact(s) %s of %s ? [y/N]: ", strings.Join(shown, ", "), t.where())) {
		fmt.Println("Aborted, no artifact is deleted")
		return nil
	}
	for _, id := range ids {
		if e := a.Delete(ctx, t.owner, t.repo, id); e != nil {
			return runError(fmt.Sprintf("Unable to delete artifact %d of %s", id, t.where()), "artifact", e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign(fmt.Sprintf("Deleted artifact %d of %s", id, t.where())))
		}
	}
	return nil
}
